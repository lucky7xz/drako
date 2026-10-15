package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/cli"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/multiplex"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/ui"
)

// Run wires everything together. It keeps the program running so that after a command
// finishes we jump back into the TUI without losing state or the screen layout.
// It returns the process exit code; main is the only caller of os.Exit, so
// deferred cleanups here always run before the process ends.

func Run() int {
	glassrootMode := false
	isTuiMode := false

	// CLI handling
	// =======================================
	// Check for TUI-specific flags (Glassroot)
	// If present, we short-circuit the CLI handler entirely.
	for _, arg := range os.Args {
		if arg == "--glassroot" {
			isTuiMode = true
			glassrootMode = true
			break
		}
	}

	// A child cannot change its parent's directory; a shell wrapper reads this
	// file and cds. The caller owns the file, so two drakos never share one.
	if cwdFile := cwdFileArg(os.Args); cwdFile != "" {
		isTuiMode = true
		defer writeCwdFile(cwdFile)
	}

	// 1. If NOT in TUI mode, try to handle as a CLI command (e.g. "drako summon", "drako purge")
	if !isTuiMode {
		if handled, code := cli.HandleCLI(os.Args); handled {
			return code
		}
	}

	// 2. If we are here, we are launching the TUI.
	// =======================================

	// Proceed with TUI mode
	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "could not create config dir: %v\n", err)
		return 1
	}
	// =======================================
	// Logging setup

	// Rotate if > 1MB
	logPath := paths.LogFile(configDir)
	core.RotateLogIfNeeded(logPath, paths.LogArchive(configDir), 1024*1024)

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		// NOTE: cli.go treats this as a non-fatal warning and continues;
		// reconcile (warn vs exit) in a follow-up.
		fmt.Fprintf(os.Stderr, "could not open log file: %v\n", err)
		return 1
	}
	defer f.Close()
	log.SetOutput(f)

	// Each lap throws the model away so a launched command gets a clean
	// terminal, so where the cursor was has to be handed forward explicitly.
	// The zero value restores nothing, which is what the first lap wants.
	var session ui.Session

	// One watcher for the process; between laps InitialModel re-reads the config.
	var current atomic.Pointer[tea.Program]
	watching := false

	// Start of TUI Loop
	for {

		// Start the TUI program (Model/View/Update is now in internal/ui)
		// We initialize with the *current* directory which might have changed
		// from manual Chdir or from internal logic

		initial := ui.InitialModel(glassrootMode, session)
		if initial.Quitting {
			// e.g. glassroot with a broken profile: fail before the TUI starts
			return initial.ExitCode
		}

		// Watch only after the first InitialModel: on a first launch it writes
		// the bootstrap files, and drako's own writes are no reason to reload.
		if !watching {
			watching = true
			if stop, err := ui.WatchConfig(configDir, func(msg ui.ConfigChangedMsg) {
				if p := current.Load(); p != nil {
					p.Send(msg) // a no-op once that lap's program has quit
				}
			}); err != nil {
				log.Printf("Failed to watch config directory: %v", err)
			} else {
				defer stop()
			}
		}

		program := tea.NewProgram(initial)
		current.Store(program)
		result, err := program.Run()
		current.Store(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
			return 1
		}

		// Cast result to ui.Model
		state, ok := result.(ui.Model)
		if !ok {
			// Should not happen, but safe exit
			return 0
		}
		session = state.Session()

		if state.Quitting {
			return state.ExitCode
		}

		if len(state.SelectedBatch) > 0 {
			runBatch(state)
			clearScreen()
			continue
		}

		if state.Selected.Cell != "" {
			// Internal "drako …" cells are dispatched here, at the layer that
			// may wire cli and core together; everything else runs via core.
			name := state.Selected.Name(state.Config)
			switch {
			case strings.HasPrefix(name, "drako purge"):
				if runInternalPurge(name) {
					return 0 // successful purge ends the session
				}
			case strings.HasPrefix(name, "drako open"):
				cli.HandleOpenCommand(name)
			default:
				core.RunCommand(state.Config, state.Selected, state.ActiveProfileName())
			}

			clearScreen()
		}
	}
}

// clearScreen resets the terminal between a finished command and the TUI
// relaunch. Windows has no `clear`; `cls` is a cmd built-in.
func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

// runBatch launches every marked cell together in a multiplexer. The heavy
// lifting (layout, quoting, which backend) lives in the multiplex package;
// this is resolution and handoff, mirroring the single-run dispatch.
func runBatch(state ui.Model) {
	// Neither multiplexer drako drives has a Windows build worth shipping on.
	if runtime.GOOS == "windows" {
		fmt.Println("Batch launch needs tmux or herdr and is not available on Windows yet.")
		core.Pause("\nPress any key to return to the application.")
		return
	}

	cmds := batchCommands(state.Config, state.SelectedBatch, state.ActiveProfileName())
	if len(cmds) == 0 {
		return
	}

	backend, err := multiplex.Resolve(state.Config.BatchForceTmux, multiplex.OSEnv())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		core.Pause("\nPress any key to return to the application.")
		return
	}

	scriptDir, err := os.MkdirTemp("", "drako-batch-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "batch launch failed: %v\n", err)
		core.Pause("\nPress any key to return to drako.")
		return
	}
	// The temp dir's random suffix keeps two batches started together apart.
	session := "drako-" + strings.TrimPrefix(filepath.Base(scriptDir), "drako-batch-")
	// The layout chosen in the dialog; the default when there was none (a single
	// cell) or a mark no longer resolved above.
	tabs := state.SelectedTabs
	if len(tabs) == 0 || len(cmds) != len(state.SelectedBatch) {
		tabs = multiplex.Distribute(len(cmds), multiplex.MinTabs(len(cmds)))
	}
	for _, c := range cmds {
		core.LogExecution(c.Name, "batch: "+strings.Join(c.Argv, " "))
	}

	err = multiplex.Launch(backend, session, cmds, tabs, scriptDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "batch launch failed: %v\n", err)
	}
	// A session drako made outside tmux outlives a detach, and a launch that broke
	// after creating it; its cells may be running, so say how to get back.
	left := false
	if tmux, ok := backend.(*multiplex.Tmux); ok && !tmux.Inside {
		left = exec.Command("tmux", "has-session", "-t", "="+session).Run() == nil // "=": exact name, no prefix match
	}
	if left {
		fmt.Printf("\nBatch session still running: tmux attach -t %s\n", session)
	}
	if err != nil || left {
		core.Pause("\nPress any key to return to drako.")
	}
}

// batchCommands turns the marked entries into cells to launch; an entry that
// no longer resolves, or has no command, is logged and left out.
func batchCommands(cfg config.Config, sels []core.Selection, profile string) []multiplex.Command {
	var cmds []multiplex.Command
	for _, sel := range sels {
		name := sel.Name(cfg)
		parent, item, found := sel.Resolve(cfg)
		if !found {
			log.Printf("batch: skipping %q (not found)", name)
			continue
		}
		// A name resolves to either a grid cell or a dropdown item; both
		// launch the same way, from their own command string and flags.
		command, autoClosePtr, debugPtr := parent.Command, parent.AutoCloseExecution, parent.DebugExecution
		if item != nil {
			command, autoClosePtr, debugPtr = item.Command, item.AutoCloseExecution, item.DebugExecution
		}
		if command == "" {
			log.Printf("batch: skipping %q (no command)", name)
			continue
		}
		cmds = append(cmds, multiplex.Command{
			Name: name,
			Argv: core.ShellArgv(cfg.DefaultShell, command),
			// A debug run always pauses; a pane already shows the output, so it just stays.
			KeepOpen: (autoClosePtr != nil && !*autoClosePtr) || (debugPtr != nil && *debugPtr),
			Profile:  profile,
		})
	}
	return cmds
}

// runInternalPurge runs a "drako purge …" cell in-process and reports whether
// the purge succeeded (after which the app should exit rather than relaunch
// the TUI on a config that may no longer exist).
func runInternalPurge(command string) bool {
	// Strip "drako purge" to match what os.Args[2:] would provide.
	parts := strings.Fields(command)
	if len(parts) < 2 {
		log.Printf("Invalid purge command: %s", command)
		return false
	}

	if err := cli.ExecutePurge(parts[2:]); err != nil {
		if errors.Is(err, cli.ErrPurgeCancelled) {
			fmt.Printf("\nPurge cancelled.\n")
		} else {
			fmt.Printf("\nInternal Purge Error: %v\n", err)
		}
		core.Pause("\nPress any key...")
		return false
	}
	fmt.Printf("\npress any key to exit...")
	core.Pause("")
	return true
}

// cwdFileArg accepts both "--cwd-file=PATH" and "--cwd-file PATH".
func cwdFileArg(args []string) string {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--cwd-file="); ok {
			return v
		}
		if a == "--cwd-file" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// writeCwdFile reports the directory drako is ending in. The shell owns the
// file, so a failure is logged, not fatal.
func writeCwdFile(path string) {
	// Runs after the log file has closed, and a failure means the wrapper won't cd:
	// tell the user on stderr.
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako: could not resolve the working directory for --cwd-file: %v\n", err)
		return
	}
	if err := os.WriteFile(path, []byte(cwd), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "drako: could not write --cwd-file %s: %v\n", path, err)
	}
}

package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/cli"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/multiplex"
	"github.com/lucky7xz/drako/internal/proc"
	"github.com/lucky7xz/drako/internal/tui"
	"golang.org/x/term"
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
	configDir, err := lair.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}
	// The TUI starts only on a layout drako supports, like every command
	// that works in drako's folder.
	if err := lair.CheckLayout(configDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	// One handle on the config directory for the whole process; everything
	// drako writes there goes through it.
	home, err := lair.Create(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	defer home.Close()
	// =======================================
	// Logging setup

	// Each lap throws the model away so a launched command gets a clean
	// terminal, so where the cursor was has to be handed forward explicitly.
	// The zero value restores nothing, which is what the first lap wants.
	var session tui.Session

	// A log that cannot be opened is no reason to lock the user out: run
	// without one, as the CLI does, and say so on screen.
	if f, err := home.OpenLog(); err != nil {
		log.SetOutput(io.Discard)
		session.Notices = append(session.Notices, noLogNotice(lair.LogFile(configDir), err))
	} else {
		defer f.Close()
		log.SetOutput(f)
	}

	// What the TUI is told at each lap; it looks none of it up for itself.
	setup := tui.Setup{
		ConfigDir: configDir,
		Targets:   host.Targets(),
		Profile:   os.Getenv("DRAKO_PROFILE"),
		Glassroot: glassrootMode,
	}

	// One watcher for the process; between laps InitialModel re-reads the config.
	var current atomic.Pointer[tea.Program]
	watching := false

	// Start of TUI Loop
	for {

		// Start the TUI program (Model/View/Update is in internal/tui)
		// We initialize with the *current* directory which might have changed
		// from manual Chdir or from internal logic

		initial := tui.InitialModel(setup, session)
		if out := initial.Outcome(); out.Action == tui.Quit {
			// e.g. glassroot with a broken profile: fail before the TUI starts
			if out.Message != "" {
				fmt.Fprintln(os.Stderr, out.Message)
			}
			return out.ExitCode
		}

		// Watch only after the first InitialModel: on a first launch it writes
		// the bootstrap files, and drako's own writes are no reason to reload.
		if !watching {
			watching = true
			if stop, err := lair.Watch(configDir, func(path string) {
				if p := current.Load(); p != nil {
					p.Send(tui.ConfigChangedMsg{Path: path}) // a no-op once that lap's program has quit
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
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Fprintln(os.Stderr, "drako needs a terminal for its screen. Without one, use: drako ls, drako run <cell>.")
				return 1
			}
			fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
			return 1
		}

		// Cast result to tui.Model
		state, ok := result.(tui.Model)
		if !ok {
			// Should not happen, but safe exit
			return 0
		}
		session = state.Session()

		switch out := state.Outcome(); out.Action {
		case tui.Quit:
			return out.ExitCode

		case tui.Batch:
			runBatch(home, out)
			clearScreen()

		case tui.Run:
			// Internal "drako …" cells are dispatched here, at the layer that
			// may wire cli and proc together; everything else is an ordinary run.
			plan := out.Plans[0]
			switch {
			case strings.HasPrefix(plan.Name, "drako purge"):
				if runInternalPurge(plan.Name) {
					return 0 // successful purge ends the session
				}
			case strings.HasPrefix(plan.Name, "drako open"):
				cli.HandleOpenCommand(plan.Name)
			default:
				if code, end := runPlan(home, plan); end {
					return code
				}
			}
			clearScreen()
		}
	}
}

// noLogNotice is what the user is told when drako runs without its log.
func noLogNotice(path string, err error) tui.DetailState {
	return tui.DetailState{
		Title:    "Running without a log",
		KeyLabel: "Reason",
		Value:    err.Error(),
		Description: "drako could not open its log file, so nothing is logged in this " +
			"session. Everything else works as usual.\n\n" +
			"A common cause: the log file is a link that leaves drako's folder. " +
			"drako does not write through links.\n\n" +
			"Press any key to dismiss.",
		Meta: []tui.DetailMeta{{Label: "Log file", Value: path}},
	}
}

// clearScreen resets the terminal between a finished command and the TUI
// relaunch. Windows has no `clear`; `cls` is a cmd built-in.
func clearScreen() {
	cmd := proc.Clear.Command()
	if runtime.GOOS == "windows" {
		cmd = proc.Cmd.Command("/c", "cls")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

// runBatch launches every marked cell together in a multiplexer. The heavy
// lifting (layout, quoting, which backend) lives in the multiplex package;
// this is resolution and handoff, mirroring the single-run dispatch.
func runBatch(home *lair.Lair, out tui.Outcome) {
	// Neither multiplexer drako drives has a Windows build worth shipping on.
	if runtime.GOOS == "windows" {
		fmt.Println("Batch launch needs tmux or herdr and is not available on Windows yet.")
		pause("\nPress any key to return to the application.")
		return
	}

	cmds := batchCommands(out.Plans)
	if len(cmds) == 0 {
		return
	}

	backend, err := multiplex.Resolve(out.ForceTmux, multiplex.OSEnv())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		pause("\nPress any key to return to the application.")
		return
	}

	// The layout chosen in the dialog; the default when there was none (a single
	// cell) or a mark no longer resolved above.
	tabs := out.Tabs
	if len(tabs) == 0 || len(cmds) != len(out.Plans) {
		tabs = multiplex.Distribute(len(cmds), multiplex.MinTabs(len(cmds)))
	}
	for _, c := range cmds {
		home.LogExecution(c.Name, "batch: "+strings.Join(c.Argv, " "))
	}

	session, err := multiplex.Start(backend, cmds, tabs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "batch launch failed: %v\n", err)
	}
	if session == "" { // no folder, nothing launched
		pause("\nPress any key to return to drako.")
		return
	}
	// A session drako made outside tmux outlives a detach, and a launch that broke
	// after creating it; its cells may be running, so say how to get back.
	left := false
	if tmux, ok := backend.(*multiplex.Tmux); ok && !tmux.Inside {
		left = proc.Tmux.Command("has-session", "-t", "="+session).Run() == nil // "=": exact name, no prefix match
	}
	if left {
		fmt.Printf("\nBatch session still running: tmux attach -t %s\n", session)
	}
	if err != nil || left {
		pause("\nPress any key to return to drako.")
	}
}

// batchCommands turns the marked entries' plans into cells to launch; one with
// nothing to run (gone from the config, or no command) is logged and left out.
func batchCommands(plans []launch.Plan) []multiplex.Command {
	var cmds []multiplex.Command
	for _, plan := range plans {
		if !plan.Runnable() {
			log.Printf("batch: skipping %q (nothing to run)", plan.Name)
			continue
		}
		cmds = append(cmds, multiplex.Command{
			Name: plan.Name,
			Argv: plan.Argv,
			// A debug run always pauses; a pane already shows the output, so it just stays.
			KeepOpen: !plan.AutoClose || plan.Debug,
			Profile:  plan.Profile,
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
		pause("\nPress any key...")
		return false
	}
	fmt.Printf("\npress any key to exit...")
	pause("")
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

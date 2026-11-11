package cli

// drako run executes one grid cell from the shell, addressed the way drako ls
// prints it. Cells marked danger never run this way: only the grid runs them.

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/proc"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// exitRunFailed is drako run's own failure code, kept apart from the
// command's codes the way timeout(1) and env(1) do it.
const exitRunFailed = 125

// HandleRunCommand processes 'drako run [profile:]<addr>': the command's exit
// code when it ran, exitRunFailed with a message on stderr when drako refused.
func HandleRunCommand(args []string) int {
	refs := args[2:]
	if len(refs) != 1 {
		fmt.Fprint(os.Stderr, "drako run: usage: drako run [profile:]<addr>\n"+
			"One cell per run. drako ls lists addresses; drako explain shows a cell.\n")
		return exitRunFailed
	}

	quietLogs()
	bundle, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}

	t, cfg, err := resolveRun(bundle, refs[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}

	plan := launch.Shell(cfg.DefaultShell, t.name(), t.command(), t.p.Name)
	if !plan.Runnable() {
		fmt.Fprintf(os.Stderr, "drako run: %q has no command for this platform\n", plan.Name)
		return exitRunFailed
	}
	logRun(plan)

	code, err := proc.Replace(plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}
	return code
}

// logRun appends the run to history.log before the command takes over. Like
// every history line it is best-effort: a failure is logged, never fatal.
func logRun(plan launch.Plan) {
	configDir, err := lair.ConfigDir()
	if err != nil {
		log.Printf("logging error: could not get config dir: %v", err)
		return
	}
	home, err := lair.Create(configDir)
	if err != nil {
		log.Printf("logging error: %v", err)
		return
	}
	defer home.Close()
	home.LogExecution(plan.Name, strings.Join(plan.Argv, " "))
}

// resolveRun resolves a reference to its target and the config the TUI would
// run it with, refusing anything that isn't a single runnable, safe target.
func resolveRun(bundle lair.ConfigBundle, ref string) (t cellTarget, cfg settings.Config, err error) {
	if t, err = lookupCell(bundle, ref); err != nil {
		return t, cfg, err
	}

	cfg = settings.ApplyProfileOverlay(bundle.Base, t.p.Profile)
	cfg.ApplyDefaults()
	if err := profile.ValidateGrid(cfg.X, cfg.Y, cfg.Commands); err != nil {
		return t, cfg, fmt.Errorf("profile %s has an invalid grid: %v", t.p.Name, err)
	}

	if t.item == nil && t.cmd.Command == "" && len(t.cmd.Items) > 0 {
		return t, cfg, fmt.Errorf("cell %s is a folder; address one of its items (e.g. %s.1)", t.addr, t.addr)
	}
	if t.danger {
		return t, cfg, fmt.Errorf("%s (%s) is marked danger; run it from the drako grid", t.addr, t.name())
	}
	if t.command() == "" {
		return t, cfg, fmt.Errorf("%s (%s) has no command for this platform", t.addr, t.name())
	}
	return t, cfg, nil
}

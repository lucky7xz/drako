package cli

// drako run executes one grid cell from the shell, addressed the way drako ls
// prints it. Cells marked danger never run this way: only the grid runs them.

import (
	"fmt"
	"os"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
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
	bundle, err := config.LoadConfig(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}

	t, cfg, err := resolveRun(bundle, refs[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}

	code, err := core.RunForExit(cfg, t.name(), t.command(), t.p.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}
	return code
}

// resolveRun resolves a reference to its target and the config the TUI would
// run it with, refusing anything that isn't a single runnable, safe target.
func resolveRun(bundle config.ConfigBundle, ref string) (t cellTarget, cfg config.Config, err error) {
	if t, err = lookupCell(bundle, ref); err != nil {
		return t, cfg, err
	}

	cfg = config.ApplyProfileOverlay(bundle.Base, t.p.Profile)
	cfg.ApplyDefaults()
	if err := config.ValidateConfig(cfg); err != nil {
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

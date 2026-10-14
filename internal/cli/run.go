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
		fmt.Fprint(os.Stderr, "Usage: drako run [profile:]<addr>\n"+
			"One cell per run. drako ls lists addresses; drako explain shows a cell.\n")
		return exitRunFailed
	}

	quietLogs()
	bundle, err := config.LoadConfig(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}

	name, commandStr, p, err := resolveRun(bundle, refs[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}

	// The same effective config the TUI runs this profile with: shell, env.
	effective := config.ApplyProfileOverlay(bundle.Base, p.Profile)
	effective.ApplyDefaults()

	code, err := core.RunForExit(effective, name, commandStr, p.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run: %v\n", err)
		return exitRunFailed
	}
	return code
}

// resolveRun turns a [profile:]<addr>[.item] reference into the name and
// command to run, refusing anything that isn't a single runnable, safe target.
func resolveRun(bundle config.ConfigBundle, ref string) (name, commandStr string, p config.ProfileInfo, err error) {
	profileName, addr, itemIdx, err := parseCellRef(ref)
	if err != nil {
		return "", "", p, err
	}
	if p, err = pickProfile(bundle, profileName); err != nil {
		return "", "", p, err
	}
	cmd := config.FindCell(p.Profile.Commands, addr)
	if cmd == nil {
		return "", "", p, fmt.Errorf("no cell %s in profile %s", addr, p.Name)
	}

	name, commandStr, danger := cmd.Name, cmd.Command, cmd.Danger
	switch {
	case itemIdx > 0:
		if itemIdx > len(cmd.Items) {
			return "", "", p, fmt.Errorf("no item %d in cell %s (%d items)", itemIdx, addr, len(cmd.Items))
		}
		item := cmd.Items[itemIdx-1]
		name, commandStr = item.Name, item.Command
		danger = danger || item.Danger
		addr = cmd.ItemAddress(itemIdx)
	case commandStr == "" && len(cmd.Items) > 0:
		return "", "", p, fmt.Errorf("cell %s is a folder; address one of its items (e.g. %s.1)", addr, addr)
	}

	if danger {
		return "", "", p, fmt.Errorf("%s (%s) is marked danger; run it from the drako grid", addr, name)
	}
	if commandStr == "" {
		return "", "", p, fmt.Errorf("%s (%s) has no command for this platform", addr, name)
	}
	return name, commandStr, p, nil
}

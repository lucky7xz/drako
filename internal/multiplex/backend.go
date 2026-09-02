// Package multiplex launches drako's batch mode through a terminal
// multiplexer. Everything a batch risks — quoting, the script per cell, how
// the cells split into tabs — is decided here, table-testable, before any
// backend is called. A Backend only has to put the finished scripts on screen,
// which is the one part that differs between multiplexers.
package multiplex

import (
	"fmt"
	"os"
	"path/filepath"
)

// Backend is one multiplexer. Everything risky and shared — quoting, the script
// per cell, the tab vector — is settled before it is called; a backend only has
// to put each script on screen.
type Backend interface {
	// Name is what config.toml calls this backend.
	Name() string

	// Launch runs paths[i] for cmds[i], grouped into tabs. It reports whether
	// it handed the terminal over: an attached session has already ended when
	// Launch returns, a detached one is still spawning.
	Launch(session string, cmds []Command, tabs []int, paths, env []string) (attached bool, err error)
}

// Launch writes one script per cell into scriptDir, then hands the paths to b.
// The scripts are 0700 — they hold the user's command text, the same
// sensitivity as history.log — and are removed once an attached session ends.
// A detached launch leaves them for the OS tmp reaper rather than racing the
// panes that are still starting.
func Launch(b Backend, session string, cmds []Command, tabs []int, scriptDir string, env []string) error {
	if len(cmds) == 0 {
		return fmt.Errorf("nothing to launch")
	}
	if len(cmds) > MaxCommands {
		return fmt.Errorf("at most %d commands per batch (got %d)", MaxCommands, len(cmds))
	}
	if err := checkTabs(tabs, len(cmds)); err != nil {
		return err
	}

	if err := os.MkdirAll(scriptDir, 0o700); err != nil {
		return fmt.Errorf("could not create script dir: %w", err)
	}
	paths := make([]string, len(cmds))
	for i, c := range cmds {
		paths[i] = filepath.Join(scriptDir, scriptName(i, c))
		if err := os.WriteFile(paths[i], []byte(buildScript(c)), 0o700); err != nil {
			return fmt.Errorf("could not write %s: %w", paths[i], err)
		}
	}

	attached, err := b.Launch(session, cmds, tabs, paths, env)
	if err != nil {
		return fmt.Errorf("%s: %w", b.Name(), err)
	}
	if attached {
		if err := os.RemoveAll(scriptDir); err != nil {
			return fmt.Errorf("could not clean up scripts: %w", err)
		}
	}
	return nil
}

// scriptName is the filename for cell i: ordered, and safe as a path and as a
// shell word.
func scriptName(i int, c Command) string {
	return fmt.Sprintf("%02d-%s.sh", i+1, sanitizeName(c.Name))
}

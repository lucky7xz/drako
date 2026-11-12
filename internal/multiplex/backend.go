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
	"strings"
)

// Backend is one multiplexer. Everything risky and shared — quoting, the script
// per cell, the tab vector — is settled before it is called; a backend only has
// to put each script on screen.
type Backend interface {
	// Name is what config.toml calls this backend.
	Name() string

	// Launch runs paths[i] for cmds[i], grouped into tabs.
	Launch(session string, cmds []Command, tabs []int, paths []string) error
}

// Start launches a batch. multiplex owns the scripts from start to end: it
// makes a private folder for them, names the session after the folder (its
// random suffix keeps two batches started together apart), and launches. The
// session comes back even when the launch fails, so the caller can still look
// for one left running; the folder does not outlive a failed launch.
func Start(b Backend, cmds []Command, tabs []int) (session string, err error) {
	dir, err := os.MkdirTemp("", "drako-batch-*")
	if err != nil {
		return "", fmt.Errorf("could not create script dir: %w", err)
	}
	session = "drako-" + strings.TrimPrefix(filepath.Base(dir), "drako-batch-")
	if err := launchIn(b, session, cmds, tabs, dir); err != nil {
		os.RemoveAll(dir)
		return session, err
	}
	return session, nil
}

// launchIn writes one 0700 script per cell into scriptDir (command text, like
// history.log) and hands b the paths; each script deletes itself once it runs.
func launchIn(b Backend, session string, cmds []Command, tabs []int, scriptDir string) error {
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
			os.RemoveAll(scriptDir)
			return fmt.Errorf("could not write %s: %w", paths[i], err)
		}
	}

	if err := b.Launch(session, cmds, tabs, paths); err != nil {
		os.RemoveAll(scriptDir) // the scripts of cells that never started
		return fmt.Errorf("%s: %w", b.Name(), err)
	}
	return nil
}

// scriptName is the filename for cell i: ordered, and safe as a path and as a
// shell word.
func scriptName(i int, c Command) string {
	return fmt.Sprintf("%02d-%s.sh", i+1, sanitizeName(c.Name))
}

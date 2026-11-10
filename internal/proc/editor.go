package proc

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Editor prepares the user's own editor on path. Nothing runs until the
// caller runs it; the TUI hands it to Bubble Tea, which lends it the terminal.
func Editor(path string) (*exec.Cmd, error) {
	editor, err := resolveEditor()
	if err != nil {
		return nil, err
	}
	return exec.Command(editor[0], append(editor[1:], path)...), nil
}

// resolveEditor picks the user's editor: $VISUAL, $EDITOR (either may carry
// arguments, e.g. "code -w"), then common fallbacks.
func resolveEditor() ([]string, error) {
	for _, v := range []string{os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if parts := strings.Fields(v); len(parts) > 0 {
			return parts, nil
		}
	}
	for _, candidate := range []string{"nano", "vim", "vi"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return []string{candidate}, nil
		}
	}
	if runtime.GOOS == "windows" {
		return []string{"notepad"}, nil
	}
	return nil, errors.New("no editor found: set $EDITOR")
}

package ui

// Editing hands the terminal to the user's own editor (tea.ExecProcess) and
// resumes drako when it exits — harness, don't replace. Reachable only from
// inventory mode, which glassroot already gates: an editor is a shell.

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// editorFinishedMsg reports that the external editor exited.
type editorFinishedMsg struct {
	err  error
	path string // the file that was edited
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

// openInEditorCmd suspends the TUI and opens path in the user's editor.
func openInEditorCmd(path string) tea.Cmd {
	editor, err := resolveEditor()
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err, path: path} }
	}
	c := exec.Command(editor[0], append(editor[1:], path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err, path: path}
	})
}

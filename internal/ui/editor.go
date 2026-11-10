package ui

// Editing hands the terminal to the user's own editor (tea.ExecProcess) and
// resumes drako when it exits — harness, don't replace. Reachable only from
// inventory mode, which glassroot already gates: an editor is a shell.

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/proc"
)

// editorFinishedMsg reports that the external editor exited.
type editorFinishedMsg struct {
	err  error
	path string // the file that was edited
}

// openInEditorCmd suspends the TUI and opens path in the user's editor.
func openInEditorCmd(path string) tea.Cmd {
	c, err := proc.Editor(path)
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err, path: path} }
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err, path: path}
	})
}

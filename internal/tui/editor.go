package tui

// Editing hands the terminal to the user's own editor (tea.ExecProcess) and
// resumes drako when it exits — harness, don't replace. Reachable only from
// inventory mode, which glassroot already gates: an editor is a shell.

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/lair"
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

// afterEdit reloads after the editor closes: broken decks go to the queue, else
// back to the inventory with a verdict. Its lists stay; an edit moves no deck.
func (m Model) afterEdit(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.inventory.status = "Edit failed: " + msg.err.Error()
		if !m.protectsMode() {
			m.mode = inventoryMode
		}
		return m, nil
	}

	// The edit may have changed any equipped profile — reload the bundle.
	bundle, err := m.reload()
	if err != nil {
		m.inventory.status = "Reload failed: " + err.Error()
		if !m.protectsMode() {
			m.mode = inventoryMode
		}
		return m, nil
	}
	if m.GlassrootMode && glassrootRejectsBundle(bundle) {
		return m.failGlassroot()
	}
	m.applyBundle(bundle)
	if len(bundle.Broken) > 0 {
		m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
		m.profile.errorQueueActive = true
		return m.presentNextBrokenProfile(), nil
	}

	name := filepath.Base(msg.path)
	rel, relErr := filepath.Rel(m.profile.configDir, msg.path)
	if relErr != nil {
		rel = msg.path // not under the config directory: lair will refuse it
	}
	if err := lair.CheckProfileFile(m.profile.configDir, rel); err != nil {
		m.inventory.status = fmt.Sprintf("⚠️ %s has errors: %v", name, err)
	} else {
		m.inventory.status, m.inventory.statusOK = "✓ Saved: "+name, true
	}
	if !m.protectsMode() {
		m.mode = inventoryMode
	}
	return m, nil
}

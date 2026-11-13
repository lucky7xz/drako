package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/host"
)

// copyToClipboardCmd copies text to clipboard using the best available method
func copyToClipboardCmd(s string) tea.Cmd {
	return func() tea.Msg {
		host.CopyToClipboard(s)
		return nil
	}
}

func (m Model) updateInfoMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.profile.errorQueueActive {
		var cmds []tea.Cmd
		if key == "y" && m.allowCopy() && m.activeDetail != nil {
			cmds = append(cmds, copyToClipboardCmd(m.activeDetail.Value))
		}
		// Always delegate to presentNextBrokenProfile to handle queue exhaustion and Rescue Trigger
		m = m.presentNextBrokenProfile()
		return m, tea.Batch(cmds...)
	}
	if m.activeDetail != nil {
		l := m.explainLayout()
		if next, paged := pageOffset(key, m.activeDetail.ScrollOffset, l.viewportH, l.maxOffset); paged {
			m.activeDetail.ScrollOffset = next
			return m, nil
		}
	}
	switch key {
	case "y":
		if !m.allowCopy() {
			return m, nil
		}
		if m.activeDetail != nil {
			prev := m.previousMode
			m.mode = prev
			cmd := copyToClipboardCmd(m.activeDetail.Value)
			m.activeDetail = nil // Clear detail state
			return m, cmd
		}
		m.mode = m.previousMode
		m.activeDetail = nil
		return m, nil
	default:
		m.mode = m.previousMode
		m.activeDetail = nil // Clear detail state
		return m, nil
	}
}

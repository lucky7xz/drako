package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/settings"
)

// copyToClipboardCmd copies text to clipboard using the best available method
func copyToClipboardCmd(s string) tea.Cmd {
	return func() tea.Msg {
		host.CopyToClipboard(s)
		return nil
	}
}

func (m Model) updateInfoMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	act := m.Config.Screen(settings.ExplainPopup).Act(msg.String())
	copies := act == settings.Copy && m.allowCopy() && m.activeDetail != nil
	if m.profile.errorQueueActive {
		var cmds []tea.Cmd
		if copies {
			cmds = append(cmds, copyToClipboardCmd(m.activeDetail.Value))
		}
		// Always delegate to presentNextBrokenProfile to handle queue exhaustion and Rescue Trigger
		m = m.presentNextBrokenProfile()
		return m, tea.Batch(cmds...)
	}
	switch {
	case act == settings.Scroll && m.activeDetail != nil:
		l := m.explainLayout()
		m.activeDetail.ScrollOffset, _ = pageOffset(msg.String(), m.activeDetail.ScrollOffset, l.viewportH, l.maxOffset)
		return m, nil
	case act == settings.Copy && !m.allowCopy():
		return m, nil
	}
	// Any other key closes the popup; the copy key copies on its way out.
	var cmd tea.Cmd
	if copies {
		cmd = copyToClipboardCmd(m.activeDetail.Value)
	}
	m.mode = m.previousMode
	m.activeDetail = nil
	return m, cmd
}

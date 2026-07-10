package ui

import (
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// The leader key arms a short two-key sequence in grid mode, quick-nav style:
// leader then 'b' enters batch mode, leader then 1-9 switches profile. An
// unmatched continuation or the timeout cancels the sequence and swallows the
// key — a half-typed sequence must never fire the key's normal meaning.

const leaderWindow = 500 * time.Millisecond // same feel as quick-nav

type leaderState struct {
	pending bool
	timer   *time.Timer
}

type leaderTimeoutMsg struct{}

// IsLeader reports whether the key is the configured leader.
func IsLeader(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Leader
}

// armLeader starts a sequence: mark pending and schedule the timeout.
func (m Model) armLeader() (tea.Model, tea.Cmd) {
	m.leader.pending = true
	m.leader.timer = time.NewTimer(leaderWindow)
	timer := m.leader.timer
	return m, func() tea.Msg {
		<-timer.C
		return leaderTimeoutMsg{}
	}
}

// disarmLeader ends a sequence (continuation seen, or cancelled).
func (m *Model) disarmLeader() {
	if m.leader.timer != nil {
		m.leader.timer.Stop()
		m.leader.timer = nil
	}
	m.leader.pending = false
}

// handleLeaderContinuation consumes the key after the leader. Every path
// returns with the sequence disarmed; unknown keys are swallowed on purpose.
func (m Model) handleLeaderContinuation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.disarmLeader()
	key := msg.String()

	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		if updated, ok := m.switchToProfileIndex(num - 1); ok {
			return updated, nil
		}
		return m, nil
	}

	if key == "b" {
		return m.enterBatchMode()
	}

	return m, nil
}

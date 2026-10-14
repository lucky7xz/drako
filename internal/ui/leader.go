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
	seq     int // tags each arming's timeout; a stale one is ignored
}

type leaderTimeoutMsg struct{ seq int }

// IsLeader reports whether the key is the configured leader.
func IsLeader(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Leader
}

// armLeader starts a sequence: mark pending and schedule the timeout.
func (m Model) armLeader() (tea.Model, tea.Cmd) {
	m.leader.pending = true
	m.leader.seq++
	seq := m.leader.seq
	return m, tea.Tick(leaderWindow, func(time.Time) tea.Msg {
		return leaderTimeoutMsg{seq: seq}
	})
}

// disarmLeader ends a sequence (continuation seen, or cancelled).
func (m *Model) disarmLeader() {
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

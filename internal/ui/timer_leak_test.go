package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// returnsWithin runs a Cmd the way bubbletea does (its own goroutine) and
// reports whether it finishes; a Cmd that never returns is a leaked goroutine.
func returnsWithin(cmd tea.Cmd, d time.Duration) bool {
	done := make(chan struct{})
	go func() { cmd(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

func timerTestModel() Model {
	cfg := config.Config{X: 2, Y: 2, Commands: []config.Command{
		{Name: "one", Command: "true", Row: 0, Col: "a"},
		{Name: "two", Command: "true", Row: 1, Col: "a"},
	}}
	cfg.ApplyDefaults()
	return Model{Config: cfg, styles: BuildStyles(cfg), mode: gridMode,
		gridNav: gridNav{grid: config.BuildGrid(cfg)}}
}

func TestInterruptedQuickNavCmdReturns(t *testing.T) {
	m := timerTestModel()
	next, first := m.quickNav(0)
	next, flash := next.(Model).quickNav(1) // second digit interrupts the first timer
	next, _ = next.(Model).quickNav(0)      // new sequence interrupts the flash

	if !returnsWithin(first, time.Second) {
		t.Error("first-digit timeout Cmd never returned after the second digit")
	}
	if !returnsWithin(flash, time.Second) {
		t.Error("flash timeout Cmd never returned after being interrupted")
	}
}

func TestInterruptedLeaderCmdReturns(t *testing.T) {
	m := timerTestModel()
	next, cmd := m.armLeader()
	next.(Model).handleLeaderContinuation(tea.KeyMsg{Type: tea.KeyEsc})

	if !returnsWithin(cmd, time.Second) {
		t.Error("leader timeout Cmd never returned after the sequence ended")
	}
}

// A timeout from an earlier arming must not cut a newer sequence short.
func TestStaleTimeoutsAreIgnored(t *testing.T) {
	m := timerTestModel()
	next, _ := m.quickNav(0)
	m = next.(Model)
	stale := navTimeoutMsg{seq: m.gridNav.navSeq}
	m.gridNav.cancel()
	next, _ = m.quickNav(0) // re-armed
	next, _ = next.(Model).Update(stale)
	if !next.(Model).gridNav.navPending {
		t.Error("stale nav timeout ended the new quicknav sequence")
	}

	next, _ = m.armLeader()
	m = next.(Model)
	staleLeader := leaderTimeoutMsg{seq: m.leader.seq}
	m.disarmLeader()
	next, _ = m.armLeader()
	next, _ = next.(Model).Update(staleLeader)
	if !next.(Model).leader.pending {
		t.Error("stale leader timeout disarmed the new leader sequence")
	}
}

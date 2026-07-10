package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// withTmux stubs the batch tmux gate.
func withTmux(t *testing.T, present bool) {
	t.Helper()
	old := tmuxLookPath
	t.Cleanup(func() { tmuxLookPath = old })
	tmuxLookPath = func(string) (string, error) {
		if present {
			return "/usr/bin/tmux", nil
		}
		return "", errors.New("not found")
	}
}

// press drives one key through the grid-mode handler and returns the model.
func press(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, _ := m.updateGridMode(msg)
	out, ok := next.(Model)
	if !ok {
		t.Fatalf("updateGridMode returned %T", next)
	}
	return out
}

func TestLeaderThenDigitSwitchesProfile(t *testing.T) {
	m := switchTestModel(t)

	m = press(t, m, keyRunes("m"))
	if !m.leader.pending {
		t.Fatal("leader press must arm the sequence")
	}
	m = press(t, m, keyRunes("2"))
	if got := m.ActiveProfileName(); got != "beta" {
		t.Errorf("m,2 should switch to beta, got %q", got)
	}
	if m.leader.pending {
		t.Error("sequence must disarm after the continuation")
	}
}

func TestLeaderThenBEntersBatchMode(t *testing.T) {
	withTmux(t, true)
	m := switchTestModel(t)

	m = press(t, m, keyRunes("m"))
	m = press(t, m, keyRunes("b"))
	if m.mode != batchMode {
		t.Fatalf("m,b should enter batch mode, mode = %v", m.mode)
	}
}

func TestLeaderBatchWithoutTmux(t *testing.T) {
	withTmux(t, false)
	m := switchTestModel(t)

	m = press(t, m, keyRunes("m"))
	m = press(t, m, keyRunes("b"))
	if m.mode == batchMode {
		t.Fatal("batch must not activate without tmux")
	}
	if !strings.Contains(strings.ToLower(m.profile.statusMessage), "tmux") {
		t.Errorf("status should explain the tmux gate, got %q", m.profile.statusMessage)
	}
}

func TestLeaderMismatchIsSwallowed(t *testing.T) {
	m := switchTestModel(t)

	m = press(t, m, keyRunes("m"))
	// "q" would normally quit — inside a pending sequence it must be inert.
	m = press(t, m, keyRunes("q"))
	if m.Quitting {
		t.Fatal("unmatched continuation must be swallowed, not act")
	}
	if m.leader.pending {
		t.Error("mismatch must disarm the sequence")
	}
}

func TestLeaderTimeoutDisarms(t *testing.T) {
	m := switchTestModel(t)
	m = press(t, m, keyRunes("m"))

	next, _ := m.Update(leaderTimeoutMsg{})
	m = next.(Model)
	if m.leader.pending {
		t.Error("timeout must disarm the sequence")
	}
}

func TestLegacyAltDigitStillSwitches(t *testing.T) {
	m := switchTestModel(t)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2"), Alt: true}
	next, _ := m.Update(msg)
	got := next.(Model)
	if name := got.ActiveProfileName(); name != "beta" {
		t.Errorf("alt+2 legacy chord should still switch to beta, got %q", name)
	}
}

func TestLeaderGlassrootBlocksBatchOnly(t *testing.T) {
	withTmux(t, true)
	m := switchTestModel(t)
	m.GlassrootMode = true

	b := press(t, press(t, m, keyRunes("m")), keyRunes("b"))
	if b.mode == batchMode {
		t.Fatal("batch mode must be unavailable in glassroot")
	}

	p := press(t, press(t, m, keyRunes("m")), keyRunes("2"))
	if got := p.ActiveProfileName(); got != "beta" {
		t.Errorf("profile switching stays allowed in glassroot, got %q", got)
	}
}

package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// glassrootModel builds a grid-mode model with known bindings for every
// restricted action, in or out of glassroot.
func glassrootModel(glassroot bool) Model {
	return Model{
		mode:          gridMode,
		GlassrootMode: glassroot,
		gridNav:       gridNav{grid: [][]string{{"A"}}},
		Config: config.Config{
			Keys: config.InputConfig{
				Inventory:    "i",
				PathGridMode: "tab",
				Lock:         "r",
			},
		},
	}
}

// The glassroot property: every restricted key is a complete no-op — the mode
// does not change and no command is issued. One table guards the whole gate.
func TestGlassrootBlocksRestrictedKeys(t *testing.T) {
	restricted := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"inventory", keyRune("i")},
		{"path mode", tea.KeyMsg{Type: tea.KeyTab}},
		{"lock toggle", keyRune("r")},
	}

	for _, tc := range restricted {
		t.Run(tc.name, func(t *testing.T) {
			m := glassrootModel(true)
			tm, cmd := m.Update(tc.msg)
			got := tm.(Model)
			if got.mode != gridMode {
				t.Errorf("mode = %v, want gridMode (key must be a no-op)", got.mode)
			}
			if cmd != nil {
				t.Error("expected no command from a blocked key")
			}
		})
	}

	// Sanity: the same keys do act outside glassroot, so the table above is
	// testing the gate, not dead bindings.
	t.Run("inventory key works without glassroot", func(t *testing.T) {
		m := glassrootModel(false)
		tm, _ := m.Update(keyRune("i"))
		if got := tm.(Model); got.mode != inventoryMode {
			t.Errorf("mode = %v, want inventoryMode", got.mode)
		}
	})
	t.Run("path key works without glassroot", func(t *testing.T) {
		m := glassrootModel(false)
		tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if got := tm.(Model); got.mode != pathMode {
			t.Errorf("mode = %v, want pathMode", got.mode)
		}
	})
}

func TestGlassrootBlocksClipboard(t *testing.T) {
	m := glassrootModel(true)
	m.mode = infoMode
	m.previousMode = gridMode
	m.activeDetail = &DetailState{Value: "secret"}

	tm, cmd := m.updateInfoMode(keyRune("y"))
	got := tm.(Model)
	if cmd != nil {
		t.Error("expected no clipboard command in glassroot")
	}
	if got.mode != infoMode {
		t.Errorf("mode = %v, want infoMode (y is ignored, popup stays)", got.mode)
	}
}

// A broken profile in glassroot must end the session through the normal
// bubbletea shutdown — previously this path called os.Exit(1) mid-loop and
// was untestable.
func TestGlassrootFailQuitsWithExitCode(t *testing.T) {
	m := glassrootModel(true)
	got, cmd := m.failGlassroot()

	if !got.Quitting || got.ExitCode != 1 {
		t.Errorf("Quitting=%v ExitCode=%d, want true/1", got.Quitting, got.ExitCode)
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected the command to produce tea.QuitMsg")
	}
}

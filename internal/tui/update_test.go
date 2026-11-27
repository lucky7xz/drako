package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/settings"
	gopsutil_net "github.com/shirou/gopsutil/v3/net"
)

// Helper to create a basic model for Grid testing
func createTestGridModel() Model {
	m := Model{
		mode: gridMode,
		gridNav: gridNav{
			grid: [][]string{
				{"A", "B"},
				{"C", "D"},
			},
		},
		Config: settings.Config{
			Keys: settings.InputConfig{
				NavUp:    []string{"up", "k"},
				NavDown:  []string{"down", "j"},
				NavLeft:  []string{"left", "h"},
				NavRight: []string{"right", "l"},
			},
		},
	}
	return m
}

func TestUpdateGridMode_Navigation(t *testing.T) {
	m := createTestGridModel()
	var tm tea.Model

	// Test Right
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyRight})
	m = tm.(Model)
	if m.gridNav.cursorCol != 1 {
		t.Errorf("Expected cursorCol 1, got %d", m.gridNav.cursorCol)
	}

	// Test Down
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyDown})
	m = tm.(Model)
	if m.gridNav.cursorRow != 1 {
		t.Errorf("Expected cursorRow 1, got %d", m.gridNav.cursorRow)
	}

	// Test Left
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyLeft})
	m = tm.(Model)
	if m.gridNav.cursorCol != 0 {
		t.Errorf("Expected cursorCol 0, got %d", m.gridNav.cursorCol)
	}

	// Test Up
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyUp})
	m = tm.(Model)
	if m.gridNav.cursorRow != 0 {
		t.Errorf("Expected cursorRow 0, got %d", m.gridNav.cursorRow)
	}
}

func createTestInventoryModel() Model {
	state := equip.NewInventoryState(
		[]string{"a.profile.toml", "b.profile.toml"}, // Visible
		[]string{"c.profile.toml", "d.profile.toml"}, // Inventory
		equip.MaxEquipped,
	)
	m := Model{
		mode: inventoryMode,
		inventory: inventoryModel{
			State:       state,
			cursor:      0,
			focusedList: 0, // 0=Visible, 1=Inventory, 2=Apply, 3=Rescue
		},
		Config: settings.Config{
			Keys: settings.InputConfig{
				NavUp:    []string{"up", "k"},
				NavDown:  []string{"down", "j"},
				NavLeft:  []string{"left", "h"},
				NavRight: []string{"right", "l"},
			},
		},
	}
	return m
}

func TestUpdateInventoryMode_Navigation(t *testing.T) {
	m := createTestInventoryModel()
	var tm tea.Model

	// Initial State: Focused List 0 (Visible), Cursor 0

	// Test Right (Next item in list)
	tm, _ = m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyRight})
	m = tm.(Model)
	if m.inventory.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", m.inventory.cursor)
	}

	// Test Down (Switch to Inventory List - index 1)
	tm, _ = m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyDown})
	m = tm.(Model)
	if m.inventory.focusedList != 1 {
		t.Errorf("Expected focusedList 1, got %d", m.inventory.focusedList)
	}
	// Cursor should reset to 0 on list switch
	if m.inventory.cursor != 0 {
		t.Errorf("Expected cursor 0 after list switch, got %d", m.inventory.cursor)
	}

	// Test Up (Switch back to Visible List - index 0)
	tm, _ = m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyUp})
	m = tm.(Model)
	if m.inventory.focusedList != 0 {
		t.Errorf("Expected focusedList 0, got %d", m.inventory.focusedList)
	}
}

// A status clears when its own timer fires; an older timer leaves a newer
// status alone.
func TestStatusClearsOnlyOnItsOwnTimer(t *testing.T) {
	m := createTestGridModel()
	m.setProfileStatus("first", true)
	first := m.profile.statusClearTimerID
	m.setProfileStatus("second", true)

	next, _ := m.Update(profileStatusClearMsg{id: first})
	if got := next.(Model).profile.statusMessage; got != "second" {
		t.Fatalf("the first status's timer left %q, want second", got)
	}
	next, _ = next.Update(profileStatusClearMsg{id: m.profile.statusClearTimerID})
	if got := next.(Model).profile.statusMessage; got != "" {
		t.Errorf("its own timer left %q", got)
	}
}

// The status line: traffic once two samples span time (averaged over the
// window), online (active) above 2 KB/s, online (idle) below, offline, and
// error when the probe fails.
func TestNetworkStatusLine(t *testing.T) {
	m := createTestGridModel()
	m.net.meter = host.TrafficMeter{WindowSeconds: 7.5}
	t0 := time.Now()
	sample := func(m Model, msg networkStatusMsg) Model {
		t.Helper()
		next, cmd := m.Update(msg)
		if cmd == nil {
			t.Error("no next probe scheduled")
		}
		return next.(Model)
	}
	kb := func(n uint64) gopsutil_net.IOCountersStat { return gopsutil_net.IOCountersStat{BytesRecv: n * 1024} }

	for _, step := range []struct {
		msg             networkStatusMsg
		traffic, online string
	}{
		{networkStatusMsg{online: true, counters: kb(0), t: t0}, "calculating...", "online (idle)"},
		{networkStatusMsg{online: true, counters: kb(1), t: t0.Add(time.Second)}, "↓ 1024 B/s", "online (idle)"},
		{networkStatusMsg{online: true, counters: kb(201), t: t0.Add(2 * time.Second)}, "↓ 100.50 KB/s", "online (active)"},
		{networkStatusMsg{online: false, counters: kb(201), t: t0.Add(3 * time.Second)}, "↓ 67.00 KB/s", "offline"},
	} {
		m = sample(m, step.msg)
		if !strings.Contains(m.net.traffic, step.traffic) || !strings.Contains(m.net.online, step.online) {
			t.Errorf("at %v: traffic %q, online %q; want %q, %q", step.msg.t.Sub(t0), m.net.traffic, m.net.online, step.traffic, step.online)
		}
	}
	if m = sample(m, networkStatusMsg{err: errors.New("no interfaces")}); !strings.Contains(m.net.traffic, "error") {
		t.Errorf("a failed probe shows %q, want error", m.net.traffic)
	}
}

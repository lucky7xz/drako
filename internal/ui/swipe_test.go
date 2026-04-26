package ui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

func makeSwipeTestModel(t *testing.T) Model {
	t.Helper()
	applyThemeStyles(config.Config{Theme: "dracula"})

	p1 := t.TempDir() + "/p1.toml"
	p2 := t.TempDir() + "/p2.toml"
	p3 := t.TempDir() + "/p3.toml"
	os.WriteFile(p1, []byte(""), 0644)
	os.WriteFile(p2, []byte(""), 0644)
	os.WriteFile(p3, []byte(""), 0644)

	return Model{
		termWidth:  150,
		termHeight: 50,
		mode:       gridMode,
		Config:     config.Config{X: 2, Y: 2},
		baseConfig: config.Config{X: 2, Y: 2},
		grid:       [][]string{{"a", "b"}, {"c", "d"}},
		profiles: []config.ProfileInfo{
			{Name: "p1", Path: p1},
			{Name: "p2", Path: p2},
			{Name: "p3", Path: p3},
		},
		activeProfileIndex: 1,
	}
}

func TestSwipeRightSwitchesToNextProfile(t *testing.T) {
	m := makeSwipeTestModel(t)

	// Press at X=20
	press := tea.MouseMsg{Action: tea.MouseActionPress, X: 20, Y: 25}
	result, _ := m.Update(press)
	m = result.(Model)

	// Release at X=40 (delta=+20, swipe right → next profile)
	release := tea.MouseMsg{Action: tea.MouseActionRelease, X: 40, Y: 26}
	result, _ = m.Update(release)
	m = result.(Model)

	if m.activeProfileIndex != 2 {
		t.Errorf("swipe right: expected profile index 2, got %d", m.activeProfileIndex)
	}
}

func TestSwipeLeftSwitchesToPrevProfile(t *testing.T) {
	m := makeSwipeTestModel(t)

	// Press at X=60
	press := tea.MouseMsg{Action: tea.MouseActionPress, X: 60, Y: 25}
	result, _ := m.Update(press)
	m = result.(Model)

	// Release at X=40 (delta=-20, swipe left → prev profile)
	release := tea.MouseMsg{Action: tea.MouseActionRelease, X: 40, Y: 26}
	result, _ = m.Update(release)
	m = result.(Model)

	if m.activeProfileIndex != 0 {
		t.Errorf("swipe left: expected profile index 0, got %d", m.activeProfileIndex)
	}
}

func TestSwipeBelowThresholdDoesNotSwitchProfile(t *testing.T) {
	m := makeSwipeTestModel(t)

	press := tea.MouseMsg{Action: tea.MouseActionPress, X: 20, Y: 25}
	result, _ := m.Update(press)
	m = result.(Model)

	// Only 5 columns - below threshold
	release := tea.MouseMsg{Action: tea.MouseActionRelease, X: 25, Y: 26}
	result, _ = m.Update(release)
	m = result.(Model)

	if m.activeProfileIndex != 1 {
		t.Errorf("small swipe: expected profile index 1 (unchanged), got %d", m.activeProfileIndex)
	}
}

func TestMostlyVerticalSwipeDoesNotSwitchProfile(t *testing.T) {
	m := makeSwipeTestModel(t)

	press := tea.MouseMsg{Action: tea.MouseActionPress, X: 20, Y: 10}
	result, _ := m.Update(press)
	m = result.(Model)

	// Horizontal delta=15 but vertical delta=20 → not a horizontal swipe
	release := tea.MouseMsg{Action: tea.MouseActionRelease, X: 35, Y: 30}
	result, _ = m.Update(release)
	m = result.(Model)

	if m.activeProfileIndex != 1 {
		t.Errorf("vertical swipe: expected profile index 1 (unchanged), got %d", m.activeProfileIndex)
	}
}

func TestSwipeInInventoryModeDoesNotSwitchProfile(t *testing.T) {
	m := makeSwipeTestModel(t)
	m.mode = inventoryMode
	m.inventory = InitInventoryModel("")

	press := tea.MouseMsg{Action: tea.MouseActionPress, X: 20, Y: 25}
	result, _ := m.Update(press)
	m = result.(Model)

	release := tea.MouseMsg{Action: tea.MouseActionRelease, X: 50, Y: 26}
	result, _ = m.Update(release)
	m = result.(Model)

	if m.activeProfileIndex != 1 {
		t.Errorf("inventory mode swipe: expected profile index 1 (unchanged), got %d", m.activeProfileIndex)
	}
}

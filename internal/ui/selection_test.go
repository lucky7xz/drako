package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/profile"
)

// twoFolders: folders "git" (A0) and "docker" (B0) each hold an item named
// "Status", and a plain cell "Logs" (C0) shares its name with a git item.
func twoFolders() Model {
	cfg := config.Config{X: 3, Y: 1, Commands: []profile.Command{
		{Name: "git", Row: 0, Col: "a", Items: []profile.CommandItem{
			{Name: "Status", Command: "git status"},
			{Name: "Logs", Command: "git log"},
		}},
		{Name: "docker", Row: 0, Col: "b", Items: []profile.CommandItem{
			{Name: "Status", Command: "docker ps"},
		}},
		{Name: "Logs", Command: "journalctl -f", Row: 0, Col: "c"},
	}}
	cfg.ApplyDefaults()
	return Model{Config: cfg, styles: BuildStyles(cfg), mode: gridMode,
		gridNav: gridNav{grid: profile.BuildGrid(cfg.X, cfg.Y, cfg.Commands)}}
}

func TestDropdownChoiceRunsTheItemFromItsOwnFolder(t *testing.T) {
	m := twoFolders()
	m.gridNav.cursorCol = 1 // docker
	next, _ := m.updateGridMode(tea.KeyMsg{Type: tea.KeyEnter})
	next, _ = next.(Model).updateDropdownMode(tea.KeyMsg{Type: tea.KeyEnter})

	sel := next.(Model).selected
	_, item, ok := sel.Resolve(m.Config.Commands)
	if !ok || item == nil || item.Command != "docker ps" {
		t.Errorf("docker → Status (%+v) resolves to %+v, want docker ps", sel, item)
	}
}

func TestGridCellRunsItselfNotAFolderItemOfTheSameName(t *testing.T) {
	m := twoFolders()
	m.gridNav.cursorCol = 2 // Logs cell
	next, _ := m.updateGridMode(tea.KeyMsg{Type: tea.KeyEnter})

	cell, item, ok := next.(Model).selected.Resolve(m.Config.Commands)
	if !ok || item != nil || cell.Command != "journalctl -f" {
		t.Errorf("Logs cell resolves to %+v / %+v, want the cell itself", cell, item)
	}
}

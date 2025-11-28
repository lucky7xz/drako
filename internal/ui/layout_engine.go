package ui

import "github.com/lucky7xz/drako/internal/config"

// Layout controls the visibility of UI elements based on terminal size.
type Layout struct {
	ShowHeader bool
}

// CalculateLayout determines which UI elements should be visible.
// It prioritizes the Grid and Footer information.
func CalculateLayout(termW, termH int, cfg config.Config) Layout {
	// Calculate the height of the essential central grid
	gridHeight := cfg.Y * GridCellHeight

	// Calculate estimated height of footer elements (Help, Status, Profile, Path)
	// This is roughly 8-10 lines depending on state.
	// Let's assume a safe minimum for the footer area.
	footerHeight := 10

	// Total height needed for everything including header
	fullHeight := gridHeight + LayoutHeaderHeight + footerHeight + LayoutVertPadding

	l := Layout{
		ShowHeader: true,
	}

	// If terminal is too short, hide the header first
	if termH < fullHeight {
		l.ShowHeader = false
	}

	return l
}

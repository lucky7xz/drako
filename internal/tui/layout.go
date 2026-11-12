package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/settings"
)

// Layout constants define the geometry of the TUI elements.
const (
	// Grid Geometry Defaults (used for fallbacks or min sizing)
	GridCellHeight   = 4
	GridMaxTextWidth = 25
	// MinCellFootprint is the narrowest a cell may be compressed to when
	// terminal space is tight (8 text columns + padding + border). It caps
	// the size gate's width demand at a content-independent constant.
	MinCellFootprint = 12

	// comfortableCellFootprint mirrors cellFootprint's cap (viewport.go):
	// the widest a cell's content is ever allowed to be. Used here, not to
	// measure real content, but to estimate how much width the grid wants
	// before fitFootprint would start compressing cells below it.
	comfortableCellFootprint = GridMaxTextWidth + 4

	// UI Elements Height
	LayoutHeaderHeight = 10 // Logo + spacing
	LayoutStatusHeight = 5  // Status bar + network + path
	LayoutSideMargin   = 4  // Left + Right margins
	LayoutVertPadding  = 2  // Top + Bottom padding

	// A bigger grid scrolls instead of demanding more terminal, so layout
	// math never claims space for more than this many rows/columns.
	MinVisibleGridRows = 3
	MinVisibleGridCols = 3
)

// Layout controls the visibility of UI elements based on terminal size.
type Layout struct {
	ShowHeader bool
	ShowFooter bool
}

// widestLineWidth is the on-screen width of the widest line in s. Used to
// measure the resolved header art (or view-specific title).
func widestLineWidth(s string) int {
	widest := 0
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > widest {
			widest = w
		}
	}
	return widest
}

// CalculateLayout determines which UI elements should be visible.
// It prioritizes the Grid and Footer information. headerArt is the
// resolved header template (m.styles.HeaderArt for the ASCII-art views; a
// view-specific title string, e.g. inventory mode's; "" if the view has no
// header-equivalent element) — used only to measure its width.
func CalculateLayout(termW, termH int, cfg settings.Config, headerArt string) Layout {
	// The essential central grid: beyond the minimum window it scrolls,
	// so it never claims more height than that window.
	gridHeight := min(cfg.Y, MinVisibleGridRows) * GridCellHeight

	// Calculate estimated height of footer elements (Help, Status, Profile, Path)
	// This is roughly 8-10 lines depending on state.
	// We increase this request to ensure plenty of breathing room for the grid.
	footerHeight := 14

	// Total height needed for everything including header
	fullHeight := gridHeight + LayoutHeaderHeight + footerHeight + LayoutVertPadding

	l := Layout{
		ShowHeader: true,
		ShowFooter: true,
	}

	// If terminal is too short, hide the header first
	if termH < fullHeight {
		l.ShowHeader = false

		// If still too short, hide the footer
		neededWithoutHeader := gridHeight + footerHeight + LayoutVertPadding
		if termH < neededWithoutHeader {
			l.ShowFooter = false
		}
	}

	// If terminal is too narrow, hide the header too — before the grid
	// would need to compress cells below their comfortable footprint
	// (fitFootprint, viewport.go), and before the terminal is narrower than
	// the header art itself. The footer is never hidden by width: its bars
	// trim to fit instead.
	neededWidth := min(cfg.X, MinVisibleGridCols)*comfortableCellFootprint + LayoutSideMargin
	if hw := widestLineWidth(headerArt) + LayoutSideMargin; hw > neededWidth {
		neededWidth = hw
	}
	if termW < neededWidth {
		l.ShowHeader = false
	}

	return l
}

package ui

// Layout constants define the geometry of the TUI elements.
const (
	// Grid Geometry
	// CellWidth = Content (25) + Padding (2) + Border (2) = 29
	GridCellWidth    = 29
	GridCellHeight   = 4
	GridMaxTextWidth = 25

	// UI Elements Height
	LayoutHeaderHeight = 10 // Logo + spacing
	LayoutStatusHeight = 5  // Status bar + network + path
	LayoutSideMargin   = 4  // Left + Right margins
	LayoutVertPadding  = 2  // Top + Bottom padding
)

/*
TODO(Grid 2.0): Dynamic Responsive Layout

Current implementation uses static GridCellWidth/Height.
We should move towards a calculated layout system:

1. Flexible Container Logic:
   - Calculate AvailableWidth = TermWidth - (Margins)
   - CellWidth = AvailableWidth / GridCols (X)
   - This allows cells to stretch/shrink with the terminal instead of being fixed.

2. Priority Rendering (Responsive Design):
   - Define breakpoints for UI elements.
   - If TermHeight < 20: Hide Header (ASCII Art).
   - If TermWidth < 60: Hide Other UI Elements.
   - Only show the "Terminal too small" overlay as a last resort.

3. Dynamic Truncation:
   - MaxTextWidth should be derived from the calculated CellWidth.
   - e.g. MaxTextWidth = CellWidth - (Padding + Borders).

Implementation Plan:
- Convert these consts into a `Layout` struct.
- Create `func CalculateLayout(termW, termH int, config Config) Layout`.
- Pass this Layout to the View functions.
*/

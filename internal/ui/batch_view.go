package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/multiplex"
)

// Batch mode rendering: the counter line swaps for a mark count, runnable
// cells get a mark glyph, and the footer explains the three keys. Everything
// else reuses the normal grid view.

// renderBatchCounter replaces the profile counter while marking.
func (m Model) renderBatchCounter() string {
	return m.styles.Title.Render(fmt.Sprintf("[ BATCH %d/%d ]", len(m.batch.marked), multiplex.MaxCommands))
}

// batchHelpText is the batch-mode footer line.
func (m Model) batchHelpText() string {
	if m.batch.choosing() {
		return "Layout | ←/→/ad: Field, ↑/↓/ws: Adjust, Enter: Launch, Esc/q: Back"
	}
	return fmt.Sprintf("Batch | Space: Mark, Enter: Launch %d, Esc/q: Cancel", len(m.batch.marked))
}

// layoutFields are the dialog's selectable boxes, in a row: the tab count,
// then one per tab holding the marks it will run. Unstyled — the popup draws
// them with the grid's own cell chrome.
func (m Model) layoutFields() (labels, bodies []string) {
	labels = []string{"tabs"}
	bodies = []string{strconv.Itoa(len(m.batch.tabs))}
	cell := 0
	for i, panes := range m.batch.tabs {
		glyphs := make([]string, panes)
		for j := range glyphs {
			glyphs[j] = markGlyphs[cell+j]
		}
		labels = append(labels, fmt.Sprintf("T%d", i+1))
		bodies = append(bodies, strings.Join(glyphs, " "))
		cell += panes
	}
	return labels, bodies
}

// layoutFieldWidth is the content width every box uses. A full tab is the
// widest a box can get, so it does not change while the knobs turn and the
// boxes stay put under the cursor.
func (m Model) layoutFieldWidth() int {
	full := make([]string, min(len(m.batch.marked), multiplex.PanesPerTab))
	for i := range full {
		full[i] = markGlyphs[i]
	}
	return max(lipgloss.Width(strings.Join(full, " ")), len("tabs")+2)
}

// markGlyphs carry a mark's position in the launch order. Circled digits stay
// tellable apart from the dropdown's own item numbering, and MaxCommands never
// exceeds nine so every mark has one.
var markGlyphs = []string{"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨"}

// markGlyph is the prefix for name: its position when marked, ○ when not.
func (b batchState) markGlyph(name string) string {
	if i := b.mark(name); i > 0 {
		return markGlyphs[i-1] + " "
	}
	return "○ "
}

// itemMarkPrefix is batchCellPrefix's dropdown twin: mark glyphs for the
// folder's items while a dropdown batch is active, nothing otherwise.
func (m Model) itemMarkPrefix(item config.CommandItem) string {
	if !m.batch.dropdown || item.Command == "" {
		return ""
	}
	return m.batch.markGlyph(item.Name)
}

// batchCellPrefix returns the mark prefix for a cell, and nothing for cells a
// batch can't include.
func (m Model) batchCellPrefix(name string) string {
	if m.mode != batchMode || name == "" || !m.markable(name) {
		return ""
	}
	return m.batch.markGlyph(name)
}

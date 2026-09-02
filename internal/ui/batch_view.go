package ui

import (
	"fmt"
	"strings"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/multiplex"
)

// Batch mode rendering: the counter line swaps for a mark count, runnable
// cells get a mark glyph, and the footer explains the three keys. Everything
// else reuses the normal grid view.

// renderBatchCounter replaces the profile counter while marking, and the
// layout row once the dialog is open.
func (m Model) renderBatchCounter() string {
	if m.batch.choosing() {
		return m.styles.Title.Render(m.tabDialogRow())
	}
	return m.styles.Title.Render(fmt.Sprintf("[ BATCH %d/%d ]", len(m.batch.marked), multiplex.MaxCommands))
}

// tabDialogRow shows the layout being edited: the tab count, then one box per
// tab. The focused field is marked with ‹› so focus reads without colour.
func (m Model) tabDialogRow() string {
	field := func(i int, label string, v int) string {
		if m.batch.focus == i {
			return fmt.Sprintf("%s‹%d›", label, v)
		}
		return fmt.Sprintf("%s[%d]", label, v)
	}
	parts := []string{field(0, "tabs", len(m.batch.tabs))}
	for i, panes := range m.batch.tabs {
		parts = append(parts, field(i+1, fmt.Sprintf("T%d", i+1), panes))
	}
	return strings.Join(parts, " ")
}

// batchHelpText is the batch-mode footer line.
func (m Model) batchHelpText() string {
	if m.batch.choosing() {
		return "Layout | ←→: Field, ↑↓: Adjust, Enter: Launch, Esc: Back"
	}
	return fmt.Sprintf("Batch | Space: Mark, Enter: Launch %d, Esc/q: Cancel", len(m.batch.marked))
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

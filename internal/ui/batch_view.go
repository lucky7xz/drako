package ui

import (
	"fmt"

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
	return fmt.Sprintf("Batch | Space: Mark, Enter: Launch %d, Esc/q: Cancel", len(m.batch.marked))
}

// itemMarkPrefix is batchCellPrefix's dropdown twin: mark glyphs for the
// folder's items while a dropdown batch is active, nothing otherwise.
func (m Model) itemMarkPrefix(item config.CommandItem) string {
	if !m.batch.dropdown || item.Command == "" {
		return ""
	}
	if m.batch.marked[item.Name] {
		return "◉ "
	}
	return "○ "
}

// batchCellPrefix returns the mark glyph for a cell: ◉ marked, ○ markable,
// nothing for cells a batch can't include.
func (m Model) batchCellPrefix(name string) string {
	if m.mode != batchMode || name == "" || !m.markable(name) {
		return ""
	}
	if m.batch.marked[name] {
		return "◉ "
	}
	return "○ "
}
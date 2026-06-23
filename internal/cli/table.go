package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// table writes headers and rows as a box-drawing grid to out. Columns are
// left-aligned and sized to the widest cell, counting runes (not bytes) so
// multi-byte content still aligns. Rows shorter than the header are padded with
// blanks; cells beyond the header count are ignored.
func table(out io.Writer, headers []string, rows [][]string) {
	// Column widths: start from the headers, then grow to the widest cell.
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, r := range rows {
		for i := range headers {
			if i < len(r) {
				if w := utf8.RuneCountInString(r[i]); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	// A horizontal rule with the given corner/junction pieces.
	bar := func(left, mid, right string) {
		var b strings.Builder
		b.WriteString(left)
		for i, w := range widths {
			if i > 0 {
				b.WriteString(mid)
			}
			b.WriteString(strings.Repeat("─", w+2))
		}
		b.WriteString(right)
		fmt.Fprintln(out, b.String())
	}

	// One content line. cells is padded/truncated to the column count.
	row := func(cells []string) {
		var b strings.Builder
		b.WriteString("│")
		for i, w := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			pad := w - utf8.RuneCountInString(cell)
			b.WriteString(" " + cell + strings.Repeat(" ", pad) + " │")
		}
		fmt.Fprintln(out, b.String())
	}

	bar("┌", "┬", "┐")
	row(headers)
	bar("├", "┼", "┤")
	for _, r := range rows {
		row(r)
	}
	bar("└", "┴", "┘")
}

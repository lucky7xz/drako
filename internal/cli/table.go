package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// safeText shows control characters as visible escapes (\x1b), so text from a
// deck can't drive the terminal. Newlines and tabs pass through.
func safeText(s string) string {
	if !strings.ContainsFunc(s, isUnsafeRune) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if isUnsafeRune(r) {
			fmt.Fprintf(&b, "\\x%02x", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isUnsafeRune(r rune) bool {
	return unicode.IsControl(r) && r != '\n' && r != '\t'
}

// table writes headers and rows as a box-drawing grid to out. Columns are
// left-aligned and sized to the widest cell, measured in terminal CELLS
// (ansi.StringWidth) rather than runes or bytes, so emoji and CJK content
// align — this is the same width opinion the TUI (lipgloss) uses. Rows
// shorter than the header are padded with blanks; cells beyond the header
// count are ignored.
func table(out io.Writer, headers []string, rows [][]string) {
	rows = safeRows(rows)
	// Column widths: start from the headers, then grow to the widest cell.
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = ansi.StringWidth(h)
	}
	for _, r := range rows {
		for i := range headers {
			if i < len(r) {
				if w := ansi.StringWidth(r[i]); w > widths[i] {
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
			pad := max(w-ansi.StringWidth(cell), 0)
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

func safeRows(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = make([]string, len(r))
		for j, cell := range r {
			out[i][j] = safeText(cell)
		}
	}
	return out
}

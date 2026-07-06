package cli

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

func TestTable(t *testing.T) {
	var out strings.Builder
	table(&out, []string{"Spec", "Profiles"}, [][]string{
		{"work", "git, docker"},
		{"minimal", "core"},
	})

	got := out.String()
	for _, want := range []string{"Spec", "Profiles", "work", "git, docker", "┌", "┼", "┘"} {
		if !strings.Contains(got, want) {
			t.Errorf("table output missing %q, got:\n%s", want, got)
		}
	}

	// Every rendered line must have the same display width (rune count), which
	// is what keeps the grid aligned.
	var width int
	for i, line := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		w := utf8.RuneCountInString(line)
		if i == 0 {
			width = w
			continue
		}
		if w != width {
			t.Errorf("line %d width %d != %d:\n%s", i, w, width, got)
		}
	}
}

func TestTable_MultiByteAligns(t *testing.T) {
	var out strings.Builder
	// A multi-byte cell must not break alignment (rune-width, not byte-width).
	table(&out, []string{"Name", "X"}, [][]string{
		{"café", "1"},
		{"ab", "2"},
	})

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	width := utf8.RuneCountInString(lines[0])
	for i, line := range lines {
		if w := utf8.RuneCountInString(line); w != width {
			t.Errorf("line %d width %d != %d:\n%s", i, w, width, out.String())
		}
	}
}

func TestTable_ShortRowPadded(t *testing.T) {
	var out strings.Builder
	// A row with fewer cells than headers must still render without panicking.
	table(&out, []string{"A", "B", "C"}, [][]string{
		{"x"},
		{"y", "z"},
	})
	if !strings.Contains(out.String(), "│ x │") {
		t.Errorf("short row not padded correctly:\n%s", out.String())
	}
}

// Every rendered line must occupy the same number of terminal cells, even
// with emoji content — rune counting used to shift borders by one.
func TestTableEmojiAlignment(t *testing.T) {
	var sb strings.Builder
	table(&sb, []string{"ADDR", "NAME"}, [][]string{
		{"A0", "🧹 Maintenance"}, // true emoji: 1 rune, 2 cells
		{"A1", "⬆️ Update"},     // VS16 sequence: 2 runes, 2 cells
		{"A2", "plain"},
		{"A3", "啸龙志"}, // CJK wide
	})
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	want := ansi.StringWidth(lines[0])
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != want {
			t.Errorf("line %d width = %d, want %d: %q", i, got, want, line)
		}
	}
}

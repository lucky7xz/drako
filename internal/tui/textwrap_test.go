package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// A word wider than the line is cut into pieces that fit, and nothing is
// lost: a long URL in a command or a path in a description stays whole.
func TestWrapKeepsLongWordsWithinWidth(t *testing.T) {
	texts := []string{
		"curl -fsSL https://github.com/lucky7xz/drako/releases/latest/download/install.sh | sh",
		"see /home/user/.config/drako/inventory/a-very-long-profile-name.profile.toml first",
		"中文说明没有空格所以必须在任意字符之间换行",
		"🐉🐉🐉🐉🐉🐉🐉🐉 dragons",
	}
	wraps := map[string]func(string, int) []string{"WrapText": WrapText, "WrapScript": WrapScript}
	noSpace := func(s string) string { return strings.Join(strings.Fields(s), "") }
	for _, text := range texts {
		for _, width := range []int{4, 10, 24} {
			for name, wrap := range wraps {
				lines := wrap(text, width)
				for _, l := range lines {
					if w := lipgloss.Width(l); w > width {
						t.Errorf("%s(%q, %d): %q is %d wide", name, text, width, l, w)
					}
				}
				if got := noSpace(strings.Join(lines, "")); got != noSpace(text) {
					t.Errorf("%s(%q, %d) lost text: %q", name, text, width, lines)
				}
			}
		}
	}
}

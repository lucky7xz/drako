package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/settings"
	"github.com/muesli/termenv"
)

var sgrCode = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// colourAt is the colour escape in force where want starts in out: "" for
// none, which is the terminal's own text colour.
func colourAt(t *testing.T, out, want string) string {
	t.Helper()
	i := strings.Index(out, want)
	if i < 0 {
		t.Fatalf("%q is not on screen:\n%s", want, out)
	}
	codes := sgrCode.FindAllString(out[:i], -1)
	if len(codes) == 0 || codes[len(codes)-1] == "\x1b[0m" {
		return ""
	}
	return codes[len(codes)-1]
}

// The key hint is the line a new user needs (TH-01). On the terminal's own
// background it carries no colour, so it reads on a dark and on a light
// terminal; the version beside it stays dim.
func TestKeyHintIsNotDimmed(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(old)

	m, _ := editInventoryModel(t)
	m.termWidth, m.termHeight = 100, 30
	m.styles = BuildStyles(settings.Config{Theme: "dracula"}, "")

	if c := colourAt(t, m.renderProfileBar(90, "? help · q quit"), "? help"); c != "" {
		t.Errorf("grid: the hint is coloured %q, want the terminal's own text colour", c)
	}
	inv := m.viewInventoryMode("? help · q back")
	if c := colourAt(t, inv, "? help"); c != "" {
		t.Errorf("inventory: the hint is coloured %q, want the terminal's own text colour", c)
	}
	if c := colourAt(t, inv, "drako"); c == "" {
		t.Error("inventory: the version label should stay dim")
	}
}

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

// The logo's marks and the spinner were pure white, whatever the theme, and
// vanished on a light terminal. They carry no colour now.
func TestLogoMarksCarryNoColour(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(old)

	m := Model{}
	m.spinner = newSpinner()
	m.spinner.style = lipgloss.NewStyle()
	s := BuildStyles(settings.Config{Theme: "dracula"}, "")
	out := s.renderHeaderArt(m.spinner.View())
	if strings.Contains(out, "255;255;255") {
		t.Error("the logo still draws something in pure white")
	}
	if c := colourAt(t, out, "z"); strings.Contains(c, "38;2") {
		t.Errorf("the mark z is coloured %q, want the terminal's own text colour", c)
	}
	if c := colourAt(t, out, "╭"); !strings.Contains(c, "38;2") {
		t.Errorf("the frame should keep the theme's primary colour, got %q", c)
	}
}

// While the leader waits, the hint shows it, at the same width as before.
func TestKeyHintShowsTheWaitingLeader(t *testing.T) {
	m, _ := editInventoryModel(t)
	m.mode = gridMode
	m.Config.Keys.Leader, m.Config.Keys.Help = "m", "?"
	usual := m.keyHint()
	m.leader.pending = true
	waiting := m.keyHint()
	if !strings.HasPrefix(waiting, "m …") {
		t.Errorf("hint = %q, want it to show the waiting leader", waiting)
	}
	if lipgloss.Width(waiting) != lipgloss.Width(usual) {
		t.Errorf("hint widths differ (%q, %q): the footer would move", usual, waiting)
	}
}

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestOverlayCentersThePopupAndKeepsTheScreenAroundIt(t *testing.T) {
	base := "0123456789\nabcdefghij\nABCDEFGHIJ\nklmnopqrst"
	got := strings.Split(ansi.Strip(overlay(base, "XX\nYY", 10, 4)), "\n")
	want := []string{"0123456789", "abcdXXghij", "ABCDYYGHIJ", "klmnopqrst"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The renderer shows only the last h lines of a taller frame, so the popup is
// centered on those.
func TestOverlayCentersOnWhatTheTerminalShows(t *testing.T) {
	base := "cut1\ncut2\n....\n....\n....\n...."
	got := ansi.Strip(overlay(base, "XX", 4, 4))
	if want := "....\n.XX.\n....\n...."; got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestOverlayKeepsTheStyleRightOfThePopup(t *testing.T) {
	red := "\x1b[31m"
	row := strings.Split(overlay("..........\n"+red+"abcdefghij\x1b[0m\n..........", "XX", 10, 3), "\n")[1]
	if got := ansi.Strip(row); got != "abcdXXghij" {
		t.Fatalf("row reads %q", got)
	}
	right := row[strings.LastIndex(row, ansi.ResetStyle)+len(ansi.ResetStyle):]
	if !strings.HasPrefix(right, red) {
		t.Errorf("text right of the popup lost its colour: %q", right)
	}
	if w := ansi.StringWidth(row); w != 10 {
		t.Errorf("row is %d wide, want 10", w)
	}
}

func TestOverlayPadsShortLinesUpToThePopup(t *testing.T) {
	got := strings.Split(ansi.Strip(overlay("ab\nab\nab", "XX", 10, 3)), "\n")[1]
	if got != "ab  XX" {
		t.Errorf("got %q, want %q", got, "ab  XX")
	}
}

// A popup taller than the screen keeps its bottom, as the renderer would: its
// way out (the hint and the border) stays on screen, and the title goes first.
func TestOverlayKeepsTheBottomOfATallPopup(t *testing.T) {
	if got := ansi.Strip(overlay("1\n2\n3", "a\nb\nc\nd", 1, 3)); got != "b\nc\nd" {
		t.Errorf("got %q, want the popup's last three rows", got)
	}
}

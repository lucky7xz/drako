package ui

import (
	"strconv"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

// hexChannels parses "#RRGGBB" into its three byte channels.
func hexChannels(t *testing.T, hex string) (r, g, b uint64) {
	t.Helper()
	r, err1 := strconv.ParseUint(hex[1:3], 16, 8)
	g, err2 := strconv.ParseUint(hex[3:5], 16, 8)
	b, err3 := strconv.ParseUint(hex[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("not a valid #RRGGBB hex: %q", hex)
	}
	return r, g, b
}

func luminance(r, g, b uint64) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

// LockedFG must always be darker than Accent (the cursor color), never an
// unrelated color — otherwise a locked cell can clash with the cursor.
func TestLockedFGIsADarkerShadeOfAccent(t *testing.T) {
	for _, name := range []string{"dracula", "jade", "nord", "everforest", "orasaka", "dracula2", "viper"} {
		theme := config.GetTheme(name)
		ui := config.MapThemeToUI(theme)

		ar, ag, ab := hexChannels(t, ui.CursorFG)
		lr, lg, lb := hexChannels(t, ui.LockedFG)

		if luminance(lr, lg, lb) >= luminance(ar, ag, ab) {
			t.Errorf("%s: LockedFG %s is not darker than Accent %s", name, ui.LockedFG, ui.CursorFG)
		}
	}
}

// BuildStyles is pure, so we can assert it wires the right theme role onto the
// right style — the behavior the old side-effecting applyThemeStyles could not
// have covered.
func TestBuildStylesMapsThemeRolesToStyles(t *testing.T) {
	cfg := config.Config{Theme: "dracula"}
	ui := config.MapThemeToUI(config.GetTheme(cfg.Theme))
	s := BuildStyles(cfg)

	cases := []struct {
		name string
		got  lipgloss.TerminalColor
		want string
	}{
		{"Header", s.Header.GetForeground(), ui.HeaderFG},
		{"Online", s.Online.GetForeground(), ui.StatusPositive},
		{"Offline", s.Offline.GetForeground(), ui.StatusNegative},
		{"Title", s.Title.GetForeground(), ui.TitleFG},
		{"Footer", s.Footer.GetForeground(), ui.FooterFG},
		// Locked cells derive from Accent (the cursor color), not Warning.
		{"LockedCell", s.LockedCell.GetForeground(), ui.LockedFG},
		{"LockedSelectedCell", s.LockedSelectedCell.GetForeground(), ui.LockedFG},
		{"FlashedRowLabel", s.FlashedRowLabel.GetForeground(), ui.GridSelText},
	}
	for _, c := range cases {
		if c.got != lipgloss.Color(c.want) {
			t.Errorf("%s foreground = %v, want %s", c.name, c.got, c.want)
		}
	}

	// ColumnPendingCell tints only the border, not the text, so it's
	// asserted separately from the foreground-based cases above.
	borderCases := []struct {
		name string
		got  lipgloss.TerminalColor
		want string
	}{
		{"ColumnPendingCell", s.ColumnPendingCell.GetBorderTopForeground(), ui.GridPendingBorder},
	}
	for _, c := range borderCases {
		if c.got != lipgloss.Color(c.want) {
			t.Errorf("%s border = %v, want %s", c.name, c.got, c.want)
		}
	}
}

func TestBuildStylesResolvesHeaderArt(t *testing.T) {
	custom := "my custom art"
	blank := "   \n  "

	tests := []struct {
		name string
		art  *string
		want string
	}{
		{"nil falls back to default", nil, headerArt},
		{"whitespace falls back to default", &blank, headerArt},
		{"custom art is honored", &custom, custom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := BuildStyles(config.Config{Theme: "dracula", HeaderArt: tt.art})
			if s.HeaderArt != tt.want {
				t.Errorf("HeaderArt = %q, want %q", s.HeaderArt, tt.want)
			}
		})
	}
}

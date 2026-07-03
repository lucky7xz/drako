package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

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
	}
	for _, c := range cases {
		if c.got != lipgloss.Color(c.want) {
			t.Errorf("%s foreground = %v, want %s", c.name, c.got, c.want)
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

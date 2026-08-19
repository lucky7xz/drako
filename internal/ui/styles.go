package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

// Styling lives in one place so colors and layout tweaks are easy to reason about.

// appStyle is static chrome (not theme-derived), so it stays a package var.
var appStyle = lipgloss.NewStyle().Margin(1, 2)

// headerArt is the built-in default template, used when a profile sets no
// custom header_art. It is a constant, not mutable state.
const headerArt = `
╭───────────────────────────────╮
│   //┏━ ┓z       ┏━ ┓\\:...    │
│  ┏━━┫  ┣━━━┳━━━━┫  ┣━┓━━━━━┓  │
│  ┃  ✘  ┃ ┏━┫ ━━ ┃   ◄┃  %s  ┃  │
│  ┗━━━━━┻━┛ ┗━╱╲━┻━━┻━┻━━━━━┛  │
◄═══════════════════════════════►
     ◄═════[  啸龙志  ]═════►
╰───────────────────────────────╯
`

// Styles holds every theme-derived lipgloss.Style plus the resolved header
// art. It is built once per config by BuildStyles and carried on the Model, so
// rendering reads a value it was handed rather than mutable package globals.
type Styles struct {
	Header    lipgloss.Style
	Help      lipgloss.Style
	StatusBar lipgloss.Style
	Online    lipgloss.Style
	Offline   lipgloss.Style

	Cell         lipgloss.Style
	SelectedCell lipgloss.Style
	// Locked variants tint only the text, so the border keeps showing which
	// cell the cursor is on.
	LockedCell         lipgloss.Style
	LockedSelectedCell lipgloss.Style

	Path          lipgloss.Style
	SelectedPath  lipgloss.Style
	PathSeparator lipgloss.Style

	ChildDir         lipgloss.Style
	SelectedChildDir lipgloss.Style

	LockBadge      lipgloss.Style
	StatusPositive lipgloss.Style
	StatusNegative lipgloss.Style

	Title          lipgloss.Style
	InventoryTitle lipgloss.Style
	ListHeader     lipgloss.Style

	Item           lipgloss.Style
	SelectedItem   lipgloss.Style
	SelectedCursor lipgloss.Style

	Button               lipgloss.Style
	SelectedButton       lipgloss.Style
	RescueButton         lipgloss.Style
	SelectedRescueButton lipgloss.Style

	ErrorTitle    lipgloss.Style
	ErrorText     lipgloss.Style
	ThemeName     lipgloss.Style
	White         lipgloss.Style
	Footer        lipgloss.Style
	DropdownPopup lipgloss.Style

	// HeaderArt is the resolved header template: the profile's custom art, or
	// the built-in default when none is set.
	HeaderArt string
}

// BuildStyles resolves a config into a concrete Styles value. It is pure: same
// config in, same styles out, no globals touched — so it is unit-testable.
func BuildStyles(cfg config.Config) Styles {
	theme := config.GetTheme(cfg.Theme)
	ui := config.MapThemeToUI(theme)

	var s Styles

	if cfg.HeaderArt != nil && strings.TrimSpace(*cfg.HeaderArt) != "" {
		s.HeaderArt = *cfg.HeaderArt
		log.Printf("Using custom header art from config")
	} else {
		s.HeaderArt = headerArt
		log.Printf("Using default header art")
	}

	s.Header = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HeaderFG)).
		PaddingBottom(1).
		Bold(true)

	s.Help = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HelpFG))

	s.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusInfo)).
		PaddingTop(1)

	s.Online = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusPositive))

	s.Offline = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative))

	retroBorder := lipgloss.Border{
		Top:         "━",
		Bottom:      "━",
		Left:        " ",
		Right:       " ",
		TopLeft:     "┍",
		TopRight:    "┑",
		BottomLeft:  "┕",
		BottomRight: "┙",
	}

	s.Cell = lipgloss.NewStyle().
		Border(retroBorder).
		BorderForeground(lipgloss.Color(ui.GridBorder)).
		Bold(true).
		Padding(0, 1)

	s.SelectedCell = lipgloss.NewStyle().
		Border(retroBorder).
		BorderForeground(lipgloss.Color(ui.GridSelBorder)).
		Foreground(lipgloss.Color(ui.GridSelText)).
		Bold(true).
		Padding(0, 1)

	// Warning is the one role that is yellow in every shipped theme, so a
	// locked cell reads as "held back" rather than as another accent.
	s.LockedCell = s.Cell.Foreground(lipgloss.Color(ui.Warning))
	s.LockedSelectedCell = s.SelectedCell.Foreground(lipgloss.Color(ui.Warning))

	s.Path = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Path)).
		Padding(0, 1)

	s.SelectedPath = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.PathSelected)).
		Underline(true).
		Padding(0, 1)

	s.ChildDir = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color(ui.HelpFG))

	s.SelectedChildDir = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color(ui.GridSelText)).
		Bold(true)

	s.PathSeparator = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.PathSeparator)).
		Padding(0, 1)

	s.LockBadge = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Background)).
		Background(lipgloss.Color(theme.Primary)).
		Bold(true).
		Padding(0, 1).
		MarginLeft(1)

	s.StatusPositive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusPositive)).
		PaddingLeft(1)

	s.StatusNegative = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative)).
		PaddingLeft(1)

	s.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Bold(true).
		Padding(0, 1)

	s.InventoryTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(ui.GridBorder)).
		Padding(0, 1).
		MarginBottom(1)

	s.ListHeader = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ListHeaderFG)).
		Underline(true).
		Padding(0, 1)

	s.Item = lipgloss.NewStyle().
		Padding(0, 1)

	s.SelectedItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Padding(0, 1)

	s.SelectedCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.CursorFG))

	s.Button = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ButtonFG)).
		Background(lipgloss.Color(ui.ButtonBG)).
		Padding(0, 3).
		MarginTop(1)

	s.SelectedButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ButtonSelFG)).
		Background(lipgloss.Color(ui.ButtonSelBG)).
		Padding(0, 3).
		MarginTop(1)

	s.RescueButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative)).
		Background(lipgloss.Color(ui.ButtonBG)).
		Padding(0, 3).
		MarginTop(1)

	s.SelectedRescueButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ui.StatusNegative)).
		Padding(0, 3).
		MarginTop(1).
		Bold(true)

	s.ErrorTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative)).
		Bold(true)

	s.ErrorText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Warning))

	s.ThemeName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Bold(true)

	s.White = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	s.Footer = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.FooterFG)).
		PaddingTop(3).
		Align(lipgloss.Center)

	s.DropdownPopup = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(ui.DropdownBorder)).
		Background(lipgloss.Color(ui.DropdownBG)).
		Foreground(lipgloss.Color(ui.DropdownFG)).
		Padding(1, 2).
		Margin(1).
		Bold(true)

	return s
}

// renderHeaderArt renders the header with x and Chinese characters in white
func (s Styles) renderHeaderArt(spinnerView string) string {
	var formattedArt string
	placeholder := "SPINNERPLACEHOLDER"

	// Only inject the spinner if the template asks for it
	if strings.Contains(s.HeaderArt, "%s") {
		formattedArt = fmt.Sprintf(s.HeaderArt, placeholder)
	} else {
		formattedArt = s.HeaderArt
	}

	lines := strings.Split(formattedArt, "\n")

	// Get the primary color from the header style
	primaryStyle := lipgloss.NewStyle().
		Foreground(s.Header.GetForeground()).
		Bold(true)

	var styledLines []string
	for _, line := range lines {
		if line == "" {
			styledLines = append(styledLines, line)
			continue
		}

		// Check if this line contains the spinner placeholder
		if strings.Contains(line, placeholder) {
			// Process the line in parts: before placeholder, placeholder, after placeholder
			parts := strings.Split(line, placeholder)
			var styledLine strings.Builder

			// Process part before spinner
			if len(parts) > 0 {
				styledLine.WriteString(s.styleLineSegment(parts[0], primaryStyle))
			}

			// Add white-styled spinner
			styledLine.WriteString(s.White.Render(spinnerView))

			// Process part after spinner
			if len(parts) > 1 {
				styledLine.WriteString(s.styleLineSegment(parts[1], primaryStyle))
			}

			styledLines = append(styledLines, styledLine.String())
			continue
		}

		// Regular line without spinner
		styledLines = append(styledLines, s.styleLineSegment(line, primaryStyle))
	}

	// Apply only padding, not color
	result := strings.Join(styledLines, "\n")
	return lipgloss.NewStyle().PaddingBottom(1).Render(result)
}

// styleLineSegment applies styling to a line segment, with X and Chinese chars in white
func (s Styles) styleLineSegment(segment string, primaryStyle lipgloss.Style) string {
	var styledLine strings.Builder
	runes := []rune(segment)
	for i := 0; i < len(runes); i++ {
		// Check for 'X'
		if runes[i] == '✘' {
			styledLine.WriteString(s.White.Render("✘"))
			continue
		}

		// Check for "╱╲"
		if i+1 < len(runes) && string(runes[i:i+2]) == "╱╲" {
			styledLine.WriteString(s.White.Render("╱╲"))
			i++ // Skip next char (loop will increment by 1)
			continue
		}

		// Check for "◄"
		if i+1 < len(runes) && runes[i] == '◄' {
			styledLine.WriteString(s.White.Render("◄"))
			continue
		}

		// Check for "►"
		if i+1 < len(runes) && runes[i] == '►' {
			styledLine.WriteString(s.White.Render("►"))
			continue
		}

		// Check for "◄═══════════════════════════════►"
		pattern := "◄═══════════════════════════════►"
		if i+len([]rune(pattern))-1 < len(runes) && string(runes[i:i+len([]rune(pattern))]) == pattern {
			styledLine.WriteString(s.White.Render(pattern))
			i += len([]rune(pattern)) - 1 // Skip the matched runes
			continue
		}

		// Check for Chinese characters "啸龙志"
		if i+2 < len(runes) && string(runes[i:i+3]) == "啸龙志" {
			styledLine.WriteString(s.White.Render("啸龙志"))
			i += 2 // Skip next 2 chars (loop will increment by 1)
			continue
		}

		// Regular character with primary color
		styledLine.WriteString(primaryStyle.Render(string(runes[i])))
	}
	return styledLine.String()
}

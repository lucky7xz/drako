package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

// Styling lives in one place so colors and layout tweaks are easy to reason about.

var (
	appStyle = lipgloss.NewStyle().
			Margin(1, 2)

	headerArt = `
╭───────────────────────────────╮
│   //┏━ ┓z       ┏━ ┓\\:...    │
│  ┏━━┫  ╋━━━┳━━━━╋  ┣━┓━━━━━┓  │
│  ┃  ✘  ┃ ┏━┫ ━━ ┃   ◄┃  %s  ┃  │
│  ┗━━━━━┻━┛ ┗━╱╲━┻━━┻━┻━━━━━┛  │
◄═══════════════════════════════► 
     ◄═════[  啸龙志  ]═════►      
╰───────────────────────────────╯
`

	activeHeaderArt = ""

	headerStyle               lipgloss.Style
	helpStyle                 lipgloss.Style
	statusBarStyle            lipgloss.Style
	onlineStyle               lipgloss.Style
	offlineStyle              lipgloss.Style
	cellStyle                 lipgloss.Style
	selectedCellStyle         lipgloss.Style
	pathStyle                 lipgloss.Style
	selectedPathStyle         lipgloss.Style
	childDirStyle             lipgloss.Style
	selectedChildDirStyle     lipgloss.Style
	pathSeparatorStyle        lipgloss.Style
	lockBadgeStyle            lipgloss.Style
	statusPositiveStyle       lipgloss.Style
	statusNegativeStyle       lipgloss.Style
	titleStyle                lipgloss.Style
	inventoryTitleStyle       lipgloss.Style
	listHeaderStyle           lipgloss.Style
	itemStyle                 lipgloss.Style
	selectedItemStyle         lipgloss.Style
	selectedCursorStyle       lipgloss.Style
	buttonStyle               lipgloss.Style
	selectedButtonStyle       lipgloss.Style
	rescueButtonStyle         lipgloss.Style
	selectedRescueButtonStyle lipgloss.Style
	errorTitleStyle           lipgloss.Style
	errorTextStyle            lipgloss.Style
	themeNameStyle            lipgloss.Style
	whiteStyle                lipgloss.Style
	footerStyle               lipgloss.Style
	dropdownPopupStyle        lipgloss.Style
)

func applyThemeStyles(cfg config.Config) {
	theme := config.GetTheme(cfg.Theme)
	ui := config.MapThemeToUI(theme)

	if cfg.HeaderArt != nil && strings.TrimSpace(*cfg.HeaderArt) != "" {
		activeHeaderArt = *cfg.HeaderArt
		log.Printf("Using custom header art from config")
	} else {
		activeHeaderArt = headerArt
		log.Printf("Using default header art")
	}

	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HeaderFG)).
		PaddingBottom(1).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HelpFG))

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusInfo)).
		PaddingTop(1)

	onlineStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusPositive))

	offlineStyle = lipgloss.NewStyle().
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

	cellStyle = lipgloss.NewStyle().
		Border(retroBorder).
		BorderForeground(lipgloss.Color(ui.GridBorder)).
		Bold(true).
		Padding(0, 1)

	selectedCellStyle = lipgloss.NewStyle().
		Border(retroBorder).
		BorderForeground(lipgloss.Color(ui.GridSelBorder)).
		Foreground(lipgloss.Color(ui.GridSelText)).
		Bold(true).
		Padding(0, 1)

	pathStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Path)).
		Padding(0, 1)

	selectedPathStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.PathSelected)).
		Underline(true).
		Padding(0, 1)

	childDirStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color(ui.HelpFG))

	selectedChildDirStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color(ui.GridSelText)).
		Bold(true)

	pathSeparatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.PathSeparator)).
		Padding(0, 1)

	lockBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Background)).
		Background(lipgloss.Color(theme.Primary)).
		Bold(true).
		Padding(0, 1).
		MarginLeft(1)

	statusPositiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusPositive)).
		PaddingLeft(1)

	statusNegativeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative)).
		PaddingLeft(1)

	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Bold(true).
		Bold(true).
		Padding(0, 1)

	inventoryTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(ui.GridBorder)).
		Padding(0, 1).
		MarginBottom(1)

	listHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ListHeaderFG)).
		Underline(true).
		Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
		Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Padding(0, 1)

	selectedCursorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.CursorFG))

	buttonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ButtonFG)).
		Background(lipgloss.Color(ui.ButtonBG)).
		Padding(0, 3).
		MarginTop(1)

	selectedButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ButtonSelFG)).
		Background(lipgloss.Color(ui.ButtonSelBG)).
		Padding(0, 3).
		MarginTop(1)

	rescueButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative)).
		Background(lipgloss.Color(ui.ButtonBG)).
		Padding(0, 3).
		MarginTop(1)

	selectedRescueButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ui.StatusNegative)).
		Padding(0, 3).
		MarginTop(1).
		Bold(true)

	errorTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.StatusNegative)).
		Bold(true)

	errorTextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Warning))

	themeNameStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.TitleFG)).
		Bold(true)

	whiteStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	footerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.FooterFG)).
		PaddingTop(3).
		Align(lipgloss.Center)

	dropdownPopupStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(ui.DropdownBorder)).
		Background(lipgloss.Color(ui.DropdownBG)).
		Foreground(lipgloss.Color(ui.DropdownFG)).
		Padding(1, 2).
		Margin(1).
		Bold(true)
}

// renderHeaderArt renders the header with x and Chinese characters in white
func renderHeaderArt(spinnerView string) string {
	// Use a placeholder for the spinner
	placeholder := "SPINNERPLACEHOLDER"
	formattedArt := fmt.Sprintf(activeHeaderArt, placeholder)
	lines := strings.Split(formattedArt, "\n")

	// Get the primary color from headerStyle
	primaryStyle := lipgloss.NewStyle().
		Foreground(headerStyle.GetForeground()).
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
				styledLine.WriteString(styleLineSegment(parts[0], primaryStyle))
			}

			// Add white-styled spinner
			styledLine.WriteString(whiteStyle.Render(spinnerView))

			// Process part after spinner
			if len(parts) > 1 {
				styledLine.WriteString(styleLineSegment(parts[1], primaryStyle))
			}

			styledLines = append(styledLines, styledLine.String())
			continue
		}

		// Regular line without spinner
		styledLines = append(styledLines, styleLineSegment(line, primaryStyle))
	}

	// Apply only padding, not color
	result := strings.Join(styledLines, "\n")
	return lipgloss.NewStyle().PaddingBottom(1).Render(result)
}

// styleLineSegment applies styling to a line segment, with X and Chinese chars in white
func styleLineSegment(segment string, primaryStyle lipgloss.Style) string {
	var styledLine strings.Builder
	runes := []rune(segment)
	for i := 0; i < len(runes); i++ {
		// Check for 'X'
		if runes[i] == '✘' {
			styledLine.WriteString(whiteStyle.Render("✘"))
			continue
		}

		// Check for "╱╲"
		if i+1 < len(runes) && string(runes[i:i+2]) == "╱╲" {
			styledLine.WriteString(whiteStyle.Render("╱╲"))
			i++ // Skip next char (loop will increment by 1)
			continue
		}

		// Check for "◄"
		if i+1 < len(runes) && runes[i] == '◄' {
			styledLine.WriteString(whiteStyle.Render("◄"))
			continue
		}

		// Check for "►"
		if i+1 < len(runes) && runes[i] == '►' {
			styledLine.WriteString(whiteStyle.Render("►"))
			continue
		}

		// Check for "◄═══════════════════════════════►"
		pattern := "◄═══════════════════════════════►"
		if i+len([]rune(pattern))-1 < len(runes) && string(runes[i:i+len([]rune(pattern))]) == pattern {
			styledLine.WriteString(whiteStyle.Render(pattern))
			i += len([]rune(pattern)) - 1 // Skip the matched runes
			continue
		}

		// Check for Chinese characters "啸龙志"
		if i+2 < len(runes) && string(runes[i:i+3]) == "啸龙志" {
			styledLine.WriteString(whiteStyle.Render("啸龙志"))
			i += 2 // Skip next 2 chars (loop will increment by 1)
			continue
		}

		// Regular character with primary color
		styledLine.WriteString(primaryStyle.Render(string(runes[i])))
	}
	return styledLine.String()
}

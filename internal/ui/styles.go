package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

// Styling lives in one place so colors and layout tweaks are easy to reason about.

var (
	appStyle = lipgloss.NewStyle()

	headerArt = `
╭───────────────────────────────╮
│   //┏━ ┓z       ┏━ ┓\\:...    │
│  ┏━━┫  ╋━━━┳━━━━╋  ┣━┓━━━━━┓  │
│  ┃  ✘  ┃ ┏━┫ ━━ ┃   ◄┃  drako ┃  │
│  ┗━━━━━┻━┛ ┗━╱╲━┻━━┻━┻━━━━━┛  │
◄═══════════════════════════════► 
     ◄═════[  啸龙志  ]═════►      
╰───────────────────────────────╯
`

	headerStyle               lipgloss.Style
	profileButtonStyle        lipgloss.Style
	activeProfileButtonStyle  lipgloss.Style
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
	inactiveHeaderStyle       lipgloss.Style
)

func applyThemeStyles(cfg config.Config) {
	theme := config.GetTheme(cfg.Theme)
	ui := config.MapThemeToUI(theme)

	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HeaderFG)).
		PaddingBottom(1).
		Bold(true)

	profileButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HelpFG)).
		Bold(true)

	activeProfileButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ButtonSelFG)).
		Background(lipgloss.Color(ui.ButtonSelBG)).
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
		Padding(0, 1)

	inactiveHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.HelpFG)).
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

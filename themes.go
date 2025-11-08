package main

// DracoThemeConfig holds the color palette for a theme.
type DracoThemeConfig struct {
	Primary    string // Main brand color
	Secondary  string // Secondary accent color
	Background string // Main background
	Foreground string // Main text color
	Comment    string // Muted text, borders
	Success    string // For positive status
	Warning    string // For warnings
	Error      string // For errors
	Info       string // For informational messages
	Accent     string // For selected items, cursors
}

// themes is a map of available theme presets.
var themes = map[string]DracoThemeConfig{
	"dracula": {
		Primary:    "#ff2e63",
		Secondary:  "#ff8c00",
		Background: "#0d0221",
		Foreground: "#f0f0f0",
		Comment:    "#5c527f",
		Success:    "#00f5d4",
		Warning:    "#f9f871",
		Error:      "#ff2e63",
		Info:       "#00f5d4",
		Accent:     "#9d4edd",

	"jade": {
		Primary:    "#50fa7b",
		Secondary:  "#8be9fd",
		Background: "#282a36",
		Foreground: "#f8f8f2",
		Comment:    "#6272a4",
		Success:    "#50fa7b",
		Warning:    "#f1fa8c",
		Error:      "#ff5555",
		Info:       "#8be9fd",
		Accent:     "#50fa7b",
	},
	"nord": {
		Primary:    "#0077be",
		Secondary:  "#5e81ac",
		Background: "#0a192f",
		Foreground: "#e5e9f0",
		Comment:    "#4c566a",
		Success:    "#a3be8c",
		Warning:    "#ebcb8b",
		Error:      "#bf616a",
		Info:       "#88c0d0",
		Accent:     "#0077be",
	},
	"everforest": {
		Primary:    "#4a7c59",
		Secondary:  "#a7c080",
		Background: "#2d353b",
		Foreground: "#d3c6aa",
		Comment:    "#5c6a72",
		Success:    "#a7c080",
		Warning:    "#dbbc7f",
		Error:      "#e67e80",
		Info:       "#83c092",
		Accent:     "#4a7c59",
	},
	"orasaka": {
		Primary:    "#f5c2e7",
		Secondary:  "#cba6f7",
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",
		Comment:    "#585b70",
		Success:    "#a6e3a1",
		Warning:    "#f9e2af",
		Error:      "#f38ba8",
		Info:       "#89dceb",
		Accent:     "#f5c2e7",
	},
}

// UIColors describes concrete UI component colors derived from a theme.
type UIColors struct {
	HeaderFG        string
	FooterFG        string

	GridBorder      string
	GridSelBorder   string
	GridSelText     string

	Path            string
	PathSelected    string
	PathSeparator   string

	StatusInfo      string
	StatusPositive  string
	StatusNegative  string
	Warning         string

	HelpFG          string
	TitleFG         string
	ListHeaderFG    string
	CursorFG        string

	ButtonFG        string
	ButtonBG        string
	ButtonSelFG     string
	ButtonSelBG     string

	DropdownBorder  string
	DropdownFG      string
	DropdownBG      string
}

// mapThemeToUI maps a DracoThemeConfig to concrete UI component colors.
func mapThemeToUI(t DracoThemeConfig) UIColors {
	return UIColors{
		HeaderFG:       t.Primary,
		FooterFG:       t.Comment,

		GridBorder:     t.Comment,
		GridSelBorder:  t.Accent,
		GridSelText:    t.Accent,

		Path:           t.Primary,
		PathSelected:   t.Accent,
		PathSeparator:  t.Comment,

		StatusInfo:     t.Info,
		StatusPositive: t.Success,
		StatusNegative: t.Error,
		Warning:        t.Warning,

		HelpFG:         t.Comment,
		TitleFG:        t.Primary,
		ListHeaderFG:   t.Secondary,
		CursorFG:       t.Accent,

		ButtonFG:       t.Foreground,
		ButtonBG:       t.Comment,
		ButtonSelFG:    t.Background,
		ButtonSelBG:    t.Primary,

		DropdownBorder: t.Primary,
		DropdownFG:     t.Foreground,
		DropdownBG:     "#1a1a1a",
	}
}

// getTheme returns the color palette for a given theme name.
// If the theme is not found, it defaults to "dracula".
func getTheme(name string) DracoThemeConfig {
	if theme, ok := themes[name]; ok {
		return theme
	}
	return themes["dracula"]
}





















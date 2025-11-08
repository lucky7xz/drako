package main

import (
	"embed"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:embed bootstrap/themes.toml
var embeddedThemesFS embed.FS

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

var loadedThemes map[string]DracoThemeConfig

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(fmt.Sprintf("Failed to get user config directory: %v", err))
	}

	userThemesPath := filepath.Join(userConfigDir, "drako", "themes.toml")

	var themesContent []byte

	if _, err := os.Stat(userThemesPath); err == nil {
		// User-defined themes.toml exists, load from there
		themesContent, err = ioutil.ReadFile(userThemesPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to read user themes file %s: %v", userThemesPath, err))
		}
	} else if os.IsNotExist(err) {
		// User-defined themes.toml does not exist, load from embedded
		themesContent, err = embeddedThemesFS.ReadFile("bootstrap/themes.toml")
		if err != nil {
			panic(fmt.Sprintf("Failed to read embedded themes file: %v", err))
		}
	} else {
		// Other error checking user themes file
		panic(fmt.Sprintf("Error checking user themes file %s: %v", userThemesPath, err))
	}

	// Decode the TOML content
	if _, err := toml.Decode(string(themesContent), &loadedThemes); err != nil {
		panic(fmt.Sprintf("Failed to decode themes TOML: %v", err))
	}
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
	if theme, ok := loadedThemes[name]; ok {
		return theme
	}
	return loadedThemes["dracula"]
}





















package config

import (
	"embed"
	"log"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/paths"
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

// dracula is the built-in fallback theme, defined in code so it always exists.
var dracula = DracoThemeConfig{
	Primary:    "#ff2e63",
	Secondary:  "#ff8c00",
	Background: "#0d0221",
	Foreground: "#f0f0f0",
	Comment:    "#5c527f",
	Success:    "#00f5d4",
	Warning:    "#f9f871",
	Error:      "#ff2e63",
	Info:       "#00f5d4",
	Accent:     "#ff2e63",
}

var (
	loadedThemes map[string]DracoThemeConfig
	themesOnce   sync.Once
)

// loadThemes runs once, on first GetTheme. It never panics: a missing or
// malformed overlay is logged and skipped.
func loadThemes() {
	configDir, err := paths.ConfigDir()
	if err != nil {
		log.Printf("themes: no config dir, using built-in themes only: %v", err)
		configDir = ""
	}
	loadedThemes = buildThemes(configDir)
}

// buildThemes layers themes, later layers overriding by name: the dracula
// foundation, the embedded themes, then the user's themes.toml.
func buildThemes(configDir string) map[string]DracoThemeConfig {
	themes := map[string]DracoThemeConfig{"dracula": dracula}

	if data, err := embeddedThemesFS.ReadFile("bootstrap/themes.toml"); err == nil {
		mergeThemes(themes, data, "embedded themes")
	} else {
		log.Printf("themes: embedded themes unreadable: %v", err)
	}

	if configDir != "" {
		userPath := paths.ThemesFile(configDir)
		if data, err := os.ReadFile(userPath); err == nil {
			mergeThemes(themes, data, userPath)
		} else if !os.IsNotExist(err) {
			log.Printf("themes: could not read %s: %v", userPath, err)
		}
	}

	return themes
}

// mergeThemes decodes data and merges its themes into dst; a malformed
// document is logged and ignored.
func mergeThemes(dst map[string]DracoThemeConfig, data []byte, source string) {
	var parsed map[string]DracoThemeConfig
	if _, err := toml.Decode(string(data), &parsed); err != nil {
		log.Printf("themes: %s is malformed, ignoring it: %v", source, err)
		return
	}
	for name, theme := range parsed {
		dst[name] = theme
	}
}

// UIColors describes concrete UI component colors derived from a theme.
type UIColors struct {
	HeaderFG string
	FooterFG string

	GridBorder    string
	GridSelBorder string
	GridSelText   string

	Path          string
	PathSelected  string
	PathSeparator string

	StatusInfo     string
	StatusPositive string
	StatusNegative string
	Warning        string

	HelpFG       string
	TitleFG      string
	ListHeaderFG string
	CursorFG     string

	ButtonFG    string
	ButtonBG    string
	ButtonSelFG string
	ButtonSelBG string

	DropdownBorder string
	DropdownFG     string
	DropdownBG     string
}

// MapThemeToUI maps a DracoThemeConfig to concrete UI component colors.
func MapThemeToUI(t DracoThemeConfig) UIColors {
	return UIColors{
		HeaderFG: t.Primary,
		FooterFG: t.Comment,

		GridBorder:    t.Comment,
		GridSelBorder: t.Accent,
		GridSelText:   t.Accent,

		Path:          t.Primary,
		PathSelected:  t.Accent,
		PathSeparator: t.Comment,

		StatusInfo:     t.Info,
		StatusPositive: t.Success,
		StatusNegative: t.Error,
		Warning:        t.Warning,

		HelpFG:       t.Comment,
		TitleFG:      t.Primary,
		ListHeaderFG: t.Secondary,
		CursorFG:     t.Accent,

		ButtonFG:    t.Foreground,
		ButtonBG:    t.Comment,
		ButtonSelFG: t.Background,
		ButtonSelBG: t.Primary,

		DropdownBorder: t.Primary,
		DropdownFG:     t.Foreground,
		DropdownBG:     "#1a1a1a",
	}
}

// GetTheme returns the color palette for a given theme name.
// If the theme is not found, it defaults to "dracula".
func GetTheme(name string) DracoThemeConfig {
	themesOnce.Do(loadThemes)
	if theme, ok := loadedThemes[name]; ok {
		return theme
	}
	return dracula
}

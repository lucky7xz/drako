package config

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strconv"
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

	GridBorder        string
	GridSelBorder     string
	GridSelText       string
	GridPendingBorder string

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
	LockedFG     string

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

		GridBorder:        t.Comment,
		GridSelBorder:     t.Accent,
		GridSelText:       t.Accent,
		GridPendingBorder: t.Secondary,

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
		LockedFG:     darkenAccent(t.Accent, t.Background),

		ButtonFG:    t.Foreground,
		ButtonBG:    t.Comment,
		ButtonSelFG: t.Background,
		ButtonSelBG: t.Primary,

		DropdownBorder: t.Primary,
		DropdownFG:     t.Foreground,
		DropdownBG:     "#1a1a1a",
	}
}

// lockedAccentWeight blends Accent toward Background instead of toward
// black, so a dark shade doesn't shift hue (e.g. red toward orange).
const lockedAccentWeight = 0.45

// darkenAccent blends accent toward background at lockedAccentWeight.
func darkenAccent(accent, background string) string {
	ar, ag, ab, errA := hexRGB(accent)
	br, bg, bb, errB := hexRGB(background)
	if errA != nil || errB != nil {
		return accent
	}
	mix := func(a, b uint64) uint8 {
		return uint8(float64(a)*lockedAccentWeight + float64(b)*(1-lockedAccentWeight))
	}
	return fmt.Sprintf("#%02x%02x%02x", mix(ar, br), mix(ag, bg), mix(ab, bb))
}

// hexRGB parses a "#RRGGBB" string into its three channels.
func hexRGB(hex string) (r, g, b uint64, err error) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("not a #RRGGBB color: %q", hex)
	}
	r, errR := strconv.ParseUint(hex[1:3], 16, 8)
	g, errG := strconv.ParseUint(hex[3:5], 16, 8)
	b, errB := strconv.ParseUint(hex[5:7], 16, 8)
	if errR != nil || errG != nil || errB != nil {
		return 0, 0, 0, fmt.Errorf("not a #RRGGBB color: %q", hex)
	}
	return r, g, b, nil
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

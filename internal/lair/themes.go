package lair

import (
	"embed"
	"log"
	"os"

	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/settings"
)

//go:embed bootstrap/themes.toml
var embeddedThemesFS embed.FS

// buildThemes layers themes, later layers overriding by name: the dracula
// foundation, the embedded themes, then the user's themes.toml.
func buildThemes(configDir string) map[string]settings.DracoThemeConfig {
	themes := map[string]settings.DracoThemeConfig{"dracula": settings.DefaultTheme()}

	if data, err := embeddedThemesFS.ReadFile("bootstrap/themes.toml"); err == nil {
		settings.MergeThemes(themes, data, "embedded themes")
	} else {
		log.Printf("themes: embedded themes unreadable: %v", err)
	}

	if configDir != "" {
		userPath := paths.ThemesFile(configDir)
		if data, err := os.ReadFile(userPath); err == nil {
			settings.MergeThemes(themes, data, userPath)
		} else if !os.IsNotExist(err) {
			log.Printf("themes: could not read %s: %v", userPath, err)
		}
	}

	return themes
}

// GetTheme returns the named palette, or dracula. The layers are read on every
// call, so an edited themes.toml applies on the next reload.
func GetTheme(name string) settings.DracoThemeConfig {
	configDir, err := paths.ConfigDir()
	if err != nil {
		log.Printf("themes: no config dir, using built-in themes only: %v", err)
		configDir = ""
	}
	if theme, ok := buildThemes(configDir)[name]; ok {
		return theme
	}
	return settings.DefaultTheme()
}

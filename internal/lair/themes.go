package lair

import (
	"embed"
	"errors"
	"io/fs"
	"log"

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
		userPath := ThemesFile(configDir)
		if data, err := readThemes(configDir); err == nil {
			settings.MergeThemes(themes, data, userPath)
		} else if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("themes: could not read %s: %v", userPath, err)
		}
	}

	return themes
}

// readThemes returns the user's themes.toml; a config directory that does not
// exist yet has none.
func readThemes(configDir string) ([]byte, error) {
	l, err := Open(configDir)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return l.read(ThemesFileName)
}

// GetTheme returns the named palette, or dracula. The layers are read on every
// call, so an edited themes.toml applies on the next reload.
func GetTheme(name string) settings.DracoThemeConfig {
	configDir, err := ConfigDir()
	if err != nil {
		log.Printf("themes: no config dir, using built-in themes only: %v", err)
		configDir = ""
	}
	if theme, ok := buildThemes(configDir)[name]; ok {
		return theme
	}
	return settings.DefaultTheme()
}

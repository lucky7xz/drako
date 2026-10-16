// Package paths is the one place that knows where drako's files live; it only
// computes strings. Never build a path to a well-known location by hand.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// Well-known names under the drako config directory. They exist exactly
// once, in this block.
const (
	appDirName         = "drako"
	inventoryDirName   = "inventory"
	specsDirName       = "specs"
	assetsDirName      = "assets"
	trashDirName       = "trash"
	pivotFileName      = "pivot.toml"
	themesFileName     = "themes.toml"
	logFileName        = "drako.log"
	logArchiveName     = "drako.log.old"
	historyFileName    = "history.log"
	historyArchiveName = "history.log.old"
)

// ConfigFileName is exported because callers match the bare name (watcher
// events, error reports); names only used as full paths stay unexported.
const ConfigFileName = "config.toml"

// ConfigDir is drako's root config directory: ~/.config/drako on every OS, or
// $XDG_CONFIG_HOME/drako when that is set. It errors rather than guess.
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		if !filepath.IsAbs(xdg) {
			return "", fmt.Errorf("XDG_CONFIG_HOME must be an absolute path, got %q", xdg)
		}
		return filepath.Join(xdg, appDirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find your home directory: %w", err)
	}
	return filepath.Join(home, ".config", appDirName), nil
}

// InventoryDir is where stashed (non-equipped) profiles live.
func InventoryDir(configDir string) string {
	return filepath.Join(configDir, inventoryDirName)
}

// SpecsDir is where *.spec.toml files (profile bundles) live.
func SpecsDir(configDir string) string {
	return filepath.Join(configDir, specsDirName)
}

// AssetsDir is where the assets bundled with one profile live.
func AssetsDir(configDir, profileName string) string {
	return filepath.Join(configDir, assetsDirName, profileName)
}

// TrashDir is where purged files are moved instead of being deleted.
func TrashDir(configDir string) string {
	return filepath.Join(configDir, trashDirName)
}

// ConfigFile is drako's main configuration file.
func ConfigFile(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

// PivotFile records the locked profile and the equipped order.
func PivotFile(configDir string) string {
	return filepath.Join(configDir, pivotFileName)
}

// ThemesFile holds user-defined theme palettes; when absent, the themes
// embedded in the binary are used.
func ThemesFile(configDir string) string {
	return filepath.Join(configDir, themesFileName)
}

// LogFile is drako's application log.
func LogFile(configDir string) string {
	return filepath.Join(configDir, logFileName)
}

// HistoryFile records launched commands.
func HistoryFile(configDir string) string {
	return filepath.Join(configDir, historyFileName)
}

// LogArchive is the rotated backup of LogFile.
func LogArchive(configDir string) string {
	return filepath.Join(configDir, logArchiveName)
}

// HistoryArchive is the rotated backup of HistoryFile.
func HistoryArchive(configDir string) string {
	return filepath.Join(configDir, historyArchiveName)
}

// LogFiles is every log drako writes, rotated backups included: exactly what
// "drako purge --logs" deletes.
func LogFiles(configDir string) []string {
	return []string{
		LogFile(configDir),
		LogArchive(configDir),
		HistoryFile(configDir),
		HistoryArchive(configDir),
	}
}

// Package paths is the single authority on where drako's files live on disk.
//
// It is intentionally pure: every function only computes strings and never
// touches the filesystem. That keeps the package at the bottom of the
// dependency graph (it imports only the standard library), so any other
// package can use it without import cycles, and its tests need no real
// directories.
//
// The rule for the rest of the codebase: never build a path to a well-known
// drako location by hand. If a location matters, it has a name here.
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

// ConfigFileName is the bare filename of drako's main configuration file.
// Exported because some callers need the name itself (matching file-watcher
// events, addressing files relative to the config dir) rather than a full
// path. Names that are only ever used as full paths stay unexported.
const ConfigFileName = "config.toml"

// ConfigDir resolves drako's root config directory, e.g. ~/.config/drako
// on Linux.
//
// It defers the platform rules to os.UserConfigDir (XDG on Linux, AppData on
// Windows, Library/Application Support on macOS) and appends "drako". If the
// platform directory cannot be resolved, ConfigDir returns an error rather
// than guessing a location.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve the user config directory (check HOME / XDG_CONFIG_HOME): %w", err)
	}
	return filepath.Join(base, appDirName), nil
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

// LogFiles lists every log file drako may have written, including the
// rotated backups. This is the authoritative deletion list for
// "drako purge --logs".
func LogFiles(configDir string) []string {
	return []string{
		LogFile(configDir),
		LogArchive(configDir),
		HistoryFile(configDir),
		HistoryArchive(configDir),
	}
}

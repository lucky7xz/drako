package profiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucky7xz/drako/internal/paths"
)

// Location is one of drako's profile-file locations under the config dir.
type Location int

const (
	Equipped  Location = iota // config dir root: profiles the TUI loads
	Inventory                 // …/inventory: stashed profiles
	Trash                     // …/trash: purged profiles
)

// Dir returns the on-disk directory for the location under configDir.
func (l Location) Dir(configDir string) string {
	switch l {
	case Inventory:
		return paths.InventoryDir(configDir)
	case Trash:
		return paths.TrashDir(configDir)
	default:
		return configDir
	}
}

// Move relocates file (a bare filename) from one location to another under
// configDir.
func Move(configDir, file string, from, to Location) error {
	src := filepath.Join(from.Dir(configDir), file)
	dstDir := to.Dir(configDir)
	dst := filepath.Join(dstDir, file)
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

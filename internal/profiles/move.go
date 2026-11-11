package profiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/paths"
)

// locationDir returns the on-disk directory for the location under configDir.
func locationDir(l equip.Location, configDir string) string {
	switch l {
	case equip.Inventory:
		return paths.InventoryDir(configDir)
	case equip.Trash:
		return paths.TrashDir(configDir)
	default:
		return configDir
	}
}

// Move relocates file (a bare filename) from one location to another under
// configDir.
func Move(configDir, file string, from, to equip.Location) error {
	src := filepath.Join(locationDir(from, configDir), file)
	dstDir := locationDir(to, configDir)
	dst := filepath.Join(dstDir, file)
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	// A relative link (stow makes these) resolves from its own folder, so re-aim it.
	if target, err := os.Readlink(src); err == nil && !filepath.IsAbs(target) {
		if target, err = filepath.Rel(dstDir, filepath.Join(filepath.Dir(src), target)); err != nil {
			return err
		}
		if err := os.Symlink(target, dst); err != nil {
			return err
		}
		return os.Remove(src)
	}
	return os.Rename(src, dst)
}

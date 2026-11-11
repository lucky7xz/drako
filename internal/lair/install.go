package lair

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/lucky7xz/drako/internal/equip"
)

// Installing puts material that arrived from elsewhere (summon) into the
// config directory. The caller hands over content and a bare name, never a
// destination path: where it goes is worked out here, inside the root.

// InstallProfile writes what r holds as the profile called file, equipped or
// in the inventory, replacing one already there. A file that was a link
// becomes a plain file; what the link pointed at is not touched.
func InstallProfile(configDir string, loc equip.Location, file string, r io.Reader) error {
	if loc != equip.Equipped && loc != equip.Inventory {
		return fmt.Errorf("a profile is installed equipped or in the inventory, nowhere else")
	}
	if err := bareName(file); err != nil {
		return err
	}
	return install(configDir, filepath.Join(folder(loc), file), r)
}

// InstallSpec writes what r holds as the spec called file in the specs folder.
func InstallSpec(configDir, file string, r io.Reader) error {
	if err := bareName(file); err != nil {
		return err
	}
	return install(configDir, filepath.Join(SpecsDirName, file), r)
}

// InstallAsset writes what r holds as rel (e.g. "scripts/run.sh") in the
// assets folder of the profile called profileName.
func InstallAsset(configDir, profileName, rel string, r io.Reader) error {
	if err := bareName(profileName); err != nil {
		return err
	}
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("asset path %q does not stay inside the profile's assets", rel)
	}
	return install(configDir, filepath.Join(assetsDirName, profileName, rel), r)
}

// MakeInventory makes sure the inventory folder exists.
func MakeInventory(configDir string) error {
	return makeFolder(configDir, InventoryDirName)
}

// MakeSpecs makes sure the specs folder exists.
func MakeSpecs(configDir string) error {
	return makeFolder(configDir, SpecsDirName)
}

func install(configDir, name string, r io.Reader) error {
	l, err := Open(configDir)
	if err != nil {
		return err
	}
	defer l.Close()
	if err := l.mkdirAll(filepath.Dir(name)); err != nil {
		return err
	}
	return l.replaceFrom(name, r)
}

func makeFolder(configDir, name string) error {
	l, err := Create(configDir)
	if err != nil {
		return err
	}
	defer l.Close()
	return l.mkdirAll(name)
}

// bareName accepts a plain file or folder name: no separators, no "..".
func bareName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return fmt.Errorf("%q is not a plain name", name)
	}
	return nil
}

package config

import (
	"embed"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucky7xz/drako/internal/paths"
)

//go:embed all:bootstrap
var bootstrapFS embed.FS

// RestoreBootstrap writes every embedded bootstrap file that is missing from
// dstRoot and returns the relative paths it created. It never overwrites an
// existing file, so it is safe to run any time: first-run bootstrap uses it to
// populate an empty config dir, and "drako restore-bootstrap" uses it to put
// back whatever the user deleted (the core deck, starter decks like ssh-101,
// themes, specs). The returned paths are config-dir-relative ("core.profile.toml",
// "inventory/ssh-101.profile.toml", …); an empty slice means nothing was missing.
func RestoreBootstrap(dstRoot string) ([]string, error) {
	log.Printf("bootstrap: restoring missing embedded files into %s", dstRoot)
	var restored []string

	// The config dir may not exist yet (a direct `drako restore-bootstrap`
	// runs before any TUI-side MkdirAll), and the root-level files below are
	// written straight into it.
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return restored, err
	}

	// 1. config.toml (from the settings template).
	if wrote, err := WriteSettingsTemplate(dstRoot); err != nil {
		return restored, err
	} else if wrote {
		restored = append(restored, paths.ConfigFileName)
		log.Printf("bootstrap: generated config.toml")
	}

	// 2. core.profile.toml (the default deck).
	if created, err := RestoreCoreProfile(dstRoot); err != nil {
		return restored, err
	} else if created {
		restored = append(restored, "core.profile.toml")
		log.Printf("bootstrap: generated core.profile.toml")
	}

	// 3. Everything else the binary ships (themes, inventory decks, specs, …).
	if _, err := fs.ReadDir(bootstrapFS, "bootstrap"); err != nil {
		log.Printf("bootstrap: no embedded assets found")
		return restored, nil
	}
	err := fs.WalkDir(bootstrapFS, "bootstrap", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel := strings.TrimPrefix(path, "bootstrap")
		rel = strings.TrimPrefix(rel, "/")

		// Skip files handled above or not meant to be materialized on disk.
		if rel == "settings_template.toml" || rel == "config.toml" || rel == "core.profile.toml" {
			return nil
		}

		if rel == "" {
			return nil // Root dir
		}

		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		// Never duplicate a deck the user has moved: a profile shipped under
		// inventory/ counts as present whether stashed or equipped (config root).
		if strings.HasSuffix(rel, ".profile.toml") && profileDeckPresent(dstRoot, filepath.Base(rel)) {
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		b, err := bootstrapFS.ReadFile(path)
		if err != nil {
			return err
		}
		if wrote, err := writeNewFile(target, b); err != nil || !wrote {
			return err
		}
		// Report with forward slashes regardless of OS separator.
		restored = append(restored, filepath.ToSlash(rel))
		return nil
	})
	return restored, err
}

// WriteSettingsTemplate writes config.toml from the settings template when it
// is missing, and reports whether it wrote one.
func WriteSettingsTemplate(dstRoot string) (bool, error) {
	settings, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		return false, err
	}
	return writeNewFile(paths.ConfigFile(dstRoot), settings)
}

// RestoreCoreProfile writes the embedded default deck as core.profile.toml
// into dstRoot. The deck uses per-platform command variants, so one file
// serves every OS — resolution happens at load time. It never overwrites an
// existing file; the bool reports whether a new file was created. It is the
// core-deck slice of RestoreBootstrap, kept separate so bootstrap can report
// the core deck distinctly.
func RestoreCoreProfile(dstRoot string) (bool, error) {
	deck, err := bootstrapFS.ReadFile("bootstrap/core.profile.toml")
	if err != nil {
		return false, err
	}
	if profileDeckPresent(dstRoot, "core.profile.toml") {
		return false, nil
	}
	return writeNewFile(filepath.Join(dstRoot, "core.profile.toml"), deck)
}

// writeNewFile creates path holding data, or reports false when anything is
// there already; O_EXCL keeps "never overwrite" true under races and symlinks.
func writeNewFile(path string, data []byte) (bool, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err == nil, err
}

// profileDeckPresent reports whether a bootstrap-shipped deck already lives in
// either place a deck can be: equipped (config root) or stashed (inventory/).
// Restore uses it so a deck the user has moved is never duplicated back.
func profileDeckPresent(dstRoot, base string) bool {
	if _, err := os.Stat(filepath.Join(dstRoot, base)); err == nil {
		return true // equipped
	}
	if _, err := os.Stat(filepath.Join(paths.InventoryDir(dstRoot), base)); err == nil {
		return true // stashed
	}
	return false
}

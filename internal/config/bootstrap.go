package config

import (
	"embed"
	"io"
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
// back whatever the user deleted (the core deck, starter decks like ssh-utils,
// themes, specs). The returned paths are config-dir-relative ("core.profile.toml",
// "inventory/ssh-utils.profile.toml", …); an empty slice means nothing was missing.
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
	settings, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		log.Printf("bootstrap warning: settings_template.toml not found: %v", err)
	} else {
		targetConfig := paths.ConfigFile(dstRoot)
		if _, err := os.Stat(targetConfig); os.IsNotExist(err) {
			if err := os.WriteFile(targetConfig, settings, 0o644); err != nil {
				return restored, err
			}
			restored = append(restored, paths.ConfigFileName)
			log.Printf("bootstrap: generated config.toml")
		}
	}

	// 2. core.profile.toml (the default deck).
	if created, err := RestoreCoreProfile(dstRoot); err != nil {
		log.Printf("bootstrap error: failed to write core profile: %v", err)
	} else if created {
		restored = append(restored, "core.profile.toml")
		log.Printf("bootstrap: generated core.profile.toml")
	}

	// 3. Everything else the binary ships (themes, inventory decks, specs, …).
	if _, err := fs.ReadDir(bootstrapFS, "bootstrap"); err != nil {
		log.Printf("bootstrap: no embedded assets found")
		return restored, nil
	}
	err = fs.WalkDir(bootstrapFS, "bootstrap", func(path string, d fs.DirEntry, err error) error {
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

		// Safety: never overwrite an existing file.
		if _, err := os.Stat(target); err == nil {
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		r, err := bootstrapFS.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		b, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return err
		}
		// Report with forward slashes regardless of OS separator.
		restored = append(restored, filepath.ToSlash(rel))
		return nil
	})
	return restored, err
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
	targetProfile := filepath.Join(dstRoot, "core.profile.toml")
	if _, err := os.Stat(targetProfile); err == nil {
		return false, nil
	}
	if err := os.WriteFile(targetProfile, deck, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

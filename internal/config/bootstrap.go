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

func bootstrapCopy(dstRoot string) error {
	log.Printf("bootstrap: running embedded copy (build tag active)")

	// 1. Write config.toml (Settings)
	settings, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		log.Printf("bootstrap warning: settings_template.toml not found: %v", err)
	} else {
		targetConfig := paths.ConfigFile(dstRoot)
		if _, err := os.Stat(targetConfig); os.IsNotExist(err) {
			if err := os.WriteFile(targetConfig, settings, 0o644); err != nil {
				return err
			}
			log.Printf("bootstrap: generated config.toml")
		}
	}

	// 2. Write core.profile.toml (Core Profile)
	if created, err := RestoreCoreProfile(dstRoot); err != nil {
		log.Printf("bootstrap error: failed to write core profile: %v", err)
	} else if created {
		log.Printf("bootstrap: generated core.profile.toml")
	}

	// 3. Copy other files
	if _, err := fs.ReadDir(bootstrapFS, "bootstrap"); err != nil {
		log.Printf("bootstrap: no embedded assets found")
		return nil
	}
	return fs.WalkDir(bootstrapFS, "bootstrap", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel := strings.TrimPrefix(path, "bootstrap")
		rel = strings.TrimPrefix(rel, "/")

		// Skip files we handle specially or don't want to expose
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

		// Safety: Don't overwrite existing files
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
		return os.WriteFile(target, b, 0o644)
	})
}

// RestoreCoreProfile writes the embedded default deck as core.profile.toml
// into dstRoot. The deck uses per-platform command variants, so one file
// serves every OS — resolution happens at load time. It never overwrites an
// existing file; the bool reports whether a new file was created. Used by
// first-run bootstrap and by "drako restore-core" (the rescue grid's way back
// once the core deck has been stashed or deleted).
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

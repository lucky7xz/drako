package config

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed all:bootstrap
var bootstrapFS embed.FS

func bootstrapCopy(dstRoot string) error {
	log.Printf("bootstrap: running embedded copy (build tag active)")

	// 1. Weave and write config.toml
	tmpl, err := bootstrapFS.ReadFile("bootstrap/core_template.toml")
	if err != nil {
		log.Printf("bootstrap warning: core_template.toml not found: %v", err)
	}
	dict, err := bootstrapFS.ReadFile("bootstrap/core_dictionary.toml")
	if err != nil {
		log.Printf("bootstrap warning: core_dictionary.toml not found: %v", err)
	}

	if tmpl != nil && dict != nil {
		woven, err := WeaveConfig(tmpl, dict)
		if err != nil {
			log.Printf("bootstrap error: failed to weave config: %v", err)
		} else {
			targetConfig := filepath.Join(dstRoot, "config.toml")
			// Only write if not exists or if we are bootstrapping (checked by caller essentially, but let's be safe)
			// The caller (LoadConfig) checks if config.toml exists. If not, it calls bootstrapCopy.
			if err := os.WriteFile(targetConfig, woven, 0o644); err != nil {
				return err
			}
			log.Printf("bootstrap: generated config.toml for runtime: %s", runtime.GOOS)
		}
	}

	// 2. Copy other files
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
		if rel == "core_template.toml" || rel == "core_dictionary.toml" || rel == "config.toml" {
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

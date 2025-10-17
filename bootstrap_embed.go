
package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:bootstrap
var bootstrapFS embed.FS

func bootstrapCopy(dstRoot string) error {
	log.Printf("bootstrap: running embedded copy (build tag active)")
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
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
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
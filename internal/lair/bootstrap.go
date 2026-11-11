package lair

import (
	"embed"
	"errors"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
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
//
// A subfolder that is a link leaving the directory is skipped, the rest is
// still restored, and the error says which folder it was (ErrLinkedFolder).
func RestoreBootstrap(dstRoot string) ([]string, error) {
	log.Printf("bootstrap: restoring missing embedded files into %s", dstRoot)

	// The config dir may not exist yet (a direct `drako restore-bootstrap`
	// runs before any TUI-side MkdirAll), and the root-level files below are
	// written straight into it.
	l, err := Create(dstRoot)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return l.restoreBootstrap()
}

// WriteSettingsTemplate writes config.toml from the settings template when it
// is missing, and reports whether it wrote one.
func WriteSettingsTemplate(dstRoot string) (bool, error) {
	l, err := Open(dstRoot)
	if err != nil {
		return false, err
	}
	defer l.Close()
	return l.writeSettingsTemplate()
}

// RestoreCoreProfile writes the embedded default deck as core.profile.toml
// into dstRoot. The deck uses per-platform command variants, so one file
// serves every OS — resolution happens at load time. It never overwrites an
// existing file; the bool reports whether a new file was created. It is the
// core-deck slice of RestoreBootstrap, kept separate so bootstrap can report
// the core deck distinctly.
func RestoreCoreProfile(dstRoot string) (bool, error) {
	l, err := Open(dstRoot)
	if err != nil {
		return false, err
	}
	defer l.Close()
	return l.restoreCoreProfile()
}

func (l *Lair) restoreBootstrap() ([]string, error) {
	var restored []string

	// 1. config.toml (from the settings template).
	if wrote, err := l.writeSettingsTemplate(); err != nil {
		return restored, err
	} else if wrote {
		restored = append(restored, ConfigFileName)
		log.Printf("bootstrap: generated config.toml")
	}

	// 2. core.profile.toml (the default deck).
	if created, err := l.restoreCoreProfile(); err != nil {
		return restored, err
	} else if created {
		restored = append(restored, coreProfileFile)
		log.Printf("bootstrap: generated core.profile.toml")
	}

	// 3. Everything else the binary ships (themes, inventory decks, specs, …).
	if _, err := fs.ReadDir(bootstrapFS, "bootstrap"); err != nil {
		log.Printf("bootstrap: no embedded assets found")
		return restored, nil
	}
	var linked error // the first linked subfolder skipped, reported once the rest is done
	err := fs.WalkDir(bootstrapFS, "bootstrap", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel := strings.TrimPrefix(path, "bootstrap")
		rel = strings.TrimPrefix(rel, "/")

		// Skip files handled above or not meant to be materialized on disk.
		if rel == "settings_template.toml" || rel == "config.toml" || rel == coreProfileFile {
			return nil
		}

		if rel == "" {
			return nil // Root dir
		}

		target := filepath.FromSlash(rel)
		if d.IsDir() {
			err := l.mkdirAll(target)
			if errors.Is(err, ErrLinkedFolder) {
				log.Printf("bootstrap: skipping %s: %v", rel, err)
				if linked == nil {
					linked = err
				}
				return fs.SkipDir
			}
			return err
		}

		// Never duplicate a deck the user has moved: a profile shipped under
		// inventory/ counts as present whether stashed or equipped (config root).
		if strings.HasSuffix(rel, ".profile.toml") && l.profileDeckPresent(filepath.Base(rel)) {
			return nil
		}

		b, err := bootstrapFS.ReadFile(path)
		if err != nil {
			return err
		}
		if wrote, err := l.writeNew(target, b); err != nil || !wrote {
			return err
		}
		// Report with forward slashes regardless of OS separator.
		restored = append(restored, filepath.ToSlash(rel))
		return nil
	})
	if err == nil {
		err = linked
	}
	return restored, err
}

// coreProfileFile is the default deck's filename.
const coreProfileFile = "core.profile.toml"

func (l *Lair) writeSettingsTemplate() (bool, error) {
	settings, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		return false, err
	}
	return l.writeNew(ConfigFileName, settings)
}

func (l *Lair) restoreCoreProfile() (bool, error) {
	deck, err := bootstrapFS.ReadFile("bootstrap/" + coreProfileFile)
	if err != nil {
		return false, err
	}
	if l.profileDeckPresent(coreProfileFile) {
		return false, nil
	}
	return l.writeNew(coreProfileFile, deck)
}

// profileDeckPresent reports whether a bootstrap-shipped deck already lives in
// either place a deck can be: equipped (config root) or stashed (inventory/).
// Restore uses it so a deck the user has moved is never duplicated back.
func (l *Lair) profileDeckPresent(base string) bool {
	if _, err := l.stat(base); err == nil {
		return true // equipped
	}
	if _, err := l.stat(filepath.Join(InventoryDirName, base)); err == nil {
		return true // stashed
	}
	return false
}

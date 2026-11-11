// Package summon fetches profiles from elsewhere (a Git repository, a file
// over HTTP, a folder on disk) and inspects what arrived: it verifies pins,
// finds the files, validates them and plans their assets. It never writes
// into drako's config folder and never talks to the user; installing and
// asking are its caller's jobs.
package summon

import (
	"os"
	"path/filepath"
	"strings"
)

// FindFiles lists the .profile.toml and .spec.toml files at the top level
// of root; subfolders are not searched.
func FindFiles(root string) (profileFiles, specFiles []string, err error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}
	for _, de := range entries {
		// Regular files only: a symlink could point outside the repo or at a device.
		if !de.Type().IsRegular() {
			continue
		}
		path := filepath.Join(root, de.Name())
		if strings.HasSuffix(path, ".profile.toml") {
			profileFiles = append(profileFiles, path)
		} else if strings.HasSuffix(path, ".spec.toml") {
			specFiles = append(specFiles, path)
		}
	}
	return profileFiles, specFiles, nil
}

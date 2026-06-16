package core

import (
	"log"
	"os"
)

// RotateLogIfNeeded renames path to archivePath if path exceeds maxBytes,
// overwriting any previous backup. The caller supplies archivePath (from
// paths) so the backup-naming convention lives in one place.
func RotateLogIfNeeded(path, archivePath string, maxBytes int64) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		// If we can't stat, we can't check size. Just return.
		return
	}

	if info.Size() > maxBytes {
		// Best effort remove old backup
		_ = os.Remove(archivePath)

		if err := os.Rename(path, archivePath); err != nil {
			log.Printf("Failed to rotate log %s: %v", path, err)
		}
	}
}

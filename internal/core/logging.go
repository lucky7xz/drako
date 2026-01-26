package core

import (
	"log"
	"os"
)

// RotateLogIfNeeded checks if the log file at path exceeds maxBytes.
// If it does, it renames the file to path + ".old" (overwriting any previous backup).
func RotateLogIfNeeded(path string, maxBytes int64) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		// If we can't stat, we can't check size. Just return.
		return
	}

	if info.Size() > maxBytes {

		oldPath := path + ".old"
		// Best effort remove old backup
		_ = os.Remove(oldPath)

		if err := os.Rename(path, oldPath); err != nil {
			log.Printf("Failed to rotate log %s: %v", path, err)
		}
	}
}

package core

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lucky7xz/drako/internal/paths"
)

// LogExecution appends one command run to history.log in the config dir:
// "[timestamp] <selected> (exec: <what actually ran>)". Best-effort — logging
// failures are logged themselves but never stop an execution.
func LogExecution(selected, executed string) {
	configDir, err := paths.ConfigDir()
	if err != nil {
		log.Printf("logging error: could not get config dir: %v", err)
		return
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		log.Printf("logging error: could not create config dir: %v", err)
		return
	}

	logPath := paths.HistoryFile(configDir)
	RotateLogIfNeeded(logPath, paths.HistoryArchive(configDir), 1024*1024)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("logging error: could not open history.log: %v", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s (exec: %s)\n", timestamp, selected, executed)
	if _, err := f.WriteString(entry); err != nil {
		log.Printf("logging error: could not write to history.log: %v", err)
	}
}

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

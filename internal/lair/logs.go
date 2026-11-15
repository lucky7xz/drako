package lair

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"
)

// logCap is the size past which a log is rotated to its archive.
const logCap = 1024 * 1024

// OpenLog opens drako.log for appending, rotating it first once it passes 1 MB.
// The TUI and every CLI command log through it.
func (l *Lair) OpenLog() (*os.File, error) {
	l.rotate(LogFileName, LogArchiveName, logCap)
	return l.root.OpenFile(LogFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
}

// LogExecution appends one command run to history.log:
// "[timestamp] <selected> (exec: <what actually ran>)". Best-effort — logging
// failures are logged themselves but never stop an execution.
func (l *Lair) LogExecution(selected, executed string) {
	l.rotate(HistoryFileName, HistoryArchiveName, logCap)

	// Full command lines can hold secrets: owner-only, like shell history.
	f, err := l.root.OpenFile(HistoryFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("logging error: could not open history.log: %v", err)
		return
	}
	defer f.Close()
	_ = f.Chmod(0o600) // a file from before this change keeps its old mode otherwise

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s (exec: %s)\n", timestamp, selected, executed)
	if _, err := f.WriteString(entry); err != nil {
		log.Printf("logging error: could not write to history.log: %v", err)
	}
}

// rotate renames the log called name to archive once it exceeds maxBytes,
// overwriting any previous archive. A log that is missing or unreadable is
// left alone.
func (l *Lair) rotate(name, archive string, maxBytes int64) {
	info, err := l.root.Stat(name)
	if err != nil {
		return
	}
	if info.Size() > maxBytes {
		// Best effort remove old backup
		_ = l.root.Remove(archive)

		if err := l.root.Rename(name, archive); err != nil {
			log.Printf("Failed to rotate log %s: %v", name, err)
		}
	}
}

// LogFileNames is every log drako keeps, archives included.
var LogFileNames = []string{LogFileName, LogArchiveName, HistoryFileName, HistoryArchiveName}

// DeleteLogs deletes drako's logs and their archives and names those it
// deleted. A log that is not there is skipped; one that is a link loses the
// link, never what it points to.
func DeleteLogs(configDir string) (deleted []string, err error) {
	l, err := Open(configDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer l.Close()
	var failed []error
	for _, name := range LogFileNames {
		switch err := l.root.Remove(name); {
		case err == nil:
			deleted = append(deleted, name)
		case errors.Is(err, fs.ErrNotExist):
		default:
			failed = append(failed, fmt.Errorf("could not delete %s: %w", name, err))
		}
	}
	return deleted, errors.Join(failed...)
}

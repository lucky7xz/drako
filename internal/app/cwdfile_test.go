package app

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCwdFile runs last, after the log file has closed (defers run in reverse).
// A failure must still reach the user, since the wrapper then cannot cd.
func TestWriteCwdFileFailureReachesTheUser(t *testing.T) {
	logFile, err := os.CreateTemp(t.TempDir(), "drako.log")
	if err != nil {
		t.Fatal(err)
	}
	logFile.Close()
	log.SetOutput(logFile) // as at exit: the log file is already closed
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	r, w, _ := os.Pipe()
	oldStderr := os.Stderr
	os.Stderr = w
	writeCwdFile(filepath.Join(t.TempDir(), "no-such-dir", "cwd"))
	os.Stderr = oldStderr
	w.Close()
	out, _ := io.ReadAll(r)

	if !strings.Contains(string(out), "--cwd-file") {
		t.Errorf("the failure was lost; stderr got %q", out)
	}
}

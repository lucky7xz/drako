package app

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestFailedTextNamesTheExitCode(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 3").Run()
	if got := failedText(err); !strings.Contains(got, "--- Command Failed (exit 3) ---") {
		t.Errorf("exit 3: got %q", got)
	}
	// No exit code to name: the reason is given instead.
	if got := failedText(errors.New("no such shell")); !strings.Contains(got, "Error: no such shell") {
		t.Errorf("other error: got %q", got)
	}
}

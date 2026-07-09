package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCheckFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const cleanProfile = `x = 1
y = 1
[[commands]]
name = "ok"
command = "echo ok"
row = 0
col = "a"
`

func TestHandleCheckCommand_ExitCodes(t *testing.T) {
	dir := t.TempDir()
	bad := writeCheckFile(t, dir, "bad.profile.toml", "x = not toml at all [")
	good := writeCheckFile(t, dir, "good.profile.toml", cleanProfile)
	empty := t.TempDir()

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"error findings exit 1 (CI contract)", []string{"drako", "check", bad}, 1},
		{"clean file exits 0", []string{"drako", "check", good}, 0},
		{"unreadable path exits 1", []string{"drako", "check", filepath.Join(dir, "nope")}, 1},
		{"empty dir exits 0", []string{"drako", "check", empty}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HandleCheckCommand(tt.args); got != tt.want {
				t.Errorf("HandleCheckCommand(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunChecksTableOutput(t *testing.T) {
	dir := t.TempDir()
	bad := writeCheckFile(t, dir, "bad.profile.toml", "x = not toml at all [")
	good := writeCheckFile(t, dir, "good.profile.toml", cleanProfile)

	var out strings.Builder
	errs := runChecks(&out, []string{bad, good})
	got := out.String()

	if errs != 1 {
		t.Fatalf("runChecks = %d files with errors, want 1", errs)
	}
	// Every file is a table row: findings for bad, an ok row for good.
	for _, want := range []string{"File", "Level", "Finding", "bad.profile.toml", "error", "good.profile.toml", "ok", "│"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q. Got:\n%s", want, got)
		}
	}
}

func TestRunChecksAllClean(t *testing.T) {
	dir := t.TempDir()
	good := writeCheckFile(t, dir, "good.profile.toml", cleanProfile)

	var out strings.Builder
	errs := runChecks(&out, []string{good})

	if errs != 0 {
		t.Fatalf("runChecks = %d, want 0", errs)
	}
	got := out.String()
	// Clean files still get listed, one ok row each.
	for _, want := range []string{"good.profile.toml", "ok", "│"} {
		if !strings.Contains(got, want) {
			t.Errorf("clean run should list the file in the table, missing %q. Got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "error") {
		t.Errorf("clean run should not mention errors. Got:\n%s", got)
	}
}

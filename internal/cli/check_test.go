package cli

import (
	"os"
	"path/filepath"
	"slices"
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

func TestCheckTargetsWithoutInventory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	writeProfile(t, filepath.Join(tmp, "drako"), "alpha")
	files, err := checkTargets(nil)
	if err != nil || len(files) != 1 {
		t.Errorf("checkTargets = (%v, %v), want the one equipped profile", files, err)
	}
}

// With no arguments check covers config.toml too, linted as settings, not a deck.
func TestCheck_CoversConfigTomlAsSettings(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := filepath.Join(tmp, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, dir, "alpha.profile.toml", cleanProfile)
	cfg := writeCheckFile(t, dir, "config.toml", "thme = \"nord\"\n")

	files, err := checkTargets(nil)
	if err != nil || !slices.Contains(files, cfg) {
		t.Fatalf("checkTargets = (%v, %v), want config.toml included", files, err)
	}
	var out strings.Builder
	if errs := runChecks(&out, files); errs != 0 {
		t.Errorf("an unknown setting is a warning, not an error: %d file(s) with errors\n%s", errs, out.String())
	}
	for _, want := range []string{"config.toml", "warning", "thme"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
}

// drako matches deck names case-insensitively, so Git and git are one deck to it.
func TestRunChecks_WarnsOnNamesDifferingOnlyByCase(t *testing.T) {
	dir := t.TempDir()
	inv := filepath.Join(dir, "inventory")
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	upper := writeCheckFile(t, dir, "Git.profile.toml", cleanProfile)
	lower := writeCheckFile(t, inv, "git.profile.toml", cleanProfile)
	other := writeCheckFile(t, dir, "work.profile.toml", cleanProfile)

	var out strings.Builder
	if errs := runChecks(&out, []string{upper, lower, other}); errs != 0 {
		t.Errorf("a case clash is a warning, not an error: %d file(s) with errors", errs)
	}
	got := out.String()
	if n := strings.Count(got, "warning"); n != 2 {
		t.Errorf("want a warning on each of the two files, got %d:\n%s", n, got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "work.profile.toml") && !strings.Contains(line, "ok") {
			t.Errorf("work should stay clean: %s", line)
		}
	}
}

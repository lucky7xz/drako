package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSpecs(t *testing.T) {
	configDir := t.TempDir()
	specsDir := filepath.Join(configDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Two valid specs plus a non-spec file that must be ignored.
	os.WriteFile(filepath.Join(specsDir, "work.spec.toml"), []byte("profiles = [\"git\", \"docker\"]\n"), 0644)
	os.WriteFile(filepath.Join(specsDir, "minimal.spec.toml"), []byte("profiles = [\"core\"]\n"), 0644)
	os.WriteFile(filepath.Join(specsDir, "README.md"), []byte("ignore me\n"), 0644)

	var out strings.Builder
	if err := ListSpecs(configDir, &out); err != nil {
		t.Fatalf("ListSpecs failed: %v", err)
	}

	got := out.String()
	// Names are shown without the .spec.toml suffix, sorted, with profiles.
	if !strings.Contains(got, "minimal") || !strings.Contains(got, "core") {
		t.Errorf("missing minimal spec row, got:\n%s", got)
	}
	if !strings.Contains(got, "work") || !strings.Contains(got, "git, docker") {
		t.Errorf("missing work spec row, got:\n%s", got)
	}
	// Header and box borders should be present.
	if !strings.Contains(got, "Spec") || !strings.Contains(got, "Profiles") || !strings.Contains(got, "┌") {
		t.Errorf("missing table header/borders, got:\n%s", got)
	}
	if strings.Contains(got, "README") {
		t.Errorf("non-spec file leaked into output:\n%s", got)
	}
	// "minimal" must sort before "work".
	if strings.Index(got, "minimal") > strings.Index(got, "work") {
		t.Errorf("specs not sorted:\n%s", got)
	}
}

func TestListSpecs_Empty(t *testing.T) {
	var out strings.Builder
	if err := ListSpecs(t.TempDir(), &out); err != nil {
		t.Fatalf("ListSpecs failed: %v", err)
	}
	if !strings.Contains(out.String(), "No specs found") {
		t.Errorf("expected 'No specs found', got:\n%s", out.String())
	}
}

func TestListSpecs_MissingDir(t *testing.T) {
	var out strings.Builder
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if err := ListSpecs(missing, &out); err != nil {
		t.Fatalf("ListSpecs should not error on missing dir: %v", err)
	}
	if !strings.Contains(out.String(), "No specs found") {
		t.Errorf("expected 'No specs found' for missing dir, got:\n%s", out.String())
	}
}

func TestSpecHandlers_ExitCodes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	tests := []struct {
		name string
		fn   func([]string) int
		args []string
		want int
	}{
		{"spec no args", HandleSpecCommand, []string{"drako", "spec"}, 1},
		{"spec not found", HandleSpecCommand, []string{"drako", "spec", "nosuch"}, 1},
		{"stash no args", HandleStashCommand, []string{"drako", "stash"}, 1},
		{"stash not found", HandleStashCommand, []string{"drako", "stash", "nosuch"}, 1},
		{"strip succeeds on empty config", HandleStripCommand, []string{"drako", "strip"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.args); got != tt.want {
				t.Errorf("%v = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestStripRejectsArguments(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	writeProfile(t, filepath.Join(tmp, "drako"), "alpha")
	if code := HandleStripCommand([]string{"drako", "strip", "alpha"}); code != 1 {
		t.Errorf("strip alpha exit = %d, want 1", code)
	}
	if _, err := os.Stat(filepath.Join(tmp, "drako", "alpha.profile.toml")); err != nil {
		t.Error("strip with an argument stripped profiles anyway")
	}
}

// spec list, spec <name> and stash <name>, end to end through the handlers.
func TestSpecCommandsEndToEnd(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", xdg)
	dir := filepath.Join(xdg, "drako")
	writeProfile(t, filepath.Join(dir, "inventory"), "alpha")
	writeAll(t, dir, map[string]string{"specs/work.spec.toml": "profiles = [\"alpha\"]\n"})

	var code int
	out := captureStdout(t, func() { code = HandleSpecCommand([]string{"drako", "spec", "list"}) })
	if code != 0 || !strings.Contains(out, "work") || !strings.Contains(out, "alpha") {
		t.Fatalf("spec list exit %d:\n%s", code, out)
	}

	out = captureStdout(t, func() { code = HandleSpecCommand([]string{"drako", "spec", "work"}) })
	if code != 0 || !exists(t, dir, "alpha.profile.toml") {
		t.Fatalf("spec work exit %d, alpha equipped %v:\n%s", code, exists(t, dir, "alpha.profile.toml"), out)
	}

	out = captureStdout(t, func() { code = HandleStashCommand([]string{"drako", "stash", "work"}) })
	if code != 0 || exists(t, dir, "alpha.profile.toml") || !exists(t, dir, "inventory", "alpha.profile.toml") {
		t.Fatalf("stash work exit %d:\n%s", code, out)
	}
}

// A spec that cannot be parsed is an error, for spec and stash alike.
func TestSpecCommandsRefuseABrokenSpec(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", xdg)
	writeAll(t, filepath.Join(xdg, "drako"), map[string]string{"specs/bad.spec.toml": "profiles = [\n"})
	for _, h := range []func([]string) int{HandleSpecCommand, HandleStashCommand} {
		var code int
		captureStderr(t, func() { code = h([]string{"drako", "x", "bad"}) })
		if code != 1 {
			t.Errorf("broken spec: exit %d, want 1", code)
		}
	}
}

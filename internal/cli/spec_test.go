package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSpecs(t *testing.T) {
	specsDir := t.TempDir()

	// Two valid specs plus a non-spec file that must be ignored.
	os.WriteFile(filepath.Join(specsDir, "work.spec.toml"), []byte("profiles = [\"git\", \"docker\"]\n"), 0644)
	os.WriteFile(filepath.Join(specsDir, "minimal.spec.toml"), []byte("profiles = [\"core\"]\n"), 0644)
	os.WriteFile(filepath.Join(specsDir, "README.md"), []byte("ignore me\n"), 0644)

	var out strings.Builder
	if err := ListSpecs(specsDir, &out); err != nil {
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

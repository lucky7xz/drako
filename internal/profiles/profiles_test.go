package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"Git":              "git",
		"git.profile.toml": "git",
		"git.profile":      "git",
		"git.toml":         "git",
		"  WORK  ":         "work",
	}
	for in, want := range cases {
		if got := NormalizeName(in); got != want {
			t.Errorf("NormalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestList_FiltersSortsAndFills(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "b.profile.toml"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "a.profile.toml"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte{}, 0o644) // ignored
	os.Mkdir(filepath.Join(dir, "sub.profile.toml"), 0o755)        // dir, ignored

	got, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d: %+v", len(got), got)
	}
	if got[0].Name != "a" || got[1].Name != "b" {
		t.Errorf("want sorted [a b], got [%s %s]", got[0].Name, got[1].Name)
	}
	if got[0].File != "a.profile.toml" || got[0].Norm != "a" {
		t.Errorf("entry not filled: %+v", got[0])
	}
}

func TestList_MissingDirIsTolerant(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Errorf("missing dir must not error, got %v", err)
	}
	if got != nil {
		t.Errorf("missing dir must yield nil, got %+v", got)
	}
}

func TestListSpecs_NormEqualsName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Work.spec.toml"), []byte{}, 0o644)

	got, err := ListSpecs(dir)
	if err != nil {
		t.Fatalf("ListSpecs: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Work" || got[0].Norm != "Work" {
		t.Errorf("specs are not normalized; want Name==Norm==\"Work\", got %+v", got)
	}
}

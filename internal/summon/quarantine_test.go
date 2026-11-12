package summon

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestClassify(t *testing.T) {
	dir := t.TempDir()
	for src, want := range map[string]Kind{
		dir:                                  Folder,
		"git@github.com:user/decks.git":      Git,
		"https://example.com/repo.git":       Git,
		"git://host/repo":                    Git,
		"https://example.com/a.profile.toml": HTTP,
		filepath.Join(dir, "missing"):        HTTP,
	} {
		if got := Classify(src); got != want {
			t.Errorf("Classify(%q) = %v, want %v", src, got, want)
		}
	}
}

// A quarantine is a private folder that Close deletes with its contents.
func TestQuarantine(t *testing.T) {
	q, err := NewQuarantine()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(q.Dir)
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("quarantine %s: %v, mode %v; want private", q.Dir, err, info.Mode())
	}
	if err := os.WriteFile(filepath.Join(q.Dir, "x.toml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(q.Dir); !os.IsNotExist(err) {
		t.Errorf("quarantine still there: %v", err)
	}
}

// A checked deck opens for installing and lists its commands.
func TestCheckedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.profile.toml")
	deck := "x = 1\ny = 1\n[[commands]]\nname = \"A\"\ncommand = { linux_arch = \"pacman\", macos = \"brew\" }\n"
	if err := os.WriteFile(path, []byte(deck), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ValidateProfileFile(path)
	if err != nil || f.Info == nil || f.Info.Size() != int64(len(deck)) || len(f.Profile.Commands) != 1 {
		t.Fatalf("ValidateProfileFile = %+v, %v", f, err)
	}
	r, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if string(got) != deck {
		t.Errorf("Open read %q", got)
	}
	if cmds, err := f.Commands(); err != nil || len(cmds) != 2 {
		t.Errorf("Commands = %v, %v; want both platforms", cmds, err)
	}
}

package summon

import (
	"os"
	"path/filepath"
	"strings"
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

// A checked deck is the bytes read once: a change on disk afterwards does
// not reach it.
func TestCheckedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.profile.toml")
	deck := "x = 1\ny = 1\n[[commands]]\nname = \"A\"\ncommand = { linux_arch = \"pacman\", macos = \"brew\" }\n"
	if err := os.WriteFile(path, []byte(deck), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Read(path)
	if err == nil {
		f, err = ValidateProfile(f)
	}
	if err != nil || f.Info == nil || len(f.Profile.Commands) != 1 {
		t.Fatalf("Read, ValidateProfile = %+v, %v", f, err)
	}
	if err := os.WriteFile(path, []byte("swapped"), 0o644); err != nil {
		t.Fatal(err)
	}
	if string(f.Data) != deck {
		t.Errorf("Data = %q, want the deck as read", f.Data)
	}
	if cmds, err := f.Commands(); err != nil || len(cmds) != 2 {
		t.Errorf("Commands = %v, %v; want both platforms", cmds, err)
	}
}

func TestReadRefusesAFolder(t *testing.T) {
	if _, err := Read(t.TempDir()); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("Read(folder) = %v, want refused", err)
	}
}

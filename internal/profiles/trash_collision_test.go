package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

// Two decks with the same file name (equipped and stashed) trashed within the
// same second must both survive in trash/.
func TestMoveToTrashNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"git.profile.toml", "inventory/git.profile.toml"} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(rel), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := MoveToTrash(dir, rel); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(Trash.Dir(dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("trash holds %d file(s), want both decks", len(entries))
	}
}

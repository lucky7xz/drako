package config

import (
	"os"
	"testing"

	"github.com/lucky7xz/drako/internal/paths"
)

func TestDeletePivotProfile_LockOnlyFileIsRemoved(t *testing.T) {
	dir := t.TempDir()
	if err := WritePivotLocked(dir, "work"); err != nil {
		t.Fatal(err)
	}
	if err := DeletePivotProfile(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.PivotFile(dir)); !os.IsNotExist(err) {
		t.Error("clearing the only thing in pivot.toml should remove the file")
	}
}

// The watcher may not have caught up with a pivot file removed elsewhere;
// clearing a lock that is already gone is not an error.
func TestDeletePivotProfile_MissingFileIsFine(t *testing.T) {
	if err := DeletePivotProfile(t.TempDir()); err != nil {
		t.Errorf("clearing an absent lock failed: %v", err)
	}
}

// A pivot.toml with a typo still holds the user's order; writing the lock must
// not silently replace it.
func TestWritePivotLocked_KeepsAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	broken := "equipped_order = [\"b\", \"a\"\n"
	if err := os.WriteFile(paths.PivotFile(dir), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WritePivotLocked(dir, "work"); err == nil {
		t.Error("writing over an unreadable pivot.toml should fail, not replace it")
	}
	if b, _ := os.ReadFile(paths.PivotFile(dir)); string(b) != broken {
		t.Errorf("pivot.toml was rewritten to %q", b)
	}
}

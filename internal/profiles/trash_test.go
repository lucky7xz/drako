package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/paths"
)

func TestMoveToTrash_TimestampsIntoTrash(t *testing.T) {
	cfg := t.TempDir()
	src := filepath.Join(cfg, "config.toml")
	if err := os.WriteFile(src, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MoveToTrash(cfg, "config.toml"); err != nil {
		t.Fatalf("MoveToTrash: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should be gone, stat err = %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(paths.TrashDir(cfg), "config.toml.*"))
	if len(matches) != 1 {
		t.Errorf("expected 1 timestamped trash file, got %v", matches)
	}
}

func TestMoveToTrash_RejectsTraversal(t *testing.T) {
	cfg := t.TempDir()
	if err := MoveToTrash(cfg, "../escape.toml"); err == nil {
		t.Error("expected path traversal to be rejected, got nil")
	}
}

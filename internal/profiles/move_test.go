package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/paths"
)

func TestMove_InventoryToEquipped(t *testing.T) {
	cfg := t.TempDir()
	invDir := paths.InventoryDir(cfg)
	if err := os.MkdirAll(invDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(invDir, "git.profile.toml")
	if err := os.WriteFile(src, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Move(cfg, "git.profile.toml", Inventory, Equipped); err != nil {
		t.Fatalf("Move: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cfg, "git.profile.toml")); err != nil {
		t.Errorf("expected file in equipped root, stat err = %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected file gone from inventory, stat err = %v", err)
	}
}

func TestMove_EquippedToTrash_CreatesDestDir(t *testing.T) {
	cfg := t.TempDir()
	src := filepath.Join(cfg, "old.profile.toml")
	if err := os.WriteFile(src, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Trash dir does not exist yet; Move must create it.
	if err := Move(cfg, "old.profile.toml", Equipped, Trash); err != nil {
		t.Fatalf("Move: %v", err)
	}

	if _, err := os.Stat(filepath.Join(paths.TrashDir(cfg), "old.profile.toml")); err != nil {
		t.Errorf("expected file in trash, stat err = %v", err)
	}
}

func TestMove_RefusesToOverwriteExisting(t *testing.T) {
	cfg := t.TempDir()
	inv := paths.InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	// Same filename exists in both inventory (source) and equipped (dest).
	if err := os.WriteFile(filepath.Join(inv, "git.profile.toml"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "git.profile.toml"), []byte("orig\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Move(cfg, "git.profile.toml", Inventory, Equipped); err == nil {
		t.Error("expected error moving onto an existing destination, got nil")
	}
	// The original equipped file must be untouched.
	got, _ := os.ReadFile(filepath.Join(cfg, "git.profile.toml"))
	if string(got) != "orig\n" {
		t.Errorf("destination was clobbered: %q", got)
	}
}

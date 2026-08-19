package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

// lockedName returns pivot.toml's current lock, "" when there is none.
func lockedName(t *testing.T, dir string) string {
	t.Helper()
	pf, err := config.ReadPivotProfile(dir)
	if err != nil {
		t.Fatalf("ReadPivotProfile: %v", err)
	}
	return pf.Locked
}

func lock(t *testing.T, dir, name string) {
	t.Helper()
	if err := config.WritePivotLocked(dir, name); err != nil {
		t.Fatalf("WritePivotLocked: %v", err)
	}
}

// A lock left pointing at a profile that is no longer equipped cannot be
// resolved on the next load, which drops that launch into the rescue grid. So
// every command that unequips or removes a profile has to release the lock.
func TestLockReleasedWhenProfileLeavesEquipped(t *testing.T) {
	t.Run("ApplySpec stashing the locked profile", func(t *testing.T) {
		cfg := t.TempDir()
		writeProfile(t, cfg, "work")
		writeProfile(t, cfg, "git")
		lock(t, cfg, "work")

		if err := ApplySpec(cfg, []string{"git"}, false); err != nil {
			t.Fatalf("ApplySpec: %v", err)
		}
		if got := lockedName(t, cfg); got != "" {
			t.Errorf("lock = %q, want it released", got)
		}
	})

	t.Run("ApplySpec keeping the locked profile", func(t *testing.T) {
		cfg := t.TempDir()
		writeProfile(t, cfg, "work")
		writeProfile(t, cfg, "git")
		lock(t, cfg, "work")

		if err := ApplySpec(cfg, []string{"work"}, false); err != nil {
			t.Fatalf("ApplySpec: %v", err)
		}
		if got := lockedName(t, cfg); got != "work" {
			t.Errorf("lock = %q, want it kept — the profile is still equipped", got)
		}
	})

	t.Run("StashSpec", func(t *testing.T) {
		cfg := t.TempDir()
		writeProfile(t, cfg, "work")
		lock(t, cfg, "work")

		if err := StashSpec(cfg, []string{"work"}); err != nil {
			t.Fatalf("StashSpec: %v", err)
		}
		if got := lockedName(t, cfg); got != "" {
			t.Errorf("lock = %q, want it released", got)
		}
	})

	t.Run("StripAllProfiles", func(t *testing.T) {
		cfg := t.TempDir()
		writeProfile(t, cfg, "work")
		lock(t, cfg, "work")

		if err := StripAllProfiles(cfg); err != nil {
			t.Fatalf("StripAllProfiles: %v", err)
		}
		if got := lockedName(t, cfg); got != "" {
			t.Errorf("lock = %q, want it released", got)
		}
	})

	t.Run("purge --target", func(t *testing.T) {
		cfg := t.TempDir()
		writeProfile(t, cfg, "work")
		lock(t, cfg, "work")

		if err := PurgeConfig(cfg, PurgeOptions{TargetProfiles: []string{"work"}}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if _, err := os.Stat(filepath.Join(cfg, "work.profile.toml")); !os.IsNotExist(err) {
			t.Error("the profile should have been trashed")
		}
		if got := lockedName(t, cfg); got != "" {
			t.Errorf("lock = %q, want it released", got)
		}
	})

	t.Run("an unrelated profile leaves the lock alone", func(t *testing.T) {
		cfg := t.TempDir()
		writeProfile(t, cfg, "work")
		writeProfile(t, cfg, "git")
		lock(t, cfg, "work")

		if err := StashSpec(cfg, []string{"git"}); err != nil {
			t.Fatalf("StashSpec: %v", err)
		}
		if got := lockedName(t, cfg); got != "work" {
			t.Errorf("lock = %q, want %q", got, "work")
		}
	})
}

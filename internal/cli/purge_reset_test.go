package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

// Reset Core resets config.toml and brings the core deck back if it is gone.
// It is not the factory reset (--destroyeverything): removed decks stay gone.
func TestPurgeConfigResetsOnlyConfigAndCore(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", xdg)
	dir := filepath.Join(xdg, "drako")
	if _, err := config.LoadConfig(nil); err != nil { // first run bootstraps
		t.Fatal(err)
	}
	jukebox := filepath.Join(dir, "inventory", "jukebox.profile.toml")
	core := filepath.Join(dir, "core.profile.toml")
	for _, p := range []string{jukebox, core} {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}

	if err := PurgeConfig(dir, PurgeOptions{TargetConfig: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := config.LoadConfig(nil); err != nil { // the next launch
		t.Fatal(err)
	}

	if _, err := os.Stat(jukebox); err == nil {
		t.Error("Reset Core brought back a deck the user had removed")
	}
	if _, err := os.Stat(core); err != nil {
		t.Errorf("Reset Core should bring the core deck back: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err != nil {
		t.Errorf("Reset Core should leave a fresh config.toml: %v", err)
	}
}

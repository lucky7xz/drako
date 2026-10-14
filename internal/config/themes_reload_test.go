package config

import (
	"os"
	"path/filepath"
	"testing"
)

// The watcher reloads on any .toml change, themes.toml included, so the next
// lookup must see the edited palette rather than one cached at first use.
func TestGetThemeSeesAnEditedThemesFile(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(color string) {
		if err := os.WriteFile(filepath.Join(dir, "themes.toml"), []byte("[mine]\nprimary = \""+color+"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("#111111")
	if got := GetTheme("mine").Primary; got != "#111111" {
		t.Fatalf("first lookup: primary = %q, want #111111", got)
	}
	write("#222222")
	if got := GetTheme("mine").Primary; got != "#222222" {
		t.Errorf("after editing themes.toml: primary = %q, want #222222", got)
	}
}

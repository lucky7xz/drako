package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDraculaFoundation: the in-code dracula is always present, even with no
// user themes file — it is the guaranteed fallback.
func TestDraculaFoundation(t *testing.T) {
	themes := buildThemes(t.TempDir()) // empty dir: no themes.toml
	if themes["dracula"] != dracula {
		t.Errorf("dracula should be the in-code foundation, got %+v", themes["dracula"])
	}
}

// TestEmbeddedThemesLoad: the themes shipped in the binary load alongside it.
func TestEmbeddedThemesLoad(t *testing.T) {
	themes := buildThemes(t.TempDir())
	for _, name := range []string{"nord", "jade", "everforest", "viper"} {
		if _, ok := themes[name]; !ok {
			t.Errorf("embedded theme %q should be loaded", name)
		}
	}
}

// TestUserThemesWin: a user's themes.toml adds new themes and overrides by
// name — including [dracula], the backward-compat case for users who already
// have one on disk.
func TestUserThemesWin(t *testing.T) {
	dir := t.TempDir()
	writeUserThemes(t, dir, `
[dracula]
Primary = "#000001"

[mine]
Primary = "#abcdef"
`)
	themes := buildThemes(dir)

	if themes["dracula"].Primary != "#000001" {
		t.Errorf("user [dracula] should win, got %q", themes["dracula"].Primary)
	}
	if _, ok := themes["mine"]; !ok {
		t.Error("user theme 'mine' should be added")
	}
	if _, ok := themes["nord"]; !ok {
		t.Error("untouched embedded themes should survive")
	}
}

// TestMalformedUserThemesIgnored: a broken themes.toml is ignored, not fatal;
// the foundation and embedded themes remain.
func TestMalformedUserThemesIgnored(t *testing.T) {
	dir := t.TempDir()
	writeUserThemes(t, dir, "this is not [valid toml")
	themes := buildThemes(dir)

	if themes["dracula"] != dracula {
		t.Error("a malformed user file must not wipe the foundation")
	}
	if _, ok := themes["nord"]; !ok {
		t.Error("a malformed user file must not wipe embedded themes")
	}
}

func TestDarkenAccent(t *testing.T) {
	if got, want := darkenAccent("#ff2e63", "#0d0221"), "#79153e"; got != want {
		t.Errorf("darkenAccent(#ff2e63, #0d0221) = %s, want %s", got, want)
	}
	if darkenAccent("not-a-color", "#000000") != "not-a-color" {
		t.Error("a malformed accent hex should be returned unchanged")
	}
	if darkenAccent("#ff2e63", "#zzzzzz") != "#ff2e63" {
		t.Error("a malformed background hex should return accent unchanged")
	}
}

func writeUserThemes(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "themes.toml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write themes.toml: %v", err)
	}
}

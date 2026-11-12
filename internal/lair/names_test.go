package lair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/equip"
)

// A deck's file name is drawn and typed, so drako refuses one it cannot draw
// as written: equipped it is broken with the reason, in the inventory it is
// left out. Escapes only in this file.
func TestRefusedFileNames(t *testing.T) {
	_, dir := lairAt(t)
	deck := "x = 1\ny = 1\n[[commands]]\nname = \"A\"\ncommand = \"echo\"\n"
	joined := "team\U0001f469\u200d\U0001f4bb.profile.toml"
	steering := "x\x1b]52;c;eA==.profile.toml"
	for _, n := range []string{"ok.profile.toml", joined, steering} {
		writeFile(t, filepath.Join(dir, n), deck)
	}
	if err := os.MkdirAll(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "inventory", joined), deck)

	b, err := LoadConfig(dir, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range b.Profiles {
		if base := filepath.Base(p.Path); base == joined || base == steering {
			t.Errorf("loaded the refused %q", base)
		}
	}
	reasons := map[string]string{}
	for _, e := range b.Broken {
		reasons[filepath.Base(e.Path)] = e.Err
	}
	if r := reasons[joined]; !strings.Contains(r, "U+200D") || !strings.Contains(r, "Rename it to \"team\U0001f469\U0001f4bb.profile.toml\"") {
		t.Errorf("joined name: %q", r)
	}
	if r := reasons[steering]; !strings.Contains(r, "U+001B, a character that steers the terminal") || strings.Contains(r, "Rename") {
		t.Errorf("steering name: %q", r)
	}

	for _, loc := range []equip.Location{equip.Equipped, equip.Inventory} {
		entries, err := List(dir, loc)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.File == joined || e.File == steering {
				t.Errorf("listed the refused %q", e.File)
			}
		}
	}
}

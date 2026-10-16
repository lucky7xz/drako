package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// localSummoner summons from a deck folder on disk; nothing may be cloned.
func localSummoner(t *testing.T) (*Summoner, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	clone := repoCloner{func(string) error { t.Fatal("a local summon must not clone"); return nil }}
	return &Summoner{ConfigDir: t.TempDir(), Downloader: &MockDownloader{}, Cloner: clone, UI: &yesToAll{}}, t.TempDir()
}

func writeAll(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSummonLocalFolderCopiesDeckAndAssets(t *testing.T) {
	s, dir := localSummoner(t)
	writeAll(t, dir, map[string]string{
		"deck.profile.toml":   "assets = [\"scripts\"]\n" + validDeck,
		"scripts/run.sh":      "v1",
		"scripts/lib/util.sh": "lib",
	})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	for _, p := range []string{"inventory/deck.profile.toml", "assets/deck/scripts/run.sh", "assets/deck/scripts/lib/util.sh"} {
		if _, err := os.Stat(filepath.Join(s.ConfigDir, p)); err != nil {
			t.Errorf("%s not summoned: %v", p, err)
		}
	}

	// Summoning again after an edit overwrites the deck's assets.
	writeAll(t, dir, map[string]string{"scripts/run.sh": "v2"})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("second Summon: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(s.ConfigDir, "assets/deck/scripts/run.sh")); string(got) != "v2" {
		t.Errorf("asset not replaced on re-summon, got %q", got)
	}
}

func TestSummonLocalFolderRefusesPins(t *testing.T) {
	s, dir := localSummoner(t)
	writeAll(t, dir, map[string]string{"deck.profile.toml": validDeck})
	s.Rev = strings.Repeat("a", 40)
	if err := s.Summon(dir); err == nil || !strings.Contains(err.Error(), "folder") {
		t.Errorf("want a pin with a folder refused, got %v", err)
	}
}

func TestSummonLocalFolderSkipsSymlinkedAssets(t *testing.T) {
	s, dir := localSummoner(t)
	secret := filepath.Join(t.TempDir(), "id_ed25519")
	writeAll(t, filepath.Dir(secret), map[string]string{"id_ed25519": "PRIVATE"})
	writeAll(t, dir, map[string]string{"deck.profile.toml": "assets = [\"data\"]\n" + validDeck, "data/ok.txt": "ok"})
	if err := os.Symlink(secret, filepath.Join(dir, "data", "leak")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(s.ConfigDir, "assets/deck/data/leak")); err == nil {
		t.Error("a symlinked asset was followed out of the folder")
	}
}

// Summoning from the config dir, or a folder holding it, would copy decks onto themselves.
func TestSummonLocalFolderRefusesTheConfigDir(t *testing.T) {
	s, _ := localSummoner(t)
	writeAll(t, s.ConfigDir, map[string]string{"inventory/deck.profile.toml": validDeck})
	for _, dir := range []string{s.ConfigDir, filepath.Join(s.ConfigDir, "inventory"), filepath.Dir(s.ConfigDir)} {
		if err := s.Summon(dir); err == nil {
			t.Errorf("%s: want a refusal", dir)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(s.ConfigDir, "inventory/deck.profile.toml")); string(got) != validDeck {
		t.Error("the inventory deck was changed")
	}
}

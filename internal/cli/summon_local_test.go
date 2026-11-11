package cli

import (
	"fmt"
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

// placed reports where file sits in s's config dir: "equipped", "inventory",
// "both", or "" when it is in neither.
func placed(s *Summoner, file string) string {
	_, inRoot := os.Stat(filepath.Join(s.ConfigDir, file))
	_, inInv := os.Stat(filepath.Join(s.ConfigDir, "inventory", file))
	switch {
	case inRoot == nil && inInv == nil:
		return "both"
	case inRoot == nil:
		return "equipped"
	case inInv == nil:
		return "inventory"
	}
	return ""
}

// equipDecks fills s's config root with n decks, e0.profile.toml onwards.
func equipDecks(t *testing.T, s *Summoner, n int) {
	t.Helper()
	for i := range n {
		writeAll(t, s.ConfigDir, map[string]string{fmt.Sprintf("e%d.profile.toml", i): validDeck})
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
	for _, p := range []string{"deck.profile.toml", "assets/deck/scripts/run.sh", "assets/deck/scripts/lib/util.sh"} {
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

// New decks are equipped while there is room; once nine are, the rest wait in the inventory.
func TestSummonEquipsNewDecksUntilFull(t *testing.T) {
	s, dir := localSummoner(t)
	equipDecks(t, s, 8)
	writeAll(t, dir, map[string]string{"a.profile.toml": validDeck, "b.profile.toml": validDeck})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got := placed(s, "a.profile.toml"); got != "equipped" {
		t.Errorf("a: placed %q, want equipped (the ninth slot)", got)
	}
	if got := placed(s, "b.profile.toml"); got != "inventory" {
		t.Errorf("b: placed %q, want inventory (equipped is full)", got)
	}
}

// A deck the user declines is never written, so it holds no equipped slot.
func TestSummonDeclinedDeckTakesNoSlot(t *testing.T) {
	s, dir := localSummoner(t)
	equipDecks(t, s, 8)
	writeAll(t, dir, map[string]string{"a.profile.toml": validDeck, "b.profile.toml": validDeck})
	s.UI = &MockUI{ConfirmFunc: func(p string) bool { return !strings.Contains(p, "a.profile.toml") }}
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got := placed(s, "b.profile.toml"); got != "equipped" {
		t.Errorf("b: placed %q, want equipped (a was declined)", got)
	}
}

// A deck that is already equipped is replaced there even when equipped is full.
func TestSummonReplacesEquippedDeckWhenFull(t *testing.T) {
	s, dir := localSummoner(t)
	equipDecks(t, s, 9)
	writeAll(t, dir, map[string]string{"e3.profile.toml": validDeck + "# v2\n"})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got := placed(s, "e3.profile.toml"); got != "equipped" {
		t.Errorf("e3: placed %q, want equipped", got)
	}
	if got, _ := os.ReadFile(filepath.Join(s.ConfigDir, "e3.profile.toml")); !strings.Contains(string(got), "# v2") {
		t.Error("the equipped e3 was not replaced")
	}
}

// Deck names match the way the rest of drako matches them, ignoring case, so a
// differently cased file replaces the deck instead of adding a twin.
func TestSummonReplacesDeckWhateverItsCase(t *testing.T) {
	s, dir := localSummoner(t)
	writeAll(t, s.ConfigDir, map[string]string{"core.profile.toml": validDeck})
	writeAll(t, dir, map[string]string{"Core.profile.toml": validDeck + "# v2\n"})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	var decks []string
	for _, d := range []string{s.ConfigDir, filepath.Join(s.ConfigDir, "inventory")} {
		entries, _ := os.ReadDir(d)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".profile.toml") {
				decks = append(decks, e.Name())
			}
		}
	}
	if got := strings.Join(decks, " "); got != "core.profile.toml" {
		t.Fatalf("decks on disk: %q, want just core.profile.toml", got)
	}
	if got, _ := os.ReadFile(filepath.Join(s.ConfigDir, "core.profile.toml")); !strings.Contains(string(got), "# v2") {
		t.Error("core.profile.toml was not replaced")
	}
}

// Only the top level of the folder is summoned; decks and specs in subfolders stay put.
func TestSummonIgnoresSubfolders(t *testing.T) {
	s, dir := localSummoner(t)
	writeAll(t, dir, map[string]string{
		"top.profile.toml":      validDeck,
		"sub/deep.profile.toml": validDeck,
		"sub/deep.spec.toml":    "profiles = [\"top\"]\n",
	})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got := placed(s, "top.profile.toml"); got != "equipped" {
		t.Errorf("top: placed %q, want equipped", got)
	}
	if got := placed(s, "deep.profile.toml"); got != "" {
		t.Errorf("a deck from a subfolder was summoned: placed %q", got)
	}
	if _, err := os.Stat(filepath.Join(s.ConfigDir, "specs", "deep.spec.toml")); err == nil {
		t.Error("a spec from a subfolder was summoned")
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

// A deck that is a link to the very file being summoned is already in place;
// copying it onto itself would truncate the source before reading it.
func TestSummonLeavesALinkedDeckAlone(t *testing.T) {
	s, dir := localSummoner(t)
	writeAll(t, dir, map[string]string{"deck.profile.toml": validDeck})
	src := filepath.Join(dir, "deck.profile.toml")
	if err := os.Symlink(src, filepath.Join(s.ConfigDir, "deck.profile.toml")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got, _ := os.ReadFile(src); string(got) != validDeck {
		t.Errorf("summon changed the deck's own source file to %q", got)
	}
	if info, err := os.Lstat(filepath.Join(s.ConfigDir, "deck.profile.toml")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the equipped link was replaced; it already was the deck (%v)", err)
	}
}

// Replacing a deck replaces its entry in the config dir; a deck that is a link
// to some other file is swapped for the new deck, never written through.
func TestSummonReplacesALinkInsteadOfWritingThroughIt(t *testing.T) {
	s, dir := localSummoner(t)
	elsewhere := filepath.Join(t.TempDir(), "elsewhere.profile.toml")
	writeAll(t, filepath.Dir(elsewhere), map[string]string{"elsewhere.profile.toml": "keep me"})
	if err := os.Symlink(elsewhere, filepath.Join(s.ConfigDir, "deck.profile.toml")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	writeAll(t, dir, map[string]string{"deck.profile.toml": validDeck + "# v2\n"})
	if err := s.Summon(dir); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got, _ := os.ReadFile(elsewhere); string(got) != "keep me" {
		t.Errorf("summon wrote through the link into %s: %q", elsewhere, got)
	}
	deck := filepath.Join(s.ConfigDir, "deck.profile.toml")
	if info, err := os.Lstat(deck); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("the equipped deck should now be a plain file: %v %v", info, err)
	}
	if got, _ := os.ReadFile(deck); !strings.Contains(string(got), "# v2") {
		t.Error("the equipped deck was not replaced")
	}
}

// The prompt says in words where a deck lands, so one "y" never replaces an
// equipped deck without saying so.
func TestSummonSaysWhereADeckLands(t *testing.T) {
	cfg := t.TempDir()
	inv := filepath.Join(cfg, "inventory")
	writeAll(t, cfg, map[string]string{"core.profile.toml": validDeck, "inventory/st.profile.toml": validDeck})
	for _, tt := range []struct{ dst, want string }{
		{filepath.Join(cfg, "core.profile.toml"), "replaces your equipped core.profile.toml"},
		{filepath.Join(inv, "st.profile.toml"), "replaces st.profile.toml in the inventory"},
		{filepath.Join(cfg, "new.profile.toml"), "equipped"},
		{filepath.Join(inv, "new.profile.toml"), "inventory (9 decks are equipped)"},
	} {
		if got := destinationLabel(cfg, tt.dst); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.dst, got, tt.want)
		}
	}
}

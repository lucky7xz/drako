package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// Trashing an equipped profile renames a root .toml, so a reload follows. It
// must not throw away the inventory screen, its verdict, or the staged lists.
func TestReloadKeepsTheInventoryOpen(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	m := Model{Config: cfg, styles: BuildStyles(cfg), mode: inventoryMode,
		inventory: inventoryModel{
			State:  core.NewInventoryState([]string{"b.profile.toml", "a.profile.toml"}, nil, 9),
			status: "Trashed c → trash/", statusOK: true,
		}}

	m = m.applyReloadedBundle(config.ConfigBundle{Config: cfg})

	if m.mode != inventoryMode {
		t.Fatalf("a clean reload left the inventory, mode = %v", m.mode)
	}
	if m.inventory.status != "Trashed c → trash/" || m.inventory.State.Visible[0] != "b.profile.toml" {
		t.Errorf("verdict or staged order lost: %q %v", m.inventory.status, m.inventory.State.Visible)
	}
}

// openInventory starts drako on a fresh config dir and presses i.
func openInventory(t *testing.T) (Model, string) {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	m := InitialModel(false, Session{})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = next.(Model)
	if m.mode != inventoryMode {
		t.Fatalf("i did not open the inventory, mode = %v", m.mode)
	}
	return m, filepath.Join(xdg, "drako")
}

// A deck appearing on disk while the inventory is open shows up at once; the
// staged moves go, since they were made against lists that are now stale.
func TestWatcherRefreshesTheOpenInventory(t *testing.T) {
	m, dir := openInventory(t)
	st := m.inventory.State
	if err := st.PickUpItem(core.ListVisible, 0); err != nil { // stage a stash
		t.Fatal(err)
	}
	if err := st.PlaceItem(core.ListInventory, 0); err != nil {
		t.Fatal(err)
	}
	staged := slices.Clone(st.Visible)

	added := filepath.Join(dir, "added.profile.toml")
	if err := os.WriteFile(added, []byte("x = 1\ny = 1\n[[commands]]\nname = \"a\"\ncommand = \"echo\"\nrow = 0\ncol = \"a\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(ConfigChangedMsg{Path: added})
	m = next.(Model)

	if !slices.Contains(m.inventory.State.Visible, "added.profile.toml") {
		t.Errorf("Visible = %v, want the new deck", m.inventory.State.Visible)
	}
	if len(m.inventory.State.Visible) != len(staged)+2 {
		t.Errorf("Visible = %v, want the staged stash undone and the new deck added", m.inventory.State.Visible)
	}
	if m.inventory.status == "" {
		t.Error("the reload should say the lists were refreshed")
	}
}

// An event that changed no deck (drako's own trash, already dropped from the
// list) leaves the inventory exactly as it is.
func TestWatcherLeavesAnUpToDateInventory(t *testing.T) {
	m, dir := openInventory(t)
	m.inventory.status, m.inventory.statusOK = "Trashed c → trash/", true

	next, _ := m.Update(ConfigChangedMsg{Path: filepath.Join(dir, "pivot.toml")})
	m = next.(Model)

	if m.inventory.status != "Trashed c → trash/" {
		t.Errorf("status = %q, want it kept", m.inventory.status)
	}
}

// Saving a deck changes what is in a file, not where the decks are, so the
// staged arrangement and the cursor survive the editor closing.
func TestEditKeepsStagedMoves(t *testing.T) {
	m, dir := openInventory(t)
	st := m.inventory.State
	if err := st.PickUpItem(core.ListVisible, 0); err != nil { // stage a stash
		t.Fatal(err)
	}
	if err := st.PlaceItem(core.ListInventory, 0); err != nil {
		t.Fatal(err)
	}
	staged := slices.Clone(st.Visible)

	core := filepath.Join(dir, "core.profile.toml")
	next, _ := m.afterEdit(editorFinishedMsg{path: core})
	next, _ = next.(Model).Update(ConfigChangedMsg{Path: core}) // the watcher sees the save
	m = next.(Model)

	if !slices.Equal(m.inventory.State.Visible, staged) {
		t.Errorf("Visible = %v, want the staged %v", m.inventory.State.Visible, staged)
	}
}

// drako's own trash is not a change on disk to the inventory: the staged
// arrangement and the verdict survive the watcher event it causes.
func TestOwnTrashKeepsStagedMoves(t *testing.T) {
	m, dir := openInventory(t)
	extra := filepath.Join(dir, "extra.profile.toml")
	for _, f := range []string{extra, filepath.Join(dir, "zz.profile.toml")} {
		if err := os.WriteFile(f, []byte("x = 1\ny = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m = m.refreshInventory() // core, extra, zz
	st := m.inventory.State
	if err := st.PickUpItem(core.ListVisible, 2); err != nil { // stage a stash of zz
		t.Fatal(err)
	}
	if err := st.PlaceItem(core.ListInventory, 0); err != nil {
		t.Fatal(err)
	}
	m.inventory.cursor = 1 // extra
	for _, k := range []tea.KeyMsg{{Type: tea.KeyDelete}, {Type: tea.KeyRunes, Runes: []rune("extra")}, {Type: tea.KeyEnter}} {
		next, _ := m.Update(k)
		m = next.(Model)
	}
	next, _ := m.Update(ConfigChangedMsg{Path: extra})
	m = next.(Model)

	if got := m.inventory.State.Visible; len(got) != 1 || got[0] != "core.profile.toml" {
		t.Errorf("Visible = %v, want only core", got)
	}
	if !strings.HasPrefix(m.inventory.status, "Trashed extra") {
		t.Errorf("status = %q, want the trash verdict kept", m.inventory.status)
	}
}

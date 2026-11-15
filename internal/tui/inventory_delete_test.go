package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/lair"
)

func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

// send feeds keys through the inventory update loop, threading the model.
func send(t *testing.T, m Model, msgs ...tea.KeyMsg) Model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.updateInventoryMode(msg)
		m = next.(Model)
	}
	return m
}

// trashed lists the files sitting in the config dir's trash.
func trashed(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(lair.TrashDir(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

func TestInventoryDelete_TypedNameConfirms(t *testing.T) {
	t.Run("the right name trashes the file", func(t *testing.T) {
		m, dir := editInventoryModel(t)

		m = send(t, m, keyType(tea.KeyDelete))
		if m.inventory.pending == nil {
			t.Fatal("delete should arm a confirmation")
		}
		if m.inventory.pending.name != "eq" {
			t.Errorf("prompt name = %q, want %q", m.inventory.pending.name, "eq")
		}

		m = send(t, m, keyRune("eq"), keyType(tea.KeyEnter))

		if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); !os.IsNotExist(err) {
			t.Error("the profile file should be gone from the config dir")
		}
		got := trashed(t, dir)
		if len(got) != 1 || !strings.HasPrefix(got[0], "eq.profile.toml.") {
			t.Errorf("trash = %v, want one timestamped eq.profile.toml", got)
		}
		if len(m.inventory.State.Visible) != 0 {
			t.Errorf("Visible = %v, want empty", m.inventory.State.Visible)
		}
		if m.inventory.pending != nil {
			t.Error("confirmation should be cleared after deleting")
		}
	})

	t.Run("a wrong name deletes nothing", func(t *testing.T) {
		m, dir := editInventoryModel(t)

		m = send(t, m, keyType(tea.KeyDelete), keyRune("nope"), keyType(tea.KeyEnter))

		if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); err != nil {
			t.Error("the profile file must survive a mistyped name")
		}
		if got := trashed(t, dir); len(got) != 0 {
			t.Errorf("trash = %v, want empty", got)
		}
		if m.inventory.status == "" {
			t.Error("a mismatch should say so")
		}
	})

	t.Run("esc cancels", func(t *testing.T) {
		m, dir := editInventoryModel(t)

		m = send(t, m, keyType(tea.KeyDelete), keyRune("eq"), keyType(tea.KeyEsc))

		if m.inventory.pending != nil {
			t.Error("esc should disarm the confirmation")
		}
		if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); err != nil {
			t.Error("the profile file must survive a cancel")
		}
	})

	t.Run("backspace edits the typed name", func(t *testing.T) {
		m, _ := editInventoryModel(t)

		m = send(t, m, keyType(tea.KeyDelete), keyRune("eqx"), keyType(tea.KeyBackspace))

		if m.inventory.pending.typed != "eq" {
			t.Errorf("typed = %q, want %q", m.inventory.pending.typed, "eq")
		}
	})

	// 'q' leaves the inventory normally, so it has to stop doing that while a
	// name is being typed or no profile containing one could be confirmed.
	t.Run("q types a letter instead of leaving", func(t *testing.T) {
		m, _ := editInventoryModel(t)

		m = send(t, m, keyType(tea.KeyDelete), keyRune("q"))

		if m.mode != inventoryMode {
			t.Error("q must not leave the inventory while a name is being typed")
		}
		if m.inventory.pending == nil || m.inventory.pending.typed != "q" {
			t.Errorf("q should have been typed into the prompt")
		}
	})

	t.Run("the name is matched loosely", func(t *testing.T) {
		m, dir := editInventoryModel(t)

		// NormalizeName lowercases and strips the suffix, so both are accepted.
		send(t, m, keyType(tea.KeyDelete), keyRune("EQ.profile.toml"), keyType(tea.KeyEnter))

		if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); !os.IsNotExist(err) {
			t.Error("a differently-cased name with the suffix should still match")
		}
	})
}

func TestInventoryDelete_Guards(t *testing.T) {
	t.Run("glassroot is a no-op", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		m.GlassrootMode = true

		m = send(t, m, keyType(tea.KeyDelete))

		if m.inventory.pending != nil {
			t.Error("glassroot must not arm a delete")
		}
	})

	t.Run("a held item blocks it", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		if err := m.inventory.State.PickUpItem(equip.ListVisible, 0); err != nil {
			t.Fatal(err)
		}

		m = send(t, m, keyType(tea.KeyDelete))

		if m.inventory.pending != nil {
			t.Error("a delete must not arm mid-drag")
		}
		if !strings.Contains(m.inventory.status, "held item") {
			t.Errorf("status = %q, want it to mention the held item", m.inventory.status)
		}
	})

	t.Run("the locked profile is refused", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		m.profile.pivotName = "eq"

		m = send(t, m, keyType(tea.KeyDelete))

		if m.inventory.pending != nil {
			t.Error("the locked profile must not be deletable")
		}
		if !strings.Contains(m.inventory.status, "Anchored profile") {
			t.Errorf("status = %q, want it to mention the anchor", m.inventory.status)
		}
	})

	t.Run("a button row arms nothing", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		m.inventory.focusedList = 2 // Apply Changes

		m = send(t, m, keyType(tea.KeyDelete))

		if m.inventory.pending != nil {
			t.Error("the button rows have no profile to delete")
		}
	})
}

// Lift-and-place is staged until Apply runs, so the list an item is shown in
// can disagree with where the file actually lives.
func TestInventoryDelete_ResolvesStagedItemToRealPath(t *testing.T) {
	m, dir := editInventoryModel(t)

	// Drag the stashed profile into the equipped list without applying.
	m.inventory.focusedList = equip.ListInventory
	if err := m.inventory.State.PickUpItem(equip.ListInventory, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.inventory.State.PlaceItem(equip.ListVisible, 1); err != nil {
		t.Fatal(err)
	}
	m.inventory.focusedList = equip.ListVisible
	m.inventory.cursor = 1

	m = send(t, m, keyType(tea.KeyDelete), keyRune("st"), keyType(tea.KeyEnter))

	stashed := filepath.Join(lair.InventoryDir(dir), "st.profile.toml")
	if _, err := os.Stat(stashed); !os.IsNotExist(err) {
		t.Error("the file should be trashed from where it really is, inventory/")
	}
	if got := trashed(t, dir); len(got) != 1 {
		t.Errorf("trash = %v, want one entry", got)
	}
	// The equipped profile was never touched, and the arrangement survives.
	if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); err != nil {
		t.Error("the equipped profile must be untouched")
	}
}

// The confirmation shares the one feedback row with messages like "equipped is
// full", directly under Holding:. A held item blocks arming a delete, so the
// two can never appear together — the check is that both land on the same row.
func TestInventoryDelete_PromptSharesTheStatusRow(t *testing.T) {
	renderedLineOf := func(t *testing.T, mutate func(*Model), want string) int {
		t.Helper()
		m, _ := editInventoryModel(t)
		m.termWidth, m.termHeight = 100, 40
		m.Config.X, m.Config.Y = 2, 2
		m.Config.Theme = "default"
		m.styles = BuildStyles(m.Config, "")
		mutate(&m)

		lines := strings.Split(m.viewInventoryMode(m.keyHint()), "\n")
		for i, ln := range lines {
			if strings.Contains(ln, want) {
				return i
			}
		}
		t.Fatalf("%q missing from the view:\n%s", want, strings.Join(lines, "\n"))
		return -1
	}

	statusRow := renderedLineOf(t, func(m *Model) {
		m.inventory.status = "equipped is full (9 max) — stash one first"
	}, "equipped is full")

	promptRow := renderedLineOf(t, func(m *Model) {
		next, _ := m.updateInventoryMode(keyType(tea.KeyDelete))
		*m = next.(Model)
		next, _ = m.updateInventoryMode(keyRune("e"))
		*m = next.(Model)
	}, "Type 'eq' to trash it now: e_")

	if promptRow != statusRow {
		t.Errorf("prompt on line %d, status on line %d — they must share the row", promptRow, statusRow)
	}
}

// sendTop drives the real entry point. The interception these tests guard
// against happens in Update, above updateInventoryMode, so going through the
// handler directly cannot see it.
func sendTop(t *testing.T, m Model, msgs ...tea.KeyMsg) Model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

// Global bindings must not eat characters aimed at a prompt: 'r' is the lock
// key, and a profile whose name contains one has to stay deletable.
func TestInventoryDelete_TypesKeysBoundElsewhere(t *testing.T) {
	for _, key := range []string{"r", "o", "p", "i", "e", "m", "q"} {
		t.Run("types "+key, func(t *testing.T) {
			m, _ := editInventoryModel(t)

			m = sendTop(t, m, keyType(tea.KeyDelete), keyRune(key))

			if m.inventory.pending == nil {
				t.Fatalf("%q disarmed the prompt instead of typing", key)
			}
			if m.inventory.pending.typed != key {
				t.Errorf("typed = %q, want %q", m.inventory.pending.typed, key)
			}
		})
	}
}

func TestInventoryDelete_SpaceTypesOneSpace(t *testing.T) {
	m, _ := editInventoryModel(t)

	m = sendTop(t, m, keyType(tea.KeyDelete),
		keyRune("a"), tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, keyRune("b"))

	if got := m.inventory.pending.typed; got != "a b" {
		t.Errorf("typed = %q, want %q", got, "a b")
	}
}

// The awkward-name end-to-end: every bound letter, a space, digits and
// punctuation, typed through Update and confirmed.
func TestInventoryDelete_AwkwardNameThroughUpdate(t *testing.T) {
	m, dir := editInventoryModel(t)
	const name = "rope mix-9_2.v"
	if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), []byte("x = 1\ny = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.inventory.State.Visible = append(m.inventory.State.Visible, name+".profile.toml")
	m.inventory.cursor = len(m.inventory.State.Visible) - 1

	keys := []tea.KeyMsg{keyType(tea.KeyDelete)}
	for _, r := range name {
		if r == ' ' {
			keys = append(keys, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		keys = append(keys, keyRune(string(r)))
	}
	keys = append(keys, keyType(tea.KeyEnter))

	m = sendTop(t, m, keys...)

	if _, err := os.Stat(filepath.Join(dir, name+".profile.toml")); !os.IsNotExist(err) {
		t.Errorf("the profile should be trashed; status was %q", m.inventory.status)
	}
	if got := trashed(t, dir); len(got) != 1 {
		t.Errorf("trash = %v, want one entry", got)
	}
}

func TestInventoryDelete_CtrlCStillQuits(t *testing.T) {
	m, _ := editInventoryModel(t)

	m = sendTop(t, m, keyType(tea.KeyDelete), tea.KeyMsg{Type: tea.KeyCtrlC})

	if !m.quitting {
		t.Error("ctrl+c must still quit while a prompt is armed")
	}
}

// Glassroot gates the inventory, so an armed prompt should be impossible in a
// kiosk session. Arm one anyway and confirm the destructive step refuses on
// its own — the gate above it must never be the only thing standing there.
func TestInventoryDelete_GlassrootCannotConfirm(t *testing.T) {
	m, dir := editInventoryModel(t)
	m.GlassrootMode = true
	m.inventory.pending = &pendingDelete{rel: "eq.profile.toml", name: "eq"}

	m = sendTop(t, m, keyRune("eq"), keyType(tea.KeyEnter))

	if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); err != nil {
		t.Error("glassroot must not be able to complete a delete")
	}
	if got := trashed(t, dir); len(got) != 0 {
		t.Errorf("trash = %v, want empty", got)
	}
	if m.inventory.pending != nil {
		t.Error("the prompt should be discarded under glassroot")
	}
}

// A deck that is a link: the prompt and the result say only the link goes.
func TestInventoryDelete_LinkedDeck(t *testing.T) {
	m, dir := editInventoryModel(t)
	orig := filepath.Join(t.TempDir(), "eq.profile.toml")
	inDir := filepath.Join(dir, "eq.profile.toml")
	if err := os.Rename(inDir, orig); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(orig, inDir); err != nil {
		t.Fatal(err)
	}

	m.termWidth = 100
	m = send(t, m, keyType(tea.KeyDelete))
	if view := m.viewInventoryMode(m.keyHint()); !strings.Contains(view, "Type 'eq' to trash its link (the file stays): _") {
		t.Errorf("the prompt should say only the link goes:\n%s", view)
	}
	m = send(t, m, keyRune("eq"), keyType(tea.KeyEnter))
	if want := "Trashed the link eq → trash/ · the file stays"; m.inventory.status != want {
		t.Errorf("status = %q, want %q", m.inventory.status, want)
	}
	if _, err := os.Stat(orig); err != nil {
		t.Errorf("the linked file should stay: %v", err)
	}
}

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/settings"
)

// stagedModel is an inventory with alpha and beta equipped and gamma stashed,
// as on disk; alpha is the deck in use.
func stagedModel() Model {
	cfg := settings.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	visible := []string{"alpha.profile.toml", "beta.profile.toml"}
	stashed := []string{"gamma.profile.toml"}
	return Model{Config: cfg, styles: BuildStyles(cfg, ""), mode: inventoryMode, termWidth: 120, termHeight: 40,
		profile: profileState{profiles: []lair.ProfileInfo{{Name: "alpha"}, {Name: "beta"}}},
		inventory: inventoryModel{
			State: equip.NewInventoryState(append([]string(nil), visible...), append([]string(nil), stashed...), 9),
			seen:  [2][]string{visible, stashed},
		}}
}

func TestStagedSaysWhatApplyWouldChange(t *testing.T) {
	m := stagedModel()
	if got := m.inventory.staged(); got != "" {
		t.Fatalf("nothing is staged yet, got %q", got)
	}
	m = send(t, m, keyType(tea.KeyEnter)) // lift alpha
	if got := m.inventory.staged(); got != "1 move" {
		t.Errorf("a held deck is a move, got %q", got)
	}
	m = send(t, m, keyType(tea.KeyEnter)) // put it back
	if got := m.inventory.staged(); got != "" {
		t.Errorf("alpha back in its place is no change, got %q", got)
	}
	m = send(t, m, keyType(tea.KeyRight), keyType(tea.KeyEnter), keyType(tea.KeyLeft), keyType(tea.KeyEnter)) // lift beta, place it before alpha
	if got := m.inventory.staged(); got != "new order" {
		t.Errorf("beta before alpha is a new order, got %q", got)
	}
	m = stagedModel()
	m = send(t, m, keyType(tea.KeyEnter), keyType(tea.KeyDown), keyType(tea.KeyEnter)) // stash alpha
	if got := m.inventory.staged(); got != "1 move" {
		t.Errorf("stashing alpha is one move, got %q", got)
	}
	if out := ansi.Strip(m.viewInventoryMode("")); !strings.Contains(out, "1 move waiting — nothing changes until Apply") {
		t.Errorf("the Apply button should say a move is waiting:\n%s", out)
	}
}

// U-04, V-02: Esc used to leave and drop staged moves without a word. The
// first Esc now stays and says what would be lost; the second leaves.
func TestEscWithStagedMovesAsksOnce(t *testing.T) {
	m := stagedModel()
	m = send(t, m, keyType(tea.KeyEnter), keyType(tea.KeyDown), keyType(tea.KeyEnter)) // stash alpha

	m = send(t, m, keyType(tea.KeyEsc))
	if m.mode != inventoryMode || m.inventory.status != "1 move not applied — Esc again to discard" {
		t.Fatalf("first Esc: mode %v, status %q", m.mode, m.inventory.status)
	}
	// Any other key takes the warning back: Esc after it asks again.
	m = send(t, m, keyType(tea.KeyLeft), keyType(tea.KeyEsc))
	if m.mode != inventoryMode {
		t.Fatal("Esc after another key must ask again, not leave")
	}
	m = send(t, m, keyType(tea.KeyEsc))
	if m.mode != gridMode {
		t.Errorf("second Esc should leave, mode = %v", m.mode)
	}
}

func TestEscWithNothingStagedLeavesAtOnce(t *testing.T) {
	if m := send(t, stagedModel(), keyType(tea.KeyEsc)); m.mode != gridMode {
		t.Errorf("mode = %v, want the grid", m.mode)
	}
}

func TestRescueButtonWithStagedMovesAsksOnce(t *testing.T) {
	m := stagedModel()
	m = send(t, m, keyType(tea.KeyEnter), keyType(tea.KeyDown), keyType(tea.KeyEnter)) // stash alpha
	m.inventory.focusedList = focusRescue
	m = send(t, m, keyType(tea.KeyEnter))
	if m.mode != inventoryMode || !strings.Contains(m.inventory.status, "Enter again to discard") {
		t.Fatalf("first Enter: mode %v, status %q", m.mode, m.inventory.status)
	}
	if m = send(t, m, keyType(tea.KeyEnter)); m.mode != gridMode {
		t.Errorf("second Enter should go to rescue, mode = %v", m.mode)
	}
}

// V-04: the deck in use is marked, and the status line says what the deck
// under the cursor is.
func TestInventoryMarksTheActiveAndPinnedDeck(t *testing.T) {
	m := stagedModel()
	m.profile.pivotName = "beta"
	out := ansi.Strip(m.viewInventoryMode(""))
	if !strings.Contains(out, "▸ alpha.profile.toml") || strings.Contains(out, "▸ beta") {
		t.Errorf("only alpha, the deck in use, carries the mark:\n%s", out)
	}
	if !strings.Contains(out, "active deck") {
		t.Errorf("the cursor is on alpha; the status line should say it is the active deck:\n%s", out)
	}
	m = send(t, m, keyType(tea.KeyRight))
	if out := ansi.Strip(m.viewInventoryMode("")); !strings.Contains(out, "pinned — unpin with r in the grid") {
		t.Errorf("the cursor is on beta, the pinned deck:\n%s", out)
	}
}

// A held deck is drawn where Enter will put it, and left/right carry it one
// place at a time, the place after the last deck included. It used to land
// before the deck under the cursor, which could never be past the last one.
func TestHeldDeckIsDrawnWhereItLands(t *testing.T) {
	m := stagedModel()
	m.inventory.State.Visible = append(m.inventory.State.Visible, "delta.profile.toml")
	m.inventory.seen[0] = append(m.inventory.seen[0], "delta.profile.toml")

	row := func(m Model) string {
		for _, line := range strings.Split(ansi.Strip(m.viewInventoryMode("")), "\n") {
			if strings.Contains(line, "beta.profile.toml") {
				return strings.Join(strings.Fields(strings.ReplaceAll(line, "┃", " ")), " ")
			}
		}
		return ""
	}
	m = send(t, m, keyType(tea.KeyEnter)) // lift alpha: it stays drawn where it was
	if got, want := row(m), "« ▸ alpha.profile.toml » beta.profile.toml delta.profile.toml"; got != want {
		t.Errorf("lifted:\n got %s\nwant %s", got, want)
	}
	m = send(t, m, keyType(tea.KeyRight))
	if got, want := row(m), "beta.profile.toml « ▸ alpha.profile.toml » delta.profile.toml"; got != want {
		t.Errorf("one right:\n got %s\nwant %s", got, want)
	}
	m = send(t, m, keyType(tea.KeyRight), keyType(tea.KeyRight), keyType(tea.KeyRight)) // the end, and no further
	if got, want := row(m), "beta.profile.toml delta.profile.toml « ▸ alpha.profile.toml »"; got != want {
		t.Errorf("at the end:\n got %s\nwant %s", got, want)
	}
	m = send(t, m, keyType(tea.KeyEnter))
	want := []string{"beta.profile.toml", "delta.profile.toml", "alpha.profile.toml"}
	if got := m.inventory.State.Visible; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("placed: %v, want %v", got, want)
	}
	if got, want := row(m), "beta.profile.toml delta.profile.toml ▸ alpha.profile.toml"; got != want {
		t.Errorf("after placing the cursor is on the deck:\n got %s\nwant %s", got, want)
	}
}

// The held deck goes with the cursor into the other list, an empty one too,
// but not down to the buttons, where there is nowhere to draw it.
func TestHeldDeckFollowsIntoTheOtherList(t *testing.T) {
	m := stagedModel()
	m.inventory.State.Inventory, m.inventory.seen[1] = nil, nil
	m = send(t, m, keyType(tea.KeyEnter), keyType(tea.KeyDown)) // lift alpha, carry it to the empty stash
	out := ansi.Strip(m.viewInventoryMode(""))
	if strings.Contains(out, "(empty)") || strings.Count(out, "« ▸ alpha.profile.toml »") != 1 {
		t.Errorf("the held deck stands in the empty list, once:\n%s", out)
	}
	m = send(t, m, keyType(tea.KeyDown))
	if m.inventory.focusedList != equip.ListInventory || m.inventory.status != "Place the deck first" {
		t.Errorf("focus = %d, status = %q", m.inventory.focusedList, m.inventory.status)
	}
	if strings.Contains(out, "Holding") {
		t.Error("the Holding line is gone: the list shows the held deck")
	}
}

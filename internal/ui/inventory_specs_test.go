package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// specsInventoryModel builds an inventory sitting on a real config dir: the
// named profiles exist as files, and specs maps a spec name to its file body.
func specsInventoryModel(t *testing.T, equipped, stashed []string, specs map[string]string) (Model, string) {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"inventory", "specs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	valid := "x = 1\ny = 1\n[[commands]]\nname = \"a\"\ncol = \"A\"\nrow = 0\n"

	var vis, inv []string
	for _, n := range equipped {
		file := n + profiles.ProfileSuffix
		if err := os.WriteFile(filepath.Join(dir, file), []byte(valid), 0o644); err != nil {
			t.Fatal(err)
		}
		vis = append(vis, file)
	}
	for _, n := range stashed {
		file := n + profiles.ProfileSuffix
		if err := os.WriteFile(filepath.Join(paths.InventoryDir(dir), file), []byte(valid), 0o644); err != nil {
			t.Fatal(err)
		}
		inv = append(inv, file)
	}
	for name, body := range specs {
		if err := os.WriteFile(filepath.Join(paths.SpecsDir(dir), name+profiles.SpecSuffix), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	keys := config.InputConfig{EditFile: "e", Delete: "delete", Lock: "r", PathGridMode: "tab"}
	keys.InitControls()
	m := Model{
		mode:       inventoryMode,
		termWidth:  100,
		termHeight: 40,
		profile:    profileState{configDir: dir},
		Config:     config.Config{Keys: keys},
		styles:     BuildStyles(config.Config{Keys: keys}),
		inventory: inventoryModel{
			State:       core.NewInventoryState(vis, inv, profiles.MaxEquipped),
			focusedList: core.ListVisible,
		},
	}
	return m, dir
}

func specList(profs ...string) string {
	return "profiles = [\"" + strings.Join(profs, "\", \"") + "\"]\n"
}

// planFor finds the plan for one verb+spec, so tests don't depend on ordering.
func planFor(t *testing.T, m Model, verb specVerb, name string) specPlan {
	t.Helper()
	for _, p := range m.inventory.specs.plans {
		if p.verb == verb && p.name == name {
			return p
		}
	}
	t.Fatalf("no plan for verb %d name %q in %+v", verb, name, m.inventory.specs.plans)
	return specPlan{}
}

// verdictFor is the picker's answer about one plan: the kind is the policy, the
// text is only how it is worded. Tests assert on the kind wherever they can.
func verdictFor(t *testing.T, m Model, verb specVerb, name string) (string, verdictKind) {
	t.Helper()
	return m.planVerdict(planFor(t, m, verb, name))
}

// selectRow points the overlay cursor at one row and presses enter.
func selectRow(t *testing.T, m Model, verb specVerb, name string) Model {
	t.Helper()
	for i, r := range m.inventory.specs.plans {
		if r.verb == verb && r.name == name {
			m.inventory.specs.cursor = i
			return send(t, m, keyType(tea.KeyEnter))
		}
	}
	t.Fatalf("no row for verb %d name %q", verb, name)
	return m
}

func names(files []string) string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = strings.TrimSuffix(f, profiles.ProfileSuffix)
	}
	return strings.Join(out, " ")
}

func TestSpecsOverlay_TabOpensAndCloses(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	if m.inventory.specs == nil {
		t.Fatal("tab should open the spec overlay")
	}

	m = send(t, m, keyType(tea.KeyTab))
	if m.inventory.specs != nil {
		t.Error("tab should close the spec overlay again")
	}
}

func TestSpecsOverlay_EscClosesWithoutChanging(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab), keyType(tea.KeyEsc))

	if m.inventory.specs != nil {
		t.Error("esc should close the overlay")
	}
	if m.mode != inventoryMode {
		t.Error("esc in the overlay must not leave the inventory")
	}
	if names(m.inventory.State.Visible) != "eq" {
		t.Errorf("Visible = %v, want untouched", m.inventory.State.Visible)
	}
}

func TestSpecsOverlay_TabRefusesWhileHolding(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})
	if err := m.inventory.State.PickUpItem(core.ListVisible, 0); err != nil {
		t.Fatal(err)
	}

	m = send(t, m, keyType(tea.KeyTab))

	if m.inventory.specs != nil {
		t.Error("the overlay must not open while an item is held")
	}
	if !strings.Contains(m.inventory.status, "held") {
		t.Errorf("status = %q, want it to mention the held item", m.inventory.status)
	}
}

func TestSpecRows_OneRowPerFunction(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
		"ops": specList("eq"),
	})

	m = send(t, m, keyType(tea.KeyTab))

	plans := m.inventory.specs.plans
	if len(plans) != 5 {
		t.Fatalf("got %d rows, want 2 equip + 2 stash + 1 strip: %+v", len(plans), plans)
	}
	// Grouped by verb so the headers render from row transitions.
	wantVerbs := []specVerb{specEquip, specEquip, specStash, specStash, specStrip}
	for i, want := range wantVerbs {
		if plans[i].verb != want {
			t.Errorf("row %d verb = %d, want %d", i, plans[i].verb, want)
		}
	}
	if _, kind := m.planVerdict(plans[4]); kind != verdictNone {
		t.Errorf("strip should have nothing to report, got kind %d", kind)
	}
}

func TestSpecRows_StripOfferedWithNoSpecs(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, nil, nil)

	m = send(t, m, keyType(tea.KeyTab))

	plans := m.inventory.specs.plans
	if len(plans) != 1 || plans[0].verb != specStrip {
		t.Fatalf("want just the strip row, got %+v", plans)
	}
}

// A spec naming a deck you never installed is not an error. Reconcile ignores
// names it has no file for, and "drako spec" equips the rest without comment —
// so the picker equips the rest too, and says what it skipped.
func TestSpecRows_IncompleteSpecWarnsButStillRuns(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"ops": specList("st", "ghost"),
	})

	m = send(t, m, keyType(tea.KeyTab))

	equip := planFor(t, m, specEquip, "ops")
	if strings.Join(equip.missing, ",") != "ghost" {
		t.Errorf("missing = %v, want [ghost]", equip.missing)
	}
	if strings.Join(equip.desired, ",") != "st.profile.toml" {
		t.Errorf("desired = %v, want the deck that does exist", equip.desired)
	}
	_, kind := verdictFor(t, m, specEquip, "ops")
	if kind != verdictWarn {
		t.Errorf("kind = %d, want verdictWarn — an incomplete spec still runs", kind)
	}

	// Stashing decks you do have is still legal.
	if _, kind := verdictFor(t, m, specStash, "ops"); kind == verdictBlocked {
		t.Error("the stash plan should stay usable")
	}
}

func TestSpecsOverlay_IncompleteSpecEquipsWhatExists(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"ops": specList("st", "ghost"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specEquip, "ops")

	if m.inventory.specs != nil {
		t.Error("an incomplete spec should still stage and close the overlay")
	}
	if got := names(m.inventory.State.Visible); got != "st" {
		t.Errorf("Visible = %q, want the deck that does exist", got)
	}
	if !strings.Contains(m.inventory.status, "ghost") {
		t.Errorf("status = %q, want it to say what was skipped", m.inventory.status)
	}
}

// The warning has to be readable before you commit to the row, not only after.
func TestSpecsOverlay_VerdictShowsOnSelection(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"ops": specList("st", "ghost"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	for i, p := range m.inventory.specs.plans {
		if p.verb == specEquip && p.name == "ops" {
			m.inventory.specs.cursor = i
		}
	}

	// No key has been pressed against the row — only landed on.
	if !strings.Contains(m.View(), "ghost") {
		t.Error("the popup should show the warning for the selected row")
	}
}

func TestSpecRows_DimsOverCap(t *testing.T) {
	var many []string
	for i := range profiles.MaxEquipped + 1 {
		many = append(many, string(rune('a'+i)))
	}
	m, _ := specsInventoryModel(t, nil, many, map[string]string{
		"big": specList(many...),
	})

	m = send(t, m, keyType(tea.KeyTab))

	text, kind := verdictFor(t, m, specEquip, "big")
	if kind != verdictBlocked {
		t.Fatal("a spec over the equip cap must be blocked before it can be staged")
	}
	if !strings.Contains(text, "9") {
		t.Errorf("verdict = %q, want it to name the limit", text)
	}
}

func TestSpecRows_DimsUnreadableSpec(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, nil, map[string]string{
		"broken": "profiles = [\"eq\"\n",
	})

	m = send(t, m, keyType(tea.KeyTab))

	for _, verb := range []specVerb{specEquip, specStash} {
		if _, kind := verdictFor(t, m, verb, "broken"); kind != verdictBlocked {
			t.Errorf("verb %d: an unreadable spec must be blocked", verb)
		}
	}
}

func TestSpecsOverlay_EquipStagesTheSpec(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st", "other"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specEquip, "dev")

	if m.inventory.specs != nil {
		t.Error("staging should close the overlay")
	}
	// Apply is the only thing left to do, so that is where you land.
	if m.inventory.focusedList != focusApply {
		t.Errorf("focusedList = %d, want the Apply button (%d)", m.inventory.focusedList, focusApply)
	}
	if got := names(m.inventory.State.Visible); got != "st" {
		t.Errorf("Visible = %q, want %q", got, "st")
	}
	if got := names(m.inventory.State.Inventory); got != "eq other" {
		t.Errorf("Inventory = %q, want %q", got, "eq other")
	}
	if !strings.Contains(m.inventory.status, "dev") {
		t.Errorf("status = %q, want it to name the spec", m.inventory.status)
	}
}

// Nothing may reach disk before Apply: the whole point of staging is that esc
// is a working undo.
func TestSpecsOverlay_EquipTouchesNoFiles(t *testing.T) {
	m, dir := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	_ = selectRow(t, m, specEquip, "dev")

	if _, err := os.Stat(filepath.Join(dir, "eq.profile.toml")); err != nil {
		t.Error("the equipped file must still be where it was")
	}
	if _, err := os.Stat(filepath.Join(paths.InventoryDir(dir), "st.profile.toml")); err != nil {
		t.Error("the stashed file must still be where it was")
	}
}

func TestSpecsOverlay_StashRemovesOnlyTheSpecsDecks(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq", "keep"}, []string{"st"}, map[string]string{
		"dev": specList("eq", "st"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specStash, "dev")

	if got := names(m.inventory.State.Visible); got != "keep" {
		t.Errorf("Visible = %q, want %q", got, "keep")
	}
	if got := names(m.inventory.State.Inventory); got != "eq st" {
		t.Errorf("Inventory = %q, want %q", got, "eq st")
	}
}

func TestSpecsOverlay_StripStashesEverything(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq", "two"}, []string{"st"}, nil)

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specStrip, "")

	if len(m.inventory.State.Visible) != 0 {
		t.Errorf("Visible = %v, want empty", m.inventory.State.Visible)
	}
	if got := names(m.inventory.State.Inventory); got != "eq st two" {
		t.Errorf("Inventory = %q, want %q", got, "eq st two")
	}
}

func TestSpecsOverlay_BlockedRowStagesNothing(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"bad": "profiles = [\"st\"\n",
	})

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specEquip, "bad")

	if m.inventory.specs == nil {
		t.Fatal("a refused row must leave the overlay open so another can be picked")
	}
	if text, kind := verdictFor(t, m, specEquip, "bad"); kind != verdictBlocked ||
		!strings.Contains(text, "bad") {
		t.Errorf("verdict = %q (kind %d), want a blocked verdict naming the spec", text, kind)
	}
	if names(m.inventory.State.Visible) != "eq" {
		t.Errorf("Visible = %v, want untouched", m.inventory.State.Visible)
	}
}

// The inventory already refuses to lift the locked profile by hand; a bulk
// function must not be the way around that.
func TestSpecsOverlay_LockedProfileNeverMoves(t *testing.T) {
	t.Run("equip keeps it", func(t *testing.T) {
		m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
			"dev": specList("st"),
		})
		m.profile.pivotName = "eq"

		m = send(t, m, keyType(tea.KeyTab))
		m = selectRow(t, m, specEquip, "dev")

		if got := names(m.inventory.State.Visible); got != "eq st" {
			t.Errorf("Visible = %q, want the locked deck kept alongside the spec", got)
		}
		if !strings.Contains(m.inventory.status, "locked") {
			t.Errorf("status = %q, want it to say the locked deck was kept", m.inventory.status)
		}
	})

	t.Run("stash keeps it", func(t *testing.T) {
		m, _ := specsInventoryModel(t, []string{"eq"}, nil, map[string]string{
			"dev": specList("eq"),
		})
		m.profile.pivotName = "eq"

		m = send(t, m, keyType(tea.KeyTab))
		m = selectRow(t, m, specStash, "dev")

		if got := names(m.inventory.State.Visible); got != "eq" {
			t.Errorf("Visible = %q, want the locked deck kept", got)
		}
	})

	t.Run("strip keeps it", func(t *testing.T) {
		m, _ := specsInventoryModel(t, []string{"eq", "two"}, nil, nil)
		m.profile.pivotName = "eq"

		m = send(t, m, keyType(tea.KeyTab))
		m = selectRow(t, m, specStrip, "")

		if got := names(m.inventory.State.Visible); got != "eq" {
			t.Errorf("Visible = %q, want just the locked deck", got)
		}
	})
}

// The locked deck occupies a slot, so a full spec plus a lock is over the cap
// and has to be refused up front like any other over-cap arrangement.
func TestSpecRows_LockedDeckCountsAgainstTheCap(t *testing.T) {
	var full []string
	for i := range profiles.MaxEquipped {
		full = append(full, string(rune('a'+i)))
	}
	m, _ := specsInventoryModel(t, []string{"locked"}, full, map[string]string{
		"full": specList(full...),
	})
	m.profile.pivotName = "locked"

	m = send(t, m, keyType(tea.KeyTab))

	if _, kind := verdictFor(t, m, specEquip, "full"); kind != verdictBlocked {
		t.Error("9 spec decks plus the locked one is 10 — must be refused")
	}
}

func TestSpecsOverlay_NavigationStaysInRange(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab), keyType(tea.KeyUp))
	if m.inventory.specs.cursor != 0 {
		t.Errorf("cursor = %d, want it clamped at the top", m.inventory.specs.cursor)
	}

	for range 10 {
		m = send(t, m, keyType(tea.KeyDown))
	}
	if want := len(m.inventory.specs.plans) - 1; m.inventory.specs.cursor != want {
		t.Errorf("cursor = %d, want it clamped at %d", m.inventory.specs.cursor, want)
	}
}

func TestSpecsPopup_RendersEveryGroup(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	out := m.View()

	for _, want := range []string{"Spec functions", "EQUIP", "STASH", "STRIP", "dev", "enter: equip"} {
		if !strings.Contains(out, want) {
			t.Errorf("popup missing %q. Got:\n%s", want, out)
		}
	}
	// The inventory stays underneath — the picker is an overlay, not a screen.
	if !strings.Contains(out, "Equipped Items") {
		t.Error("the inventory should still be visible behind the popup")
	}
}

func TestSpecsPopup_ShowsTheRefusalReason(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, nil, map[string]string{
		"bad": "profiles = [\"eq\"\n",
	})

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specEquip, "bad")

	out := m.View()
	if !strings.Contains(out, "does not parse") {
		t.Errorf("popup should show why the row cannot run. Got:\n%s", out)
	}
	if m.inventory.specs == nil {
		t.Error("a blocked row must leave the overlay open")
	}
}

func TestSpecsPopup_FitsTheTerminal(t *testing.T) {
	specs := map[string]string{}
	for i := range 12 {
		specs[fmt.Sprintf("spec%02d", i)] = specList("st")
	}
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, specs)
	m.termWidth, m.termHeight = 80, 24

	m = send(t, m, keyType(tea.KeyTab))
	popup := m.renderSpecsPopup()

	if h := lipgloss.Height(popup); h > m.termHeight {
		t.Errorf("popup is %d lines tall, must fit in termHeight = %d", h, m.termHeight)
	}
	if !strings.Contains(popup, "▾") {
		t.Error("a spec list taller than the popup should show the scroll marker")
	}
}

// The status line carried only refusals before specs existed, so it was
// unconditionally red. A staged arrangement is the opposite of a refusal.
func TestSpecsOverlay_StagedStatusReadsAsSuccess(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"dev": specList("st"),
	})

	m = send(t, m, keyType(tea.KeyTab))
	m = selectRow(t, m, specEquip, "dev")
	if !m.inventory.statusOK {
		t.Error("a staged spec should read as a confirmation, not a refusal")
	}

	// And the next keystroke clears it, like every other inventory message.
	m = send(t, m, keyType(tea.KeyDown))
	if m.inventory.status != "" || m.inventory.statusOK {
		t.Errorf("status should reset, got %q (ok=%v)", m.inventory.status, m.inventory.statusOK)
	}
}

// The box is sized by the terminal, not by what it happens to contain. If the
// content drives it, every keystroke that changes a message resizes the border
// and the popup appears to shake as you move through it.
func TestSpecsPopup_SizeDoesNotChangeWithTheCursor(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"a":                                   specList("st"),
		"ops":                                 specList("st", "ghost"), // carries a warning
		"a-spec-with-a-very-long-name-indeed": specList("st"),
		"bad":                                 "profiles = [\"st\"\n", // carries a long parse error
	})

	m = send(t, m, keyType(tea.KeyTab))

	var wantW, wantH int
	for i := range m.inventory.specs.plans {
		m.inventory.specs.cursor = i
		popup := m.renderSpecsPopup()
		w, h := lipgloss.Width(popup), lipgloss.Height(popup)
		if i == 0 {
			wantW, wantH = w, h
			continue
		}
		if w != wantW || h != wantH {
			t.Fatalf("row %d (%s): popup is %dx%d, want a steady %dx%d",
				i, m.inventory.specs.plans[i].label(), w, h, wantW, wantH)
		}
	}
}

func TestSpecsPopup_NarrowTerminalStillFits(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
		"ops": specList("st", "ghost"),
	})
	m.termWidth, m.termHeight = 46, 24

	m = send(t, m, keyType(tea.KeyTab))
	popup := m.renderSpecsPopup()

	if w := lipgloss.Width(popup); w > m.termWidth {
		t.Errorf("popup is %d wide, must fit in termWidth = %d:\n%s", w, m.termWidth, popup)
	}
	// Wrapped, not cut short: the reason survives a narrow box.
	if !strings.Contains(popup, "ghost") {
		t.Errorf("the warning should still be readable at 46 columns:\n%s", popup)
	}
}

// A row shows a preview, not the list. Cutting mid-name hides how much is
// hidden — the reader can't tell a four-deck spec from a twenty-deck one.
func TestSpecDetailFit(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		width  int
		want   string
	}{
		{"fits untouched", "git rust k8s", 40, "git rust k8s"},
		{"exactly fits", "git rust k8s", 12, "git rust k8s"},
		{"drops whole names and counts them", "alpha bravo charlie delta echo", 20, "alpha bravo +3"},
		{"counts every name when none fit", "aaaaaaaaaaaa bbbbbbbbbbbb", 8, "2 decks"},
		{"single long name still truncates", "aaaaaaaaaaaaaaaaaaaa", 8, "aaaaa..."},
		{"empty stays empty", "", 20, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := specDetailFit(tc.detail, tc.width)
			if got != tc.want {
				t.Errorf("specDetailFit(%q, %d) = %q, want %q", tc.detail, tc.width, got, tc.want)
			}
			if lipgloss.Width(got) > tc.width && tc.width > 0 {
				t.Errorf("result %q is wider than the %d columns it was given", got, tc.width)
			}
		})
	}
}

func TestSummarizeNames(t *testing.T) {
	if got := summarizeNames([]string{"a", "b"}); got != "a, b" {
		t.Errorf("short lists print in full, got %q", got)
	}
	if got := summarizeNames([]string{"a", "b", "c", "d", "e"}); got != "a, b, c and 2 more" {
		t.Errorf("long lists should count the tail, got %q", got)
	}
}

// Twenty missing decks must not become a wall of names in a reserved two-line
// slot — and the row must still say how many it could not show.
func TestSpecsOverlay_ManyProfilesStayReadable(t *testing.T) {
	var many []string
	for i := range 20 {
		many = append(many, fmt.Sprintf("deck%02d", i))
	}
	m, _ := specsInventoryModel(t, []string{"eq"}, nil, map[string]string{
		"huge": specList(many...),
	})

	m = send(t, m, keyType(tea.KeyTab))
	if got := len(planFor(t, m, specEquip, "huge").missing); got != 20 {
		t.Errorf("missing = %d, want all 20 recorded as fact", got)
	}
	text, _ := verdictFor(t, m, specEquip, "huge")
	if !strings.Contains(text, "and 17 more") {
		t.Errorf("verdict = %q, want it to count the tail rather than list it", text)
	}

	// How many fit depends on the box width; that the row counts the rest
	// rather than trailing off does not.
	popup := m.renderSpecsPopup()
	if !regexp.MustCompile(`\+\d+`).MatchString(popup) {
		t.Errorf("the row should say how many decks it could not show:\n%s", popup)
	}
	if strings.Contains(popup, "deck19") {
		t.Errorf("a 20-deck list must not be spelled out in the popup:\n%s", popup)
	}
}

// Landing on Apply means the very next keypress commits, with no walk through
// the lists where the staged arrangement could be disturbed.
func TestSpecsOverlay_EnterLandsOnApply(t *testing.T) {
	for _, tc := range []struct {
		name string
		verb specVerb
		spec string
	}{
		{"equip", specEquip, "dev"},
		{"stash", specStash, "dev"},
		{"strip", specStrip, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
				"dev": specList("st"),
			})

			m = send(t, m, keyType(tea.KeyTab))
			m = selectRow(t, m, tc.verb, tc.spec)

			if m.inventory.focusedList != focusApply {
				t.Errorf("focusedList = %d, want focusApply (%d)", m.inventory.focusedList, focusApply)
			}
			// And enter there commits, rather than lifting something.
			_, cmd := m.updateInventoryMode(keyType(tea.KeyEnter))
			if cmd == nil {
				t.Error("enter on the landing spot should run the apply command")
			}
		})
	}
}

// A minimum width that outranks the terminal is not a minimum, it is an
// overflow: the box loses its right border off the edge of the screen.
func TestSpecsPopup_NeverWiderThanTheTerminal(t *testing.T) {
	for _, w := range []int{120, 100, 80, 60, 46, 40, 30, 20} {
		m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{
			"dev": specList("st"),
			"ops": specList("st", "ghost"),
		})
		m.termWidth, m.termHeight = w, 24

		m = send(t, m, keyType(tea.KeyTab))
		if got := lipgloss.Width(m.renderSpecsPopup()); got > w {
			t.Errorf("at %d columns the popup is %d wide:\n%s", w, got, m.renderSpecsPopup())
		}
	}
}

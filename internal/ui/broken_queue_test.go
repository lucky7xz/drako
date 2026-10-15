package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

func brokenErr(name string) config.ProfileParseError {
	return config.ProfileParseError{
		Name: name,
		Path: "/tmp/" + name + ".profile.toml",
		Err:  "SYNTAX ERROR: boom",
	}
}

func queueModel(errs ...config.ProfileParseError) Model {
	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()
	return Model{
		mode:   gridMode,
		styles: BuildStyles(base),
		profile: profileState{
			pendingErrors:    errs,
			errorQueueActive: true,
			acknowledged:     map[string]bool{},
		},
	}
}

func TestBrokenQueuePresentsFirstError(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))

	m = m.presentNextBrokenProfile()

	if m.mode != infoMode {
		t.Fatalf("mode = %v, want infoMode", m.mode)
	}
	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "alpha") {
		t.Fatalf("detail should present alpha, got %+v", m.activeDetail)
	}
	if len(m.profile.pendingErrors) != 1 {
		t.Errorf("queue should have consumed one entry, %d left", len(m.profile.pendingErrors))
	}
	if !m.profile.acknowledged["/tmp/alpha.profile.toml"] {
		t.Error("presented error must be marked acknowledged")
	}
}

func TestBrokenQueueSkipsAcknowledged(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))
	m.profile.acknowledged["/tmp/alpha.profile.toml"] = true

	m = m.presentNextBrokenProfile()

	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "beta") {
		t.Fatalf("already-acknowledged alpha must be skipped, got %+v", m.activeDetail)
	}
}

func TestBrokenQueueExhaustionEntersRescue(t *testing.T) {
	m := queueModel() // active queue, nothing pending

	m = m.presentNextBrokenProfile()

	if m.mode != gridMode {
		t.Fatalf("mode = %v, want gridMode after exhaustion", m.mode)
	}
	if m.profile.errorQueueActive {
		t.Error("queue must deactivate once exhausted")
	}
	if len(m.Config.Commands) == 0 {
		t.Error("exhausting an active queue applies the rescue config (helper commands)")
	}
}

func TestBrokenQueueAckWalksToRescue(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))
	m = m.presentNextBrokenProfile()

	ack := tea.KeyMsg{Type: tea.KeyEnter}
	next, _ := m.updateInfoMode(ack)
	m = next.(Model)
	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "beta") {
		t.Fatalf("first ack should present beta, got %+v", m.activeDetail)
	}

	next, _ = m.updateInfoMode(ack)
	m = next.(Model)
	if m.mode != gridMode || m.profile.errorQueueActive {
		t.Fatalf("second ack should exhaust the queue into grid mode, mode=%v active=%v", m.mode, m.profile.errorQueueActive)
	}
}

// Exiting rescue mode reloads the config; with a broken config.toml the
// bundle still reports that error, but it was acknowledged on the way in. A
// drain that shows nothing must keep the config the caller just applied
// instead of bouncing straight back into rescue.
func TestBrokenQueueAllAcknowledgedKeepsReloadedConfig(t *testing.T) {
	m := queueModel(brokenErr("alpha"))
	m.profile.acknowledged["/tmp/alpha.profile.toml"] = true

	reloaded := config.Config{X: 1, Y: 1}
	reloaded.ApplyDefaults()
	reloaded.Commands = []config.Command{{Name: "my cell", Command: "true", Row: 0, Col: "a"}}
	m.applyConfig(reloaded)

	m = m.presentNextBrokenProfile()

	if len(m.Config.Commands) != 1 || m.Config.Commands[0].Name != "my cell" {
		t.Fatalf("nothing was shown, so the reloaded config must survive; got %+v", m.Config.Commands)
	}
	if m.profile.errorQueueActive {
		t.Error("queue must deactivate once exhausted")
	}
}

// A profile that was fixed and later breaks again must be shown again,
// not silently skipped as an already-seen duplicate.
func TestApplyBundleClearsAcknowledgedOnceFixed(t *testing.T) {
	m := queueModel()
	m.profile.acknowledged["/tmp/alpha.profile.toml"] = true

	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()
	m.applyBundle(config.ConfigBundle{Base: base, Config: base}) // Broken is empty: alpha is fixed

	m.profile.pendingErrors = []config.ProfileParseError{brokenErr("alpha")}
	m = m.presentNextBrokenProfile()

	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "alpha") {
		t.Fatalf("re-broken alpha should be shown again, got %+v", m.activeDetail)
	}
}

func TestApplyBundleAdoptsBundleState(t *testing.T) {
	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()
	effective := base
	effective.Commands = []config.Command{{Name: "noop", Command: "true", Row: 0, Col: "a"}}

	var m Model
	m.applyBundle(config.ConfigBundle{
		Base:        base,
		Config:      effective,
		Profiles:    []config.ProfileInfo{{Name: "alpha"}, {Name: "work"}},
		ActiveIndex: 1,
		ConfigDir:   "/tmp/cfg",
		LockedName:  "work",
	})

	if m.ActiveProfileName() != "work" {
		t.Errorf("active profile = %q, want work", m.ActiveProfileName())
	}
	if !m.profile.locked || m.profile.pivotName != "work" {
		t.Errorf("pivot lock not adopted: locked=%v name=%q", m.profile.locked, m.profile.pivotName)
	}
	if len(m.Config.Commands) != 1 {
		t.Errorf("effective config not applied: %+v", m.Config.Commands)
	}
}

func TestApplyBundleClampsBadActiveIndex(t *testing.T) {
	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()

	var m Model
	m.applyBundle(config.ConfigBundle{
		Base:        base,
		Config:      base,
		Profiles:    []config.ProfileInfo{{Name: "only"}},
		ActiveIndex: 7,
	})

	if m.profile.activeIndex != 0 {
		t.Errorf("out-of-range index must clamp to 0, got %d", m.profile.activeIndex)
	}
}

// Background messages (resize, ticks) must not advance a queue that is
// already on screen; only the user's keypress does.
func TestBrokenQueueWaitsForKeypress(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"), brokenErr("gamma"))
	m = m.presentNextBrokenProfile()

	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	if !strings.Contains(m.activeDetail.Title, "alpha") {
		t.Fatalf("resize advanced the queue: showing %q", m.activeDetail.Title)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !strings.Contains(m.activeDetail.Title, "beta") {
		t.Fatalf("one keypress should show beta, showing %q", m.activeDetail.Title)
	}
	if len(m.profile.pendingErrors) != 1 {
		t.Errorf("gamma should still be pending, %d left", len(m.profile.pendingErrors))
	}
}

// A clean reload under the error popup (the file got fixed) ends the queue;
// the next ordinary popup must not be mistaken for it and exit into rescue.
func TestCleanReloadEndsTheErrorQueue(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1, Commands: []config.Command{{Name: "hi", Command: "echo hi", Col: "a"}}}
	cfg.ApplyDefaults()
	m := queueModel(brokenErr("alpha"))
	m.Config, m.gridNav.grid = cfg, config.BuildGrid(cfg)
	m = m.presentNextBrokenProfile()

	m = m.applyReloadedBundle(config.ConfigBundle{Config: cfg})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}) // explain "hi"
	next, _ = next.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})           // close it
	m = next.(Model)

	if m.mode != gridMode || m.Config.Commands[0].Name != "hi" {
		t.Errorf("closing an explain popup after the fix landed in mode %v on %q", m.mode, m.Config.Commands[0].Name)
	}
}

// A dropped-profile note deferred under the lock waits its turn: it must not
// replace a broken-profile popup the user hasn't dismissed yet.
func TestDeferredDroppedNoteWaitsForTheQueue(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))
	m.profile.pendingDroppedProfile = "work"
	m = m.presentNextBrokenProfile()

	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40}) // any background message
	m = next.(Model)
	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "alpha") || !m.profile.errorQueueActive {
		t.Fatalf("the queue was interrupted: showing %q, queue active %v", m.activeDetail.Title, m.profile.errorQueueActive)
	}

	for range 2 { // dismiss alpha, then beta
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(Model)
	}
	if m.activeDetail == nil || m.activeDetail.Title != "Profile no longer available" {
		t.Errorf("after the queue the deferred note should show, got %+v", m.activeDetail)
	}
}

// pinnedWorkDeck bootstraps a config dir and pins a "work" deck in it.
func pinnedWorkDeck(t *testing.T) (dir string) {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if _, err := config.LoadConfig(nil); err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(xdg, "drako")
	deck := "x = 1\ny = 1\n[[commands]]\nname = \"hi\"\ncommand = \"echo hi\"\ncol = \"a\"\nrow = 0\n"
	if err := os.WriteFile(filepath.Join(dir, "work.profile.toml"), []byte(deck), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.WritePivotLocked(dir, "work"); err != nil {
		t.Fatal(err)
	}
	return dir
}

// dropWorkAndBreakAnother deletes the pinned deck and adds a broken one.
func dropWorkAndBreakAnother(t *testing.T, dir string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, "work.profile.toml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.profile.toml"), []byte("x = [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func expectBrokenThenDropped(t *testing.T, m Model) {
	t.Helper()
	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "bad") {
		t.Fatalf("want the broken-deck notice first, got %+v", m.activeDetail)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.activeDetail == nil || m.activeDetail.Title != "Profile no longer available" {
		t.Errorf("after the broken-deck notice, want the dropped-deck note, got %+v", m.activeDetail)
	}
}

// Deleting the active deck while another is broken: both get explained, the
// broken one first, whether a reload or a new lap finds them.
func TestDroppedNoteFollowsBrokenNoticeOnReload(t *testing.T) {
	dir := pinnedWorkDeck(t)
	m := InitialModel(false, Session{})
	dropWorkAndBreakAnother(t, dir)
	next, _ := m.Update(ConfigChangedMsg{Path: filepath.Join(dir, "bad.profile.toml")})
	expectBrokenThenDropped(t, next.(Model))
}

func TestDroppedNoteFollowsBrokenNoticeOnNewLap(t *testing.T) {
	dir := pinnedWorkDeck(t)
	dropWorkAndBreakAnother(t, dir)
	expectBrokenThenDropped(t, InitialModel(false, Session{}))
}

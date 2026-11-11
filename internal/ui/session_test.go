package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// sessionModel is a 3x2 deck. The cell names are what a Session carries.
func sessionModel(t *testing.T) Model {
	t.Helper()
	cfg := settings.Config{X: 3, Y: 2, Commands: []profile.Command{
		{Name: "update", Command: "echo 1", Row: 0, Col: "a"},
		{Name: "monitor", Command: "echo 2", Row: 0, Col: "b"},
		{Name: "finder", Command: "echo 3", Row: 0, Col: "c"},
		{Name: "bandwidth", Command: "echo 4", Row: 1, Col: "b"},
	}}
	cfg.ApplyDefaults()
	return Model{
		Config:  cfg,
		styles:  BuildStyles(cfg, ""),
		gridNav: gridNav{grid: profile.BuildGrid(cfg.X, cfg.Y, cfg.Commands)},
	}
}

// at finds a cell by name, so the tests never hard-code coordinates the grid
// builder is free to change.
func at(t *testing.T, m Model, name string) (int, int) {
	t.Helper()
	for row, cells := range m.gridNav.grid {
		for col, got := range cells {
			if got == name {
				return row, col
			}
		}
	}
	t.Fatalf("no cell named %q in %v", name, m.gridNav.grid)
	return 0, 0
}

// The whole point: run a command from the middle of a deck and come back to it.
func TestSessionRoundTrip(t *testing.T) {
	before := sessionModel(t)
	row, col := at(t, before, "bandwidth")
	before.gridNav.cursorRow, before.gridNav.cursorCol = row, col

	s := before.Session()
	if s.Cell != "bandwidth" {
		t.Fatalf("Session().Cell = %q, want bandwidth", s.Cell)
	}

	after := sessionModel(t) // the fresh model app.go builds next lap
	after.Restore(s)

	if after.gridNav.cursorRow != row || after.gridNav.cursorCol != col {
		t.Errorf("cursor at %d,%d after restore, want %d,%d",
			after.gridNav.cursorRow, after.gridNav.cursorCol, row, col)
	}
}

// Why a name is carried and not a coordinate: the config is re-read every lap,
// so the cell may have moved by the time you return.
func TestSessionFollowsTheNameNotThePosition(t *testing.T) {
	before := sessionModel(t)
	row, col := at(t, before, "bandwidth")
	before.gridNav.cursorRow, before.gridNav.cursorCol = row, col
	s := before.Session()

	// The profile was edited while the command ran: bandwidth moved.
	moved := settings.Config{X: 3, Y: 2, Commands: []profile.Command{
		{Name: "bandwidth", Command: "echo 4", Row: 0, Col: "a"},
		{Name: "update", Command: "echo 1", Row: 1, Col: "c"},
	}}
	moved.ApplyDefaults()
	after := Model{Config: moved, styles: BuildStyles(moved, ""),
		gridNav: gridNav{grid: profile.BuildGrid(moved.X, moved.Y, moved.Commands)}}
	after.Restore(s)

	wantRow, wantCol := at(t, after, "bandwidth")
	if after.gridNav.cursorRow != wantRow || after.gridNav.cursorCol != wantCol {
		t.Errorf("cursor at %d,%d, want %d,%d — it did not follow the name",
			after.gridNav.cursorRow, after.gridNav.cursorCol, wantRow, wantCol)
	}
	// And the guard against this test proving nothing: the old coordinates must
	// genuinely point somewhere else now.
	if wantRow == row && wantCol == col {
		t.Fatal("the fixture did not move the cell, so this test asserts nothing")
	}
}

// A cell that vanished leaves the cursor at the default rather than out of
// bounds. Same for a profile name that is gone.
func TestSessionSurvivesThingsThatVanished(t *testing.T) {
	after := sessionModel(t)
	after.Restore(Session{Cell: "no-such-cell", Profile: "no-such-profile"})

	if after.gridNav.cursorRow != 0 || after.gridNav.cursorCol != 0 {
		t.Errorf("cursor moved to %d,%d for a cell that does not exist",
			after.gridNav.cursorRow, after.gridNav.cursorCol)
	}
}

// The zero Session is the first lap: it must restore nothing and panic on
// nothing.
func TestZeroSessionRestoresNothing(t *testing.T) {
	m := sessionModel(t)
	row, col := at(t, m, "monitor")
	m.gridNav.cursorRow, m.gridNav.cursorCol = row, col

	m.Restore(Session{})

	if m.gridNav.cursorRow != row || m.gridNav.cursorCol != col {
		t.Error("the zero Session moved the cursor")
	}
}

// An empty grid is reachable (no profiles, rescue paths), so snapshotting one
// must not index out of bounds.
func TestSessionOnAnEmptyGrid(t *testing.T) {
	m := Model{gridNav: gridNav{grid: nil}}
	if got := m.Session(); got.Cell != "" {
		t.Errorf("Session().Cell = %q on an empty grid, want empty", got.Cell)
	}

	m.Restore(Session{Cell: "anything"})
	if m.gridNav.cursorRow != 0 || m.gridNav.cursorCol != 0 {
		t.Error("Restore moved the cursor on an empty grid")
	}
}

// glassroot decides a session must end before the TUI starts. Restore runs
// after that gate in app.go, and must not be able to undo it.
func TestRestoreNeverRevivesAQuittingModel(t *testing.T) {
	m := sessionModel(t)
	m.quitting = true
	m.exitCode = 1

	m.Restore(Session{Cell: "bandwidth", Profile: "core"})

	if !m.quitting {
		t.Error("Restore cleared Quitting — a session glassroot ended could be revived")
	}
	if m.exitCode != 1 {
		t.Errorf("Restore changed ExitCode to %d, want 1", m.exitCode)
	}
}

// The profile is restored through switchToProfileIndex, not by writing
// activeIndex: that function also re-applies the config overlay and calls
// applyConfig, and skipping it would leave the model half-switched.
func TestSessionRestoresTheProfileProperly(t *testing.T) {
	before := switchTestModel(t)

	switched, ok := before.switchToProfileIndex(1)
	if !ok {
		t.Fatal("switch to beta failed")
	}
	s := switched.Session()
	if s.Profile != "beta" {
		t.Fatalf("Session().Profile = %q, want beta", s.Profile)
	}

	after := switchTestModel(t) // fresh, back on alpha
	if after.ActiveProfileName() != "alpha" {
		t.Fatalf("fixture starts on %q, want alpha", after.ActiveProfileName())
	}
	after.Restore(s)

	if got := after.ActiveProfileName(); got != "beta" {
		t.Errorf("ActiveProfileName = %q after restore, want beta", got)
	}
	// The handshake switchToProfileIndex performs. Writing activeIndex alone
	// would leave this empty, which is the half-switched state.
	if after.profile.sessionProfile != "beta" {
		t.Errorf("sessionProfile = %q, want beta — the switch path was bypassed",
			after.profile.sessionProfile)
	}
}

// A profile that is gone leaves the model on whatever it loaded, rather than
// wrapping to some other index.
func TestSessionIgnoresAProfileThatVanished(t *testing.T) {
	m := switchTestModel(t)
	m.Restore(Session{Profile: "gamma"})

	if got := m.ActiveProfileName(); got != "alpha" {
		t.Errorf("ActiveProfileName = %q, want alpha left untouched", got)
	}
}

// The exact value InitialModel returns when glassroot rejects a bundle: no
// grid, no profiles, already done. app.go returns before Restore ever sees it,
// but that ordering is one line in another package, so pin the behaviour here
// too — Restore on this model must be inert in every field.
func TestRestoreIsInertOnAGlassrootRejection(t *testing.T) {
	m := Model{quitting: true, exitCode: 1, GlassrootMode: true}

	m.Restore(Session{Cell: "bandwidth", Profile: "beta"})

	if !m.quitting || m.exitCode != 1 {
		t.Errorf("Quitting=%v ExitCode=%d, want true/1 untouched", m.quitting, m.exitCode)
	}
	if m.gridNav.cursorRow != 0 || m.gridNav.cursorCol != 0 {
		t.Errorf("cursor moved to %d,%d on an empty grid",
			m.gridNav.cursorRow, m.gridNav.cursorCol)
	}
	if m.profile.sessionProfile != "" {
		t.Errorf("sessionProfile = %q — a profile was switched with none loaded",
			m.profile.sessionProfile)
	}
}

// A notice from the host is shown once as a popup, survives a lap it was not
// shown in, and is never shown to a glassroot visitor.
func TestHostNoticesAreShownOnce(t *testing.T) {
	notice := DetailState{Title: "Running without a log", KeyLabel: "Reason", Value: "nope"}

	m := Model{mode: gridMode, notices: []DetailState{notice}}
	m = m.presentNextNotice()
	if m.mode != infoMode || m.activeDetail == nil || m.activeDetail.Title != notice.Title {
		t.Fatalf("notice not shown: mode %v, detail %+v", m.mode, m.activeDetail)
	}
	if len(m.notices) != 0 || len(m.Session().Notices) != 0 {
		t.Error("a shown notice must be forgotten")
	}
	next, _ := m.updateInfoMode(tea.KeyMsg{Type: tea.KeyEnter})
	if got := next.(Model); got.mode != gridMode || got.activeDetail != nil {
		t.Errorf("any key should dismiss it back to the grid, got mode %v", got.mode)
	}

	locked := Model{mode: lockedMode, notices: []DetailState{notice}}
	locked = locked.presentNextNotice()
	if locked.mode != lockedMode || len(locked.Session().Notices) != 1 {
		t.Error("a locked screen keeps its mode; the notice waits and is carried on")
	}
}

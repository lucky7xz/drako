package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

// sessionModel is a 3x2 deck. The cell names are what a Session carries.
func sessionModel(t *testing.T) Model {
	t.Helper()
	cfg := config.Config{X: 3, Y: 2, Commands: []config.Command{
		{Name: "update", Command: "echo 1", Row: 0, Col: "a"},
		{Name: "monitor", Command: "echo 2", Row: 0, Col: "b"},
		{Name: "finder", Command: "echo 3", Row: 0, Col: "c"},
		{Name: "bandwidth", Command: "echo 4", Row: 1, Col: "b"},
	}}
	cfg.ApplyDefaults()
	return Model{
		Config:  cfg,
		styles:  BuildStyles(cfg),
		gridNav: gridNav{grid: config.BuildGrid(cfg)},
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
	moved := config.Config{X: 3, Y: 2, Commands: []config.Command{
		{Name: "bandwidth", Command: "echo 4", Row: 0, Col: "a"},
		{Name: "update", Command: "echo 1", Row: 1, Col: "c"},
	}}
	moved.ApplyDefaults()
	after := Model{Config: moved, styles: BuildStyles(moved),
		gridNav: gridNav{grid: config.BuildGrid(moved)}}
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
	m.Quitting = true
	m.ExitCode = 1

	m.Restore(Session{Cell: "bandwidth", Profile: "core"})

	if !m.Quitting {
		t.Error("Restore cleared Quitting — a session glassroot ended could be revived")
	}
	if m.ExitCode != 1 {
		t.Errorf("Restore changed ExitCode to %d, want 1", m.ExitCode)
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

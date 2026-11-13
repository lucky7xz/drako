package tui

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/lair"
)

// Session is where a run left off, handed to the next one. drako quits the TUI
// so a launched command gets a clean terminal, which means the model that knew
// your position is thrown away and rebuilt -- this is the value that survives
// that gap. It lives for the process and is never written down.
//
// The square first, then the name. InitialModel re-reads the config every lap,
// so by the time you return the deck may have changed: you edited a profile, or
// the command you ran switched one. If the square still holds a cell of that
// name, that is the cell you ran. If not, the cell moved and its name is
// followed. The name alone would not do: two cells may share one.
type Session struct {
	Cell    string
	Row     int // the square Cell stood on
	Col     int
	Profile string
	// Notices are things the host wants the user told once, oldest first:
	// each is shown as a popup and then forgotten.
	Notices      []DetailState
	acknowledged map[string]bool // broken-profile errors already shown
	droppedNoted string          // dropped profile already explained
}

// Session snapshots the position for Restore to re-establish once the config
// has been re-read.
func (m Model) Session() Session {
	s := Session{Profile: m.profile.deckName(), acknowledged: maps.Clone(m.profile.acknowledged), droppedNoted: m.profile.droppedNoted}
	s.Notices = slices.Clone(m.notices) // not shown yet: carried to the next lap

	row, col := m.gridNav.cursorRow, m.gridNav.cursorCol
	if row >= 0 && row < len(m.gridNav.grid) && col >= 0 && col < len(m.gridNav.grid[row]) {
		s.Cell, s.Row, s.Col = m.gridNav.grid[row][col], row, col
	}
	return s
}

// Restore puts the cursor back. The grid has been rebuilt from a config that may
// have changed since, so nothing carried over is trusted: both names are looked
// up afresh and anything missing is simply left alone.
//
// The profile goes first. Switching one replaces the grid, so a cursor set
// before the switch would index into a grid that no longer exists.
//
// Restore never touches quitting or exitCode. InitialModel calls it *after* the
// glassroot gate, so that a session glassroot decided to end cannot be
// revived by a carried value.
//
// A carried profile outranks the pivot lock, which is the one behaviour this
// changes. The pivot answers "which profile does a fresh process start on"
// (config.resolveRequested: pivot > DRAKO_PROFILE > config.toml), and
// it still does; a lap of the launch loop is not a fresh start, and an explicit
// mid-session switch is the more recent statement of intent. Nothing is granted
// by this: the pivot pins a starting point, it is not an access control, and
// glassroot already contracts that a session can reach every equipped profile.
func (m *Model) Restore(s Session) {
	// deckName, not ActiveProfileName: a rescue load already rejected that deck.
	if s.Profile != "" && s.Profile != m.profile.deckName() {
		for i, p := range m.profile.profiles {
			if p.Name != s.Profile {
				continue
			}
			// switchToProfileIndex is the real path: it re-applies the config
			// overlay and calls applyConfig. Setting activeIndex alone would
			// leave the model half-switched.
			if updated, ok := m.switchToProfileIndex(i); ok {
				*m = updated
			}
			break
		}
	}

	if s.Cell == "" {
		return
	}
	// The same square first: a second cell of that name elsewhere is another cell.
	if g := m.gridNav.grid; s.Row >= 0 && s.Row < len(g) && s.Col >= 0 && s.Col < len(g[s.Row]) && g[s.Row][s.Col] == s.Cell {
		m.gridNav.cursorRow, m.gridNav.cursorCol = s.Row, s.Col
		return
	}
	for row, cells := range m.gridNav.grid {
		for col, name := range cells {
			if name == s.Cell {
				m.gridNav.cursorRow, m.gridNav.cursorCol = row, col
				return
			}
		}
	}
}

// InitialModel builds one lap's model from the config on disk, then applies what
// the previous lap carried over (the zero Session on the first lap).
func InitialModel(setup Setup, carried Session) Model {
	glassrootMode := setup.Glassroot
	path, err := host.WorkDir()
	if err != nil {
		path = "could not get path"
	}

	bundle, err := lair.LoadConfig(setup.ConfigDir, setup.Targets, setup.Profile)
	if err != nil {
		// No config directory means nothing to run from; hand the host a
		// model that is already done and let app.Run exit with the code.
		return Model{quitting: true, exitCode: 1, exitMessage: fmt.Sprintf("drako: %v", err), GlassrootMode: glassrootMode}
	}

	if glassrootMode && glassrootRejectsBundle(bundle) {
		// Glassroot never exposes TOML contents or Rescue Mode. Hand the host
		// a model that is already done; app.Run exits before starting the TUI.
		return Model{quitting: true, exitCode: 1, GlassrootMode: true}
	}

	s := newSpinner()
	m := Model{
		net: netStatus{
			meter:   host.TrafficMeter{WindowSeconds: 7.5},
			online:  "checking...",
			traffic: "calculating...",
		},
		path:    InitPathModel(path),
		mode:    gridMode,
		spinner: s,
		profile: profileState{
			base:         bundle.Base,
			acknowledged: make(map[string]bool),
		},
		lock: lockState{
			lastActivity: time.Now(),
			modeBefore:   gridMode,
			pumpGoal:     defaultLockPumpGoal,
		},
		GlassrootMode: glassrootMode,
		setup:         setup,
	}
	m.applyBundle(bundle)
	for path := range carried.acknowledged {
		m.profile.acknowledged[path] = true
	}
	m.pruneAcknowledged(bundle.Broken) // an error dismissed last lap stays dismissed while still broken
	if bundle.DroppedProfile == carried.droppedNoted {
		m.profile.droppedNoted = carried.droppedNoted
	}
	m.Restore(carried)
	if !glassrootMode { // a glassroot visitor is told nothing about the host
		m.notices = slices.Clone(carried.Notices)
	}

	if len(bundle.Broken) > 0 {
		m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
		m.profile.errorQueueActive = true
		if bundle.DroppedProfile != "" && bundle.DroppedProfile != m.profile.droppedNoted {
			m.profile.pendingDroppedProfile = bundle.DroppedProfile // shown once the queue is dismissed
		}
		m = m.presentNextBrokenProfile()
	} else if bundle.DroppedProfile != "" && bundle.DroppedProfile != m.profile.droppedNoted {
		// Relaunched after the active profile was deleted out from under us:
		// explain the rescue drop instead of showing a bare rescue grid.
		m = m.presentDroppedProfileNote(bundle.DroppedProfile)
	} else {
		m = m.presentNextNotice()
	}

	return m
}

package tui

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/settings"
)

// enterRescue swaps in the rescue grid without forgetting which deck it replaced.
func (m *Model) enterRescue() {
	cfg := settings.RescueConfig(m.setup.ConfigDir)
	cfg.ApplyDefaults()
	m.applyConfig(cfg)
	m.profile.rescue = true
}

func (m *Model) applyBundle(bundle lair.ConfigBundle) {
	m.profile.base = bundle.Base
	profiles := bundle.Profiles
	m.profile.profiles = profiles
	if bundle.ActiveIndex < 0 || bundle.ActiveIndex >= len(profiles) {
		m.profile.activeIndex = 0
	} else {
		m.profile.activeIndex = bundle.ActiveIndex
	}
	m.applyConfig(bundle.Config)
	m.profile.rescue = bundle.Rescue
	m.batch = batchState{} // marks name cells of the grid being replaced
	if m.showsGridCells(m.lock.modeBefore) {
		m.lock.modeBefore = gridMode // and so would the mode unlocking returns to
	}
	m.profile.configDir = bundle.ConfigDir
	m.profile.pivotName = bundle.LockedName
	m.profile.locked = strings.TrimSpace(bundle.LockedName) != ""
	if bundle.DroppedProfile == m.profile.sessionProfile {
		m.profile.sessionProfile = "" // the deck switched to is gone; stop requesting it
	}
	if bundle.DroppedProfile != m.profile.droppedNoted {
		m.profile.droppedNoted = "" // resolved, or a different drop: that one is news
	}
	m.pruneAcknowledged(bundle.Broken)
}

// pruneAcknowledged forgets an acknowledgment once its file is no longer
// reported broken, so a later re-break is shown again instead of being
// mistaken for the same still-open error.
func (m *Model) pruneAcknowledged(broken []lair.ProfileParseError) {
	stillBroken := make(map[string]bool, len(broken))
	for _, e := range broken {
		stillBroken[e.Path] = true
	}
	for path := range m.profile.acknowledged {
		if !stillBroken[path] {
			delete(m.profile.acknowledged, path)
		}
	}
}

// notShown is the part of broken whose notice hasn't been shown yet.
func (p profileState) notShown(broken []lair.ProfileParseError) []lair.ProfileParseError {
	var rest []lair.ProfileParseError
	for _, e := range broken {
		if !p.acknowledged[e.Path] {
			rest = append(rest, e)
		}
	}
	return rest
}

// applyReloadedBundle applies a reload and shows any broken/dropped-profile
// notice. presentNextBrokenProfile/presentDroppedProfileNote defer instead
// of presenting while protectsMode() is true.
func (m Model) applyReloadedBundle(bundle lair.ConfigBundle) Model {
	m.applyBundle(bundle)
	if len(bundle.Broken) > 0 {
		if bundle.DroppedProfile != "" && bundle.DroppedProfile != m.profile.droppedNoted {
			m.profile.pendingDroppedProfile = bundle.DroppedProfile // shown once the queue is dismissed
		}
		if m.showingQueue() {
			// A notice waits for its key: a reload racing it (often drako's own
			// pivot.toml write) only updates what is queued behind it.
			m.profile.pendingErrors = m.profile.notShown(bundle.Broken)
			return m
		}
		m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
		m.profile.errorQueueActive = true
		return m.presentNextBrokenProfile()
	}
	// Nothing is broken any more, so a broken-file notice has lost its subject,
	// as a cell's popup has when the grid is replaced. Any other note stays.
	outdated := m.showingQueue() || m.showsGridCells(m.mode)
	m.profile.pendingErrors, m.profile.errorQueueActive = nil, false
	if bundle.DroppedProfile != "" && bundle.DroppedProfile != m.profile.droppedNoted {
		return m.presentDroppedProfileNote(bundle.DroppedProfile)
	}
	if outdated {
		m.mode = gridMode
	}
	return m
}

// reload loads the configuration again for the running session: the profile
// the session has active takes the place of the one asked for from outside.
func (m Model) reload() (lair.ConfigBundle, error) {
	wanted := m.profile.sessionProfile
	if wanted == "" {
		wanted = m.setup.Profile
	}
	return lair.LoadConfig(m.setup.ConfigDir, m.setup.Targets, wanted)
}

// reloadAndApply loads the configuration again and applies it to the running
// session. applied is false when there is nothing new on screen: the load
// failed (the status line says so), or glassroot refuses what it found (the
// model that comes back ends the session).
func (m Model) reloadAndApply() (next Model, cmd tea.Cmd, applied bool) {
	bundle, err := m.reload()
	if err != nil {
		log.Printf("config reload failed: %v", err)
		return m, m.setProfileStatus("Config reload failed", false), false
	}
	if m.GlassrootMode && glassrootRejectsBundle(bundle) {
		next, cmd = m.failGlassroot()
		return next, cmd, false
	}
	return m.applyReloadedBundle(bundle), nil, true
}

package tui

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/settings"
)

// activeName returns the display name of the active profile, defaulting to
// "Rescue" when there are no profiles or the name is blank (no profiles on
// disk means the compiled-in rescue grid is what's showing).
func (p profileState) activeName() string {
	if len(p.profiles) == 0 || p.rescue {
		return "Rescue"
	}
	idx := p.activeIndex
	if idx < 0 || idx >= len(p.profiles) {
		idx = 0
	}
	name := p.profiles[idx].Name
	if strings.TrimSpace(name) == "" {
		return "Rescue"
	}
	return name
}

// ActiveProfileName is the name of the currently active profile, or "" when
// no profiles are loaded. Spawned commands see it as DRAKO_PROFILE.
func (m Model) ActiveProfileName() string {
	if m.profile.rescue {
		return ""
	}
	return m.profile.deckName()
}

// deckName is the active deck's name, or the one rescue goes back to.
func (p profileState) deckName() string {
	if len(p.profiles) == 0 || p.activeIndex < 0 || p.activeIndex >= len(p.profiles) {
		return ""
	}
	return p.profiles[p.activeIndex].Name
}

func (m *Model) scheduleStatusClearTimer() tea.Cmd {
	m.profile.nextTimerID++
	id := m.profile.nextTimerID
	m.profile.statusClearTimerID = id
	return statusClearTick(id)
}

// statusClearTick clears the status line set under timer id, once it has
// been up long enough.
func statusClearTick(id int) tea.Cmd {
	return tea.Tick(profileStatusDuration, func(time.Time) tea.Msg {
		return profileStatusClearMsg{id: id}
	})
}

func (m *Model) setProfileStatus(message string, positive bool) tea.Cmd {
	m.profile.statusMessage = message
	m.profile.statusPositive = positive
	if strings.TrimSpace(message) == "" {
		m.profile.statusClearTimerID = 0
		return nil
	}
	return m.scheduleStatusClearTimer()
}

func (m *Model) toggleProfileLock() tea.Cmd {
	if strings.TrimSpace(m.profile.configDir) == "" {
		return m.setProfileStatus("Pivot unavailable", false)
	}

	currentName := m.ActiveProfileName()
	if currentName == "" {
		return m.setProfileStatus("Nothing to pin in Rescue Mode", false)
	}
	normCurrent := equip.NormalizeName(currentName)
	normPivot := equip.NormalizeName(m.profile.pivotName)

	var err error
	var messageCmd tea.Cmd

	if m.profile.locked && normPivot == normCurrent && m.profile.pivotName != "" {
		err = lair.DeletePivotProfile(m.profile.configDir)
		if err == nil {
			m.profile.locked = false
			m.profile.pivotName = ""
			messageCmd = m.setProfileStatus("Pivot cleared", false)
		}
	} else {
		err = lair.WritePivotLocked(m.profile.configDir, currentName)
		if err == nil {
			m.profile.locked = true
			m.profile.pivotName = currentName
			messageCmd = m.setProfileStatus(fmt.Sprintf("Pinned %s", currentName), true)
		}
	}

	if err != nil {
		log.Printf("pivot update failed: %v", err)
		return m.setProfileStatus("Pivot error", false)
	}
	return messageCmd
}

func (m Model) switchToProfileIndex(target int) (Model, bool) {
	if target < 0 || target >= len(m.profile.profiles) {
		return m, false
	}

	selected := m.profile.profiles[target]

	// Check for existence
	if !lair.Exists(filepath.Dir(selected.Path), filepath.Base(selected.Path)) {
		log.Printf("skipping missing profile: %s", selected.Path)
		return m, false
	}

	updated := m
	updated.profile.activeIndex = target
	updated.profile.rescue = false
	updated.profile.sessionProfile = selected.Name
	updated.Config = settings.ApplyProfileOverlay(m.profile.base, selected.Profile)
	updated.applyConfig(updated.Config)

	return updated, true
}

func (m Model) handleProfileCycle(direction int) (tea.Model, tea.Cmd) {
	if len(m.profile.profiles) <= 1 {
		return m, nil
	}

	current := m.profile.activeIndex
	total := len(m.profile.profiles)
	// Only try up to 'total' times.
	// Start from 1 to avoid re-selecting the current profile immediately.
	for i := 1; i <= total; i++ {
		target := calculateNextProfileIndex(current, direction*i, total)

		nextModel, ok := m.switchToProfileIndex(target)
		if ok {
			return nextModel, nil
		}
	}
	return m, nil
}

// calculateNextProfileIndex determines the next profile index based on direction and wrapping.
// direction should be 1 (next) or -1 (previous).
func calculateNextProfileIndex(currentIndex, direction, totalProfiles int) int {
	if totalProfiles <= 0 {
		return 0
	}
	next := (currentIndex + direction) % totalProfiles
	if next < 0 {
		next += totalProfiles
	}
	return next
}

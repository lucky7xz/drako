package tui

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Directions for the lock pump mechanism
const (
	directionNone  = 0
	directionLeft  = -1
	directionRight = 1
)

// pumpDirectionForKey returns the direction associated with a given key press.
// Returns directionLeft for 'left', 'h', 'a'.
// Returns directionRight for 'right', 'l', 'd'.
// Returns directionNone (0) for any other key.
func pumpDirectionForKey(key string) int {
	switch key {
	case "left", "h", "a":
		return directionLeft
	case "right", "l", "d":
		return directionRight
	default:
		return directionNone
	}
}

func (m Model) enterLockedMode() Model {
	if m.mode != lockedMode {
		m.lock.modeBefore = m.mode
	}
	m.mode = lockedMode
	m.lock.progress = 0
	m.lock.lastDirection = 0
	m.sheet = nil // nobody is reading it any more; unlocking must not spend a key on it
	return m
}

// exitLockedMode restores the mode locked from. Anything that piled up
// while locked (pendingErrors) surfaces afterward via
// presentDeferredIfSafe, run by Update() after every message.
func (m Model) exitLockedMode() Model {
	if m.mode == lockedMode {
		m.mode = m.lock.modeBefore
	}
	m.lock.lastActivity = time.Now()
	m.lock.progress = 0
	m.lock.lastDirection = 0
	return m
}

// pump advances the unlock slider for one keypress: alternating directions
// fill it, repeating a direction drains it. Reports whether the goal was hit.
func (l *lockState) pump(dir int) bool {
	if l.lastDirection == dir {
		if l.progress > 0 {
			l.progress--
		}
		return false
	}
	l.lastDirection = dir
	if l.progress < l.pumpGoal {
		l.progress++
	}
	return l.progress >= l.pumpGoal
}

func (m Model) updateLockedMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Allow Ctrl+C to quit even when locked
	if key == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	dir := pumpDirectionForKey(key)
	if dir == 0 {
		return m, nil
	}

	if m.lock.pump(dir) {
		log.Printf("Unlocking via pump sequence after %d steps", m.lock.progress)
		m = m.exitLockedMode()
	}
	return m, nil
}

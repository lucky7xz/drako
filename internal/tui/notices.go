package tui

import (
	"fmt"
	"strings"

	"github.com/lucky7xz/drako/internal/glyph"
	"github.com/lucky7xz/drako/internal/lair"
)

// showingQueue: a broken-profile notice is on screen, waiting for its key.
func (m Model) showingQueue() bool {
	return m.profile.errorQueueActive && m.mode == infoMode
}

// presentDeferredIfSafe shows the oldest queued notice after every message;
// a queue already on screen waits for the user's keypress (updateInfoMode).
func (m Model) presentDeferredIfSafe() Model {
	if m.showingQueue() {
		return m
	}
	if len(m.profile.pendingErrors) > 0 {
		return m.presentNextBrokenProfile()
	}
	if m.profile.pendingDroppedProfile != "" {
		name := m.profile.pendingDroppedProfile
		m.profile.pendingDroppedProfile = ""
		return m.presentDroppedProfileNote(name)
	}
	if m.mode != infoMode { // a popup already up keeps the screen
		return m.presentNextNotice()
	}
	return m
}

// presentNextNotice shows the oldest notice from the host as a popup, once.
// With none left, or while the mode is protected, it changes nothing.
func (m Model) presentNextNotice() Model {
	if len(m.notices) == 0 || m.protectsMode() {
		return m
	}
	notice := m.notices[0]
	m.notices = m.notices[1:]
	m.previousMode = m.mode
	m.profile.errorQueueActive = false
	m.activeDetail = &notice
	m.mode = infoMode
	return m
}

// presentNextBrokenProfile pops the next pending broken profile error and configures infoMode to display it.
// Deferred (queue left untouched) while protectsMode() is true.
func (m Model) presentNextBrokenProfile() Model {
	if m.protectsMode() {
		return m
	}
	// Filter out already acknowledged errors
	staleOnly := 0
	for len(m.profile.pendingErrors) > 0 {
		e := m.profile.pendingErrors[0]
		if m.profile.acknowledged[e.Path] { // Track by Path to be specific
			m.profile.pendingErrors = m.profile.pendingErrors[1:]
			staleOnly++
			continue
		}
		break
	}

	if len(m.profile.pendingErrors) == 0 {
		// Queue exhausted.
		// staleOnly counts errors dropped because the user has already seen
		// them: a queue made only of those showed nothing, so there is nothing
		// to drop into rescue *from*. That is the Exit Rescue Mode path — the
		// reload still reports the broken config.toml it was acknowledged for,
		// and re-entering here would discard the config the caller just applied
		// and trap the user in rescue.
		if m.profile.errorQueueActive && staleOnly == 0 {
			m.enterRescue() // we just finished processing a queue
		}

		m.profile.errorQueueActive = false
		if staleOnly == 0 {
			// Only reset here when the queue is genuinely done. If it only
			// emptied by discarding already-shown duplicates (a second reload
			// racing the one that just presented them), leave the screen
			// alone instead of tearing down what that call just put up.
			m.mode = gridMode
			m.activeDetail = nil
		}
		return m
	}

	e := m.profile.pendingErrors[0]
	m.profile.pendingErrors = m.profile.pendingErrors[1:]

	// Mark as acknowledged
	m.profile.acknowledged[e.Path] = true

	// Capture previous mode only if we are transitioning FROM a valid mode.
	// If we are already in infoMode (chained errors), we keep the original previousMode.
	// However, since we now force return to Grid Mode at the end, this is less critical
	// but good for hygiene if we ever change the exit logic.
	if m.mode != infoMode {
		m.previousMode = m.mode
	}

	desc := "This profile has an error and was hidden from selection.\n\n"
	if e.Name == lair.ConfigFileName {
		// Specific message for the main config
		desc = "If your config.toml is invalid, Rescue Mode can be helpful.\n" +
			"Warning: Default keybindings are in effect. Your custom keys may not work (since the config.toml which stores them is invalid).\n\n" +
			"Available Actions:\n" +
			"• Edit Config: Open the config.toml in your default editor and fix the syntax error.\n" +
			"• Reset Config: The command `drako purge --config` moves the config.toml to trash/. Drako writes a new one with default settings.\n" +
			"• Documentation: View default controls on Documentation website.\n" +
			"• Exit Rescue Mode: You can still keep using drako by exiting rescue mode (the top-left button, the error will keep showing up though).\n\n"

	} else if strings.HasPrefix(e.Err, "file name ") {
		desc += "drako does not use a file whose name it cannot draw exactly as written: you would have to type a name you cannot see. Rename the file.\n\n"
	} else if strings.Contains(e.Err, "empty profile file") {
		desc += "The file is completely empty. Either add valid TOML configuration or move/delete the file via Inventory (i).\n\n"
	} else if strings.Contains(e.Err, "no settings found") {
		desc += "The file exists but contains no configuration settings. Either add valid TOML configuration or move/delete the file via Inventory (i).\n\n"
	} else {
		desc += "The file has a TOML syntax error. Either fix the syntax error or move/delete the file via Inventory (i).\n\n"
	}
	desc += "Press any key to continue to the next error, or 'y' to copy error details to clipboard."

	m.activeDetail = &DetailState{
		Title:       fmt.Sprintf("%s error: %s", errorSubject(e.Name), glyph.Clean(e.Name)),
		KeyLabel:    "Error",
		Value:       fmt.Sprintf("Path: %s\nError: %s", glyph.Clean(e.Path), glyph.Clean(strings.TrimSpace(e.Err))),
		Description: desc,
		Meta: []DetailMeta{
			{Label: "CWD", Value: m.profile.configDir},
		},
	}
	m.mode = infoMode
	return m
}

// presentDroppedProfileNote shows a one-time overlay explaining that the active
// profile was deleted or moved to the inventory, which is why drako just dropped
// into Rescue Mode. Without it the rescue drop is silent (this path carries no
// broken-profile error). Dismissed by any key, returning to the grid (the rescue
// grid).
func (m Model) presentDroppedProfileNote(name string) Model {
	if m.protectsMode() {
		m.profile.pendingDroppedProfile = name
		return m
	}
	m.previousMode = gridMode
	m.profile.errorQueueActive = false
	m.profile.droppedNoted = name
	desc := "This profile is no longer available — it was deleted or moved to " +
		"the inventory. drako has dropped into Rescue Mode (any lock on it has been " +
		"cleared).\n\n" +
		"To get back to work:\n" +
		"• Inventory (i): equip a profile.\n" +
		"• Cycle (o / p): jump to a still-equipped profile.\n" +
		"• Exit Rescue Mode: the top-left button of the grid.\n\n" +
		"Press any key to dismiss."
	if active := m.ActiveProfileName(); active != "" { // none in Rescue
		desc = "This profile is no longer available — it was deleted or moved to " +
			"the inventory. drako switched to " + active + " (any lock " +
			"on it has been cleared).\n\n" +
			"Inventory (i) can equip it again.\n\n" +
			"Press any key to dismiss."
	}
	m.activeDetail = &DetailState{
		Title:       "Profile no longer available",
		KeyLabel:    "Profile",
		Value:       name,
		Description: desc,
		Meta: []DetailMeta{
			{Label: "CWD", Value: m.profile.configDir},
		},
	}
	m.mode = infoMode
	return m
}

// errorSubject names what a broken file is: drako's settings, or a profile.
func errorSubject(name string) string {
	if name == lair.ConfigFileName {
		return "Config"
	}
	return "Profile"
}

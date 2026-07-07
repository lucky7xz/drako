package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucky7xz/drako/internal/config"
)

// Glassroot is drako's kiosk lockdown (--glassroot): the session may only see
// and run what its decks offer. This file is the single policy point — every
// glassroot restriction is defined here, so "what does glassroot block?" has
// exactly one answer and one test.
//
// Blocked: entering inventory, entering path mode, toggling the pivot lock,
// copying to the clipboard, and exposing config/rescue details when a profile
// is broken or no profiles are equipped at all (the session ends silently
// instead — see failGlassroot and glassrootRejectsBundle).
// File editing is blocked transitively (it lives inside inventory mode) plus
// a defense-in-depth check at the handler — an editor is a shell (:!sh).
//
// Deliberately allowed (decided 2026-07-03): profile switching/cycling and the
// explain popup. Operator contract: every profile equipped on a glassroot box
// must be kiosk-safe, because a session can reach all of them.

// glassrootBlocksKey reports whether glassroot suppresses this keypress.
func (m Model) glassrootBlocksKey(msg tea.KeyMsg) bool {
	if !m.GlassrootMode {
		return false
	}
	return IsLock(m.Config.Keys, msg) ||
		IsInventory(m.Config.Keys, msg) ||
		IsPathGridMode(m.Config.Keys, msg)
}

// allowCopy reports whether clipboard copy is permitted. Copy execs
// server-side clipboard tools (wl-copy/xclip/…) and can expose broken-profile
// TOML contents, so glassroot forbids it.
func (m Model) allowCopy() bool {
	return !m.GlassrootMode
}

// glassrootRejectsBundle reports whether a freshly loaded bundle must not be
// shown in a glassroot session: broken profiles expose TOML details, and zero
// equipped profiles would land in the rescue grid (config-editing cells).
func glassrootRejectsBundle(b config.ConfigBundle) bool {
	return len(b.Broken) > 0 || len(b.Profiles) == 0
}

// failGlassroot ends the session silently with exit code 1. Glassroot never
// shows rescue mode or config internals on a broken profile; it quits through
// the normal bubbletea shutdown so the terminal is restored and the host can
// read Quitting/ExitCode off the final model.
func (m Model) failGlassroot() (Model, tea.Cmd) {
	m.Quitting = true
	m.ExitCode = 1
	return m, tea.Quit
}

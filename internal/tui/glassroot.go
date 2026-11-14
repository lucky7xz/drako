package tui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/settings"
)

// Glassroot is drako's kiosk lockdown (--glassroot): the session may only see
// and run what its decks offer. This file is the single policy point — every
// glassroot restriction is defined here, so "what does glassroot block?" has
// exactly one answer and one test.
//
// Blocked: entering inventory, entering path mode, toggling the pivot lock,
// copying to the clipboard, starting a batch, and exposing config/rescue
// details when a profile is broken or no profiles are equipped at all (the
// session ends silently instead — see failGlassroot and glassrootRejectsBundle).
// The inventory's spec-function picker needs no rule of its own: it lives
// inside inventory mode, and its key is the path-mode key blocked here.
// File editing is blocked transitively (it lives inside inventory mode) plus
// a defense-in-depth check at the handler — an editor is a shell (:!sh).
//
// Deliberately allowed (decided 2026-07-03): profile switching/cycling and the
// explain popup. Operator contract: every profile equipped on a glassroot box
// must be kiosk-safe, because a session can reach all of them.
//
// Which keys those are is marked in the key table (Gated, settings/keymap.go);
// the key sheet leaves the same ones out.

// glassrootBlocksKey reports whether glassroot suppresses this keypress.
func (m Model) glassrootBlocksKey(msg tea.KeyMsg) bool {
	if !m.GlassrootMode {
		return false
	}
	// The grid's gated keys, whichever screen the guest is on: a key that
	// leaves the kiosk does nothing anywhere.
	for _, b := range m.Config.Screen(settings.Grid).Binds {
		if b.Gated && slices.Contains(b.Keys, msg.String()) {
			return true
		}
	}
	return false
}

// allowCopy reports whether clipboard copy is permitted. Copy execs
// server-side clipboard tools (wl-copy/xclip/…) and can expose broken-profile
// TOML contents, so glassroot forbids it.
func (m Model) allowCopy() bool {
	return !m.GlassrootMode
}

// allowBatch reports whether a batch may start. A batch hands the guest a full
// tmux or herdr session — new windows, shells — far outside the kiosk contract,
// so glassroot forbids it. batchGate asks this for every batch entry point.
func (m Model) allowBatch() bool {
	return !m.GlassrootMode
}

// glassrootRejectsBundle reports whether a freshly loaded bundle must not be
// shown in a glassroot session: broken profiles expose TOML details, and zero
// equipped profiles would land in the rescue grid (config-editing cells).
func glassrootRejectsBundle(b lair.ConfigBundle) bool {
	return len(b.Broken) > 0 || len(b.Profiles) == 0
}

// failGlassroot ends the session silently with exit code 1. Glassroot never
// shows rescue mode or config internals on a broken profile; it quits through
// the normal bubbletea shutdown so the terminal is restored and the host can
// read it from the final model's Outcome.
func (m Model) failGlassroot() (Model, tea.Cmd) {
	m.quitting = true
	m.exitCode = 1
	return m, tea.Quit
}

package ui

import (
	"fmt"

	"github.com/lucky7xz/drako/internal/config"
)

// newCommandDetail builds the info popup for a command. value arrives resolved,
// so callers keep their own empty-command text; an empty address is left out.
func newCommandDetail(title, address, value, description string, autoCloseExec *bool, cwd string) *DetailState {
	autoClose := autoCloseExec == nil || *autoCloseExec // unset means close
	var meta []DetailMeta
	if address != "" {
		meta = append(meta, DetailMeta{Label: "Address", Value: address})
	}
	meta = append(meta,
		DetailMeta{Label: "Auto-close", Value: fmt.Sprintf("%v", autoClose)},
		DetailMeta{Label: "CWD", Value: cwd},
	)
	return &DetailState{
		Title:       title,
		KeyLabel:    "Command",
		Value:       value,
		Description: description,
		Meta:        meta,
	}
}

// runAddress is what drako run takes for the named cell, or its item n if n > 0
// ("core:A0"); "" when that wouldn't reach the cell on screen, as in rescue.
func (m Model) runAddress(name string, n int) string {
	if m.profile.activeIndex < 0 || m.profile.activeIndex >= len(m.profile.profiles) {
		return ""
	}
	p := m.profile.profiles[m.profile.activeIndex]
	for _, shown := range m.Config.Commands {
		if shown.Name != name {
			continue
		}
		own := config.FindCell(p.Profile.Commands, shown.Address())
		if own == nil || own.Name != shown.Name || own.Command != shown.Command {
			return ""
		}
		if n > 0 {
			return p.Name + ":" + own.ItemAddress(n)
		}
		return p.Name + ":" + own.Address()
	}
	return ""
}

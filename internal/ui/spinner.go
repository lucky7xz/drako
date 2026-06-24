package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerTickMsg advances the spinner one frame. A single spinner runs at a
// time, so each tick schedules exactly one successor — no id/tag guard needed.
type spinnerTickMsg struct{}

// spinnerModel is the small header spinner. It replaces bubbles/spinner, whose
// only use here was the line animation below.
type spinnerModel struct {
	frames []string
	frame  int
	fps    time.Duration
	style  lipgloss.Style
}

func newSpinner() spinnerModel {
	return spinnerModel{frames: []string{"|", "/", "-", "\\"}, fps: time.Second / 10}
}

func (s spinnerModel) tick() tea.Cmd {
	return tea.Tick(s.fps, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

func (s spinnerModel) Update() (spinnerModel, tea.Cmd) {
	s.frame = (s.frame + 1) % len(s.frames)
	return s, s.tick()
}

func (s spinnerModel) View() string {
	if s.frame >= len(s.frames) {
		return ""
	}
	return s.style.Render(s.frames[s.frame])
}

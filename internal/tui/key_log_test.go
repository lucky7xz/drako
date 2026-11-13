package tui

import (
	"bytes"
	"log"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// What is typed or pasted is not written to the log: a pasted password
// would land in drako.log.
func TestKeysAreNotLogged(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	m := createTestGridModel()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hunter2"), Paste: true})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if got := buf.String(); strings.Contains(got, "hunter2") || strings.Contains(got, "Key pressed") {
		t.Fatalf("typed text reached the log:\n%s", got)
	}
}

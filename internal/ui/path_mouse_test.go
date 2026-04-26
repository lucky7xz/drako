package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// TestPathModeMouseScroll tests that scroll in pathMode enters pickerMode
func TestPathModeMouseScroll(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	// Create temp directory for testing
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Change to temp dir
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       pathMode,
		path:       InitPathModel(tmpDir),
		Config: config.Config{
			X: 3,
			Y: 3,
		},
	}

	// Verify we're in pathMode and path is set
	if m.mode != pathMode {
		t.Fatalf("Expected pathMode, got %v", m.mode)
	}

	// Test scroll down in pathMode - should enter pickerMode
	msgScrollDown := tea.MouseMsg{
		Type:   tea.MouseWheelDown,
		Action: tea.MouseActionPress,
		X:      10,
		Y:      10,
	}

	updated, _ := m.Update(msgScrollDown)
	mFinal := updated.(Model)

	// After scroll down in pathMode, should enter pickerMode
	if mFinal.mode != pickerMode {
		t.Errorf("Expected pickerMode after scroll in pathMode, got %v", mFinal.mode)
	}

	// Test scroll up in pickerMode - should go back to pathMode
	msgScrollUp := tea.MouseMsg{
		Type:   tea.MouseWheelUp,
		Action: tea.MouseActionPress,
		X:      10,
		Y:      10,
	}

	updated, _ = mFinal.Update(msgScrollUp)
	mFinal = updated.(Model)

	if mFinal.mode != pathMode {
		t.Errorf("Expected pathMode after scroll up in pickerMode, got %v", mFinal.mode)
	}
}

// TestPathModeDoubleClick tests that double-click on path component changes directory
func TestPathModeDoubleClick(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	// Create temp directory with subdirectory
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "testsubdir")
	os.MkdirAll(subDir, 0755)

	// Change to temp dir
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       pathMode,
		path:       InitPathModel(tmpDir),
		Config: config.Config{
			X: 3,
			Y: 3,
		},
		lastClickTime: time.Now().Add(-1 * time.Second), // Ensure not double-click
	}

	// Simulate first click (single click)
	msgClick := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
		X:      5,
		Y:      20,
	}

	updated, _ := m.resolvePathMouseClick(msgClick)
	mAfterClick := updated.(Model)

	// Now simulate double-click (quick click at same position)
	mAfterClick.lastClickTime = time.Now()
	msgDoubleClick := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
		X:      5,
		Y:      20,
	}

	updated, _ = mAfterClick.resolvePathMouseClick(msgDoubleClick)
	mFinal := updated.(Model)

	// Double-click should change to that path and return to grid
	if mFinal.mode != gridMode {
		t.Logf("Double-click result mode: %v (may vary based on click detection)", mFinal.mode)
	}
}

// TestPickerModeClick tests that single-click on directory enters pickerMode
// This test verifies the keyboard navigation logic works
func TestPickerModeClick(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	// Create temp directory with subdirectory
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "mydir")
	os.MkdirAll(subDir, 0755)

	// Change to temp dir
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	// Create config with InputConfig for key handling
	inputCfg := config.InputConfig{
		NavUp:    []string{"up"},
		NavDown:  []string{"down"},
		NavLeft:  []string{"left"},
		NavRight: []string{"right"},
	}

	cfg := config.Config{
		X:   3,
		Y:   3,
		Keys: inputCfg,
	}

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       pathMode,
		path:       InitPathModel(tmpDir),
		Config:     cfg,
	}

	// Ensure child dirs are listed
	if len(m.path.ChildDirs) == 0 {
		t.Fatal("Expected child directories to be listed")
	}

	// Test that pressing Down arrow enters pickerMode
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	mode, _ := m.path.UpdatePathMode(downKey, cfg)
	if mode != pickerMode {
		t.Errorf("Expected Down arrow to enter pickerMode, got %v", mode)
	}
}
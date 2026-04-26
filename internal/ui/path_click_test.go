package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

// TestPathBarClickPosition tests that clicking on the path bar at the correct Y position works
func TestPathBarClickPosition(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	// Create temp directory for testing
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Change to temp dir
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	// Simulate typical terminal size
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

	// Mimic the position calculation from resolvePathMouseClick (backward from footer end)
	headerContent := renderDefaultHeaderArt(m.spinner.View())
	counterContent := m.renderProfileCounter()
	buttonsContent := m.renderProfileButtons()
	gridContent := m.renderGrid()
	pathBarContent := m.path.RenderPathBar(true)
	childDirsContent := m.path.RenderChildDirs(m.mode)

	helpText := "Path Mode | ←/→/ad: Select, ↑/s: Children, Enter: cd, e: Search, q/Esc: Back"
	help := helpStyle.Render(helpText)
	footerContent := m.renderCombinedFooter(help)

	hHeight := lipgloss.Height(headerContent)
	cHeight := lipgloss.Height(counterContent)
	bHeight := lipgloss.Height(buttonsContent)
	gHeight := lipgloss.Height(gridContent)
	pathBarH := lipgloss.Height(pathBarContent)
	childDirsH := lipgloss.Height(childDirsContent)
	fHeight := lipgloss.Height(footerContent)

	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	yOffset := (m.termHeight - totalHeight) / 2
	footerY := yOffset + hHeight + cHeight + bHeight + gHeight
	childDirsY := footerY + fHeight - childDirsH
	pathBarY := childDirsY - pathBarH

	t.Logf("Calculated: termH=%d, totalH=%d, yOffset=%d, footerY=%d, pathBarY=%d",
		m.termHeight, totalHeight, yOffset, footerY, pathBarY)

	// Test clicking at the calculated pathBarY position
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
		X:      5,  // Click on first path component
		Y:      pathBarY,
	}

	updated, _ := m.resolvePathMouseClick(msg)
	mFinal := updated.(Model)

	// Single click should select the path component (stay in pathMode)
	if mFinal.mode != pathMode {
		t.Errorf("Expected pathMode after single click on path bar, got %v", mFinal.mode)
	}

	// Should have selected first component (index 0)
	if mFinal.path.SelectedPathIndex != 0 {
		t.Errorf("Expected SelectedPathIndex=0, got %d", mFinal.path.SelectedPathIndex)
	}
}

// TestPickerModeDirectoryClick tests clicking on a directory in pickerMode
func TestPickerModeDirectoryClick(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	tmpDir := t.TempDir()
	subDir1 := filepath.Join(tmpDir, "dir1")
	subDir2 := filepath.Join(tmpDir, "dir2")
	os.MkdirAll(subDir1, 0755)
	os.MkdirAll(subDir2, 0755)

	os.Chdir(tmpDir)
	defer os.Chdir("/")

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       pickerMode, // Already in picker mode
		path:       InitPathModel(tmpDir),
		Config: config.Config{
			X: 3,
			Y: 3,
		},
	}

	// Verify child dirs exist
	if len(m.path.ChildDirs) < 2 {
		t.Fatalf("Expected at least 2 child dirs, got %d", len(m.path.ChildDirs))
	}

	// Calculate child dirs position (same logic as resolvePathMouseClick)
	headerContent := renderDefaultHeaderArt(m.spinner.View())
	counterContent := m.renderProfileCounter()
	buttonsContent := m.renderProfileButtons()
	gridContent := m.renderGrid()
	pathBarContent := m.path.RenderPathBar(true)
	childDirsContent := m.path.RenderChildDirs(m.mode)

	helpText := "Path Mode | ←/→/ad: Select, ↑/s: Children, Enter: cd, e: Search, q/Esc: Back"
	help := helpStyle.Render(helpText)
	footerContent := m.renderCombinedFooter(help)

	hHeight := lipgloss.Height(headerContent)
	cHeight := lipgloss.Height(counterContent)
	bHeight := lipgloss.Height(buttonsContent)
	gHeight := lipgloss.Height(gridContent)
	pathBarH := lipgloss.Height(pathBarContent)
	childDirsH := lipgloss.Height(childDirsContent)
	fHeight := lipgloss.Height(footerContent)

	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	yOffset := (m.termHeight - totalHeight) / 2
	footerY := yOffset + hHeight + cHeight + bHeight + gHeight
	childDirsY := footerY + fHeight - childDirsH

	t.Logf("Calculated: footerY=%d, pathBarH=%d, childDirsH=%d, childDirsY=%d, childDirs count=%d",
		footerY, pathBarH, childDirsH, childDirsY, len(m.path.ChildDirs))

	// Click on first directory (index 0 in the list, which appears at childDirsY)
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
		X:      10,
		Y:      childDirsY,
	}

	updated, _ := m.resolvePathMouseClick(msg)
	mFinal := updated.(Model)

	// Should select the first directory
	if mFinal.path.SelectedChildIndex != 0 {
		t.Errorf("Expected SelectedChildIndex=0, got %d", mFinal.path.SelectedChildIndex)
	}

	t.Logf("Mode after click: %v", mFinal.mode)
}

// TestGridModePathBarClick tests clicking on path bar in grid mode enters path mode
func TestGridModePathBarClick(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       gridMode, // Start in grid mode
		path:       InitPathModel(tmpDir),
		Config: config.Config{
			X: 3,
			Y: 3,
		},
	}

	// Use getPathBarPosition to get the Y position
	pathBarY, ok := m.getPathBarPosition()
	if !ok {
		t.Fatal("Expected path bar to be visible")
	}

	t.Logf("Grid mode pathBarY=%d", pathBarY)

	// Click on path bar area
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
		X:      5,
		Y:      pathBarY,
	}

	updated, _ := m.resolveMouseClick(msg)
	mFinal := updated.(Model)

	// Should enter path mode
	if mFinal.mode != pathMode {
		t.Errorf("Expected pathMode after clicking path bar in gridMode, got %v", mFinal.mode)
	}
}

// TestPathModeClickAbovePathBarReturnsToGrid tests that clicking above the path/picker area
// in pathMode returns to gridMode.
func TestPathModeClickAbovePathBarReturnsToGrid(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       pathMode,
		path:       InitPathModel(tmpDir),
		Config:     config.Config{X: 3, Y: 3},
	}

	// Calculate pathBarBlockStart the same way resolvePathMouseClick does
	headerContent := renderDefaultHeaderArt(m.spinner.View())
	counterContent := m.renderProfileCounter()
	buttonsContent := m.renderProfileButtons()
	gridContent := m.renderGrid()
	pathBarContent := m.path.RenderPathBar(true)
	childDirsContent := m.path.RenderChildDirs(m.mode)
	helpText := "Path Mode | ←/→/ad: Select, ↑/s: Children, Enter: cd, e: Search, q/Esc: Back"
	help := helpStyle.Render(helpText)
	footerContent := m.renderCombinedFooter(help)

	hHeight := lipgloss.Height(headerContent)
	cHeight := lipgloss.Height(counterContent)
	bHeight := lipgloss.Height(buttonsContent)
	gHeight := lipgloss.Height(gridContent)
	fHeight := lipgloss.Height(footerContent)
	pathBarH := lipgloss.Height(pathBarContent)
	childDirsH := lipgloss.Height(childDirsContent)

	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	yOffset := (m.termHeight - totalHeight) / 2
	footerY := yOffset + hHeight + cHeight + bHeight + gHeight
	childDirsY := footerY + fHeight - childDirsH
	pathBarBlockStart := childDirsY - pathBarH

	clickAbove := pathBarBlockStart - 1
	if clickAbove < 0 {
		t.Skip("terminal too small to place click above path bar")
	}

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      clickAbove,
	}

	updated, _ := m.resolvePathMouseClick(msg)
	mFinal := updated.(Model)

	if mFinal.mode != gridMode {
		t.Errorf("Expected gridMode after click above path bar in pathMode, got %v (clickY=%d, pathBarBlockStart=%d)", mFinal.mode, clickAbove, pathBarBlockStart)
	}
}

// TestPickerModeClickAbovePathBarReturnsToGrid tests the same in pickerMode.
func TestPickerModeClickAbovePathBarReturnsToGrid(t *testing.T) {
	applyThemeStyles(config.Config{Theme: "dracula"})

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       pickerMode,
		path:       InitPathModel(tmpDir),
		Config:     config.Config{X: 3, Y: 3},
	}

	headerContent := renderDefaultHeaderArt(m.spinner.View())
	counterContent := m.renderProfileCounter()
	buttonsContent := m.renderProfileButtons()
	gridContent := m.renderGrid()
	pathBarContent := m.path.RenderPathBar(true)
	childDirsContent := m.path.RenderChildDirs(m.mode)
	helpText := "Path Mode | ←/→/ad: Select, ↑/s: Children, Enter: cd, e: Search, q/Esc: Back"
	help := helpStyle.Render(helpText)
	footerContent := m.renderCombinedFooter(help)

	hHeight := lipgloss.Height(headerContent)
	cHeight := lipgloss.Height(counterContent)
	bHeight := lipgloss.Height(buttonsContent)
	gHeight := lipgloss.Height(gridContent)
	fHeight := lipgloss.Height(footerContent)
	pathBarH := lipgloss.Height(pathBarContent)
	childDirsH := lipgloss.Height(childDirsContent)

	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	yOffset := (m.termHeight - totalHeight) / 2
	footerY := yOffset + hHeight + cHeight + bHeight + gHeight
	childDirsY := footerY + fHeight - childDirsH
	pathBarBlockStart := childDirsY - pathBarH

	clickAbove := pathBarBlockStart - 1
	if clickAbove < 0 {
		t.Skip("terminal too small to place click above path bar")
	}

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      clickAbove,
	}

	updated, _ := m.resolvePathMouseClick(msg)
	mFinal := updated.(Model)

	if mFinal.mode != gridMode {
		t.Errorf("Expected gridMode after click above path bar in pickerMode, got %v (clickY=%d, pathBarBlockStart=%d)", mFinal.mode, clickAbove, pathBarBlockStart)
	}
}
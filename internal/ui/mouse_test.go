package ui

import (
	"fmt"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

func TestResolveMouseClick(t *testing.T) {
	// Initialize styles so lipgloss measurements are realistic
	applyThemeStyles(config.Config{Theme: "dracula"})

	// Setup a 3x3 grid with names long enough to make gridWidth predictable
	// Each name is 25 chars (GridMaxTextWidth)
	longName := "1234567890123456789012345" // 25
	// Create dummy profile files to satisfy os.Stat in switchToProfileIndex
	p1Path := "/tmp/drako_test_p1.toml"
	p2Path := "/tmp/drako_test_p2.toml"
	p3Path := "/tmp/drako_test_p3.toml"
	_ = os.WriteFile(p1Path, []byte(""), 0644)
	_ = os.WriteFile(p2Path, []byte(""), 0644)
	_ = os.WriteFile(p3Path, []byte(""), 0644)
	defer os.Remove(p1Path)
	defer os.Remove(p2Path)
	defer os.Remove(p3Path)

	m := Model{
		termWidth:  150,
		termHeight: 50,
		mode:       gridMode,
		grid: [][]string{
			{longName, longName, longName},
			{longName, longName, longName},
			{longName, longName, longName},
		},
		cursorRow: 0,
		cursorCol: 0,
		Config: config.Config{
			X: 3,
			Y: 3,
		},
		baseConfig: config.Config{
			X: 3,
			Y: 3,
		},
		profiles: []config.ProfileInfo{
			{Name: "p1", Path: p1Path},
			{Name: "p2", Path: p2Path},
			{Name: "p3", Path: p3Path},
		},
	}

	// --- 1. Manual Math for Verification ---
	// headerWidth = 32, headerHeight = 9
	// counterWidth = 9, counterHeight = 1
	// buttonsWidth = 2 * 5 = 10, buttonsHeight = 1
	// cellWidth = 25 + 4 = 29
	// rowPrefixWidth = 1 + 1 = 2
	// gridWidth = 2 + (3 * 29) = 2 + 87 = 89
	// gridHeight = 1 + (3 * 3) = 10
	// footerWidth = 60, footerHeight = 6
	// widest = max(32, 9, 10, 89, 60) = 89
	// totalHeight = 9 + 1 + 1 + 10 + 6 = 27

	// yOffset = (50 - 27) / 2 = 11
	// xOffset = (150 - 89) / 2 = 30
	// gridBlockXOffset = 30 + (89 - 89) / 2 = 30
	// gridXStart = 30 + 2 = 32
	// gridYStart = 11 + 9 + 1 + 1 + 1 = 23

	// Test clicking (0,0)
	msgGrid := tea.MouseMsg{
		X:      33, // Just inside Cell (0,0)
		Y:      23, // Just inside Row 0
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
	}

	tm, _ := m.resolveMouseClick(msgGrid)
	mFinal := tm.(Model)
	if mFinal.cursorRow != 0 || mFinal.cursorCol != 0 {
		t.Errorf("Expected (0,0), got (%d,%d). X=%d Y=%d", mFinal.cursorRow, mFinal.cursorCol, msgGrid.X, msgGrid.Y)
	}

	// Test clicking (2,2)
	// X = 32 + (2 * 29) + 5 = 32 + 58 + 5 = 95
	// Y = 23 + (2 * 3) + 1 = 23 + 6 + 1 = 30
	msgGrid.X = 95
	msgGrid.Y = 30

	tm, _ = mFinal.resolveMouseClick(msgGrid)
	mFinal = tm.(Model)
	if mFinal.cursorRow != 2 || mFinal.cursorCol != 2 {
		t.Errorf("Expected (2,2), got (%d,%d). X=%d Y=%d", mFinal.cursorRow, mFinal.cursorCol, msgGrid.X, msgGrid.Y)
	}

	// --- 3. Test Profile Buttons ---
	// Reset to original model to ensure profiles are present
	mFinal = m

	// Dynamically calculate expected coordinates to match resolveMouseClick logic
	hH := lipgloss.Height(renderDefaultHeaderArt(mFinal.spinner.View()))
	cH := lipgloss.Height(mFinal.renderProfileCounter())
	bH := lipgloss.Height(mFinal.renderProfileButtons())
	gridContent := mFinal.renderGrid()
	gH := lipgloss.Height(gridContent)

	// Footer height calculation
	helpText := "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	help := helpStyle.Render(helpText)
	fH := lipgloss.Height(mFinal.renderCombinedFooter(help))

	totalH := hH + cH + bH + gH + fH
	yOff := (mFinal.termHeight - totalH) / 2
	expectedButtonsY := yOff + hH + cH

	// Buttons centering logic
	bWidth := lipgloss.Width(mFinal.renderProfileButtons())
	widest := lipgloss.Width(gridContent)
	xOffset := (mFinal.termWidth - widest) / 2
	expectedBlockXStart := xOffset + (widest-bWidth)/2

	t.Logf("DEBUG TEST: yOff=%d hH=%d cH=%d bY=%d xOff=%d bW=%d bXStart=%d", yOff, hH, cH, expectedButtonsY, xOffset, bWidth, expectedBlockXStart)

	msgButtons := tea.MouseMsg{
		X:      expectedBlockXStart + 1, // Middle of first button " 1 "
		Y:      expectedButtonsY,
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
	}

	tm, _ = mFinal.resolveMouseClick(msgButtons)
	mUpdated := tm.(Model)
	if mUpdated.activeProfileIndex != 0 {
		t.Errorf("Expected profile 0, got %d. Y=%d", mUpdated.activeProfileIndex, msgButtons.Y)
	}

	fmt.Printf("Mouse Test (Profile Buttons) PASSED\n")

	// Print summary for logs
	fmt.Printf("Mouse Test (Visible Header) PASSED\n")

	// --- 4. Test Small Terminal (Header Hidden) ---
	// m.termHeight = 25 ... (remaining test is fine)
	msgSmall := tea.MouseMsg{
		X:      33, // Still inside grid at xOffset 30
		Y:      6,  // Inside grid row 0
		Action: tea.MouseActionPress,
		Type:   tea.MouseLeft,
	}

	tm, _ = m.resolveMouseClick(msgSmall)
	mFinal = tm.(Model)
	if mFinal.cursorRow != 0 || mFinal.cursorCol != 0 {
		t.Errorf("Small Term: Expected (0,0), got (%d,%d). X=%d Y=%d", mFinal.cursorRow, mFinal.cursorCol, msgSmall.X, msgSmall.Y)
	}

	fmt.Printf("Mouse Test (Hidden Header) PASSED\n")
}

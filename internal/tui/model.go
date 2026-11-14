package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

const (
	profileStatusDuration     = 3 * time.Second
	defaultLockTimeoutMinutes = 5
	defaultLockPumpGoal       = 6
)

// gridNav is the command grid plus the cursor moving over it, including the
// quicknav two-step (column-then-row) jump window.
type gridNav struct {
	grid      [][]string
	cursorRow int
	cursorCol int

	// navPending: a column was picked and the row digit is awaited.
	// flashActive: the row just landed on is highlighted (always cursorRow).
	// Each seq tags its tea.Tick so a timeout from an older arming is ignored.
	navPending  bool
	navSeq      int
	flashActive bool
	flashSeq    int
}

// dropdownState is an open dropdown: the grid cell that owns it, its items,
// and the current selection.
type dropdownState struct {
	row, col    int
	selectedIdx int
	items       []profile.CommandItem
}

// lockState is the idle auto-lock plus the pump-to-unlock slider.
type lockState struct {
	lastActivity  time.Time
	timeoutMins   int
	modeBefore    navMode
	progress      int
	pumpGoal      int
	lastDirection int
}

// netStatus is the rendered online/traffic status lines and the meter that
// feeds them.
type netStatus struct {
	online  string
	traffic string
	meter   host.TrafficMeter
}

// profileState is the profile lifecycle: the discovered profiles, the active
// one, pivot pinning, the transient status message, and the broken-profile
// error queue.
type profileState struct {
	base               settings.Config
	profiles           []lair.ProfileInfo
	activeIndex        int
	rescue             bool // the rescue grid is showing; activeIndex is the deck to go back to
	configDir          string
	pivotName          string
	locked             bool
	statusMessage      string
	statusPositive     bool
	nextTimerID        int
	statusClearTimerID int
	pendingErrors      []lair.ProfileParseError
	errorQueueActive   bool
	acknowledged       map[string]bool
	// sessionProfile is the profile the user explicitly switched to this
	// session; config reloads re-select it (unless a pivot lock wins).
	sessionProfile string
}

type Model struct {
	// What the host is asked to do once the TUI quits; read through Outcome.
	selected      launch.Selection   // the entry to run
	selectedBatch []launch.Selection // entries for a batch launch
	selectedTabs  []int              // how those entries split into tabs, chosen in the dialog (nil: default)
	quitting      bool
	exitCode      int    // process exit code the host should use once the TUI is down
	exitMessage   string // why it is quitting, when the host should say so

	GlassrootMode bool
	Config        settings.Config // effective config

	mode         navMode
	previousMode navMode
	termWidth    int
	termHeight   int

	styles  Styles
	spinner spinnerModel

	gridNav   gridNav
	dropdown  dropdownState
	lock      lockState
	net       netStatus
	profile   profileState
	path      PathModel
	inventory inventoryModel
	leader    leaderState
	batch     batchState
	sheet     *sheetState // the key sheet; nil while closed

	activeDetail *DetailState // Single source of truth for detail view

	notices []DetailState // from the host, each shown once (see Session.Notices)
	setup   Setup         // what the host told the TUI at its creation
}

func (m *Model) applyConfig(cfg settings.Config) {
	m.styles = BuildStyles(cfg, m.setup.ConfigDir)

	m.gridNav.grid = profile.BuildGrid(cfg.X, cfg.Y, cfg.Commands)
	m.gridNav.clampCursor()
	m.Config = cfg
	if m.spinner.frames == nil {
		m.spinner = newSpinner()
	}
	m.spinner.style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// Initialize lock timeout (default minutes if not set)
	if cfg.LockTimeoutMinutes != nil && *cfg.LockTimeoutMinutes > 0 {
		m.lock.timeoutMins = *cfg.LockTimeoutMinutes
	} else {
		m.lock.timeoutMins = defaultLockTimeoutMinutes
	}

	if m.lock.pumpGoal <= 0 {
		m.lock.pumpGoal = defaultLockPumpGoal
	}
}

// protectsMode reports whether the current mode must not be silently
// changed by a background event: the security lock, or a field mid-
// capturing typed characters (capturingText).
func (m Model) protectsMode() bool {
	return m.mode == lockedMode || m.capturingText()
}

// showsGridCells reports whether mode displays cells of the grid, which a reload
// replaces; inventory, path, the lock screen and a note stay put.
func (m Model) showsGridCells(mode navMode) bool {
	explains := mode == infoMode && m.activeDetail != nil && m.activeDetail.OfCell
	return mode == dropdownMode || mode == batchMode || explains
}

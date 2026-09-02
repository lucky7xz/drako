package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

const (
	profileStatusDuration     = 3 * time.Second
	defaultLockTimeoutMinutes = 5
	defaultLockPumpGoal       = 6
)

// gridNav is the command grid plus the cursor moving over it, including the
// quicknav two-step (column-then-row) jump timer.
type gridNav struct {
	grid      [][]string
	cursorRow int
	cursorCol int
	timer     *time.Timer
}

// dropdownState is an open dropdown: the grid cell that owns it, its items,
// and the current selection.
type dropdownState struct {
	row, col    int
	selectedIdx int
	items       []config.CommandItem
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
	meter   core.TrafficMeter
}

// profileState is the profile lifecycle: the discovered profiles, the active
// one, pivot pinning, the transient status message, and the broken-profile
// error queue.
type profileState struct {
	base               config.Config
	profiles           []config.ProfileInfo
	activeIndex        int
	configDir          string
	pivotName          string
	locked             bool
	statusMessage      string
	statusPositive     bool
	nextTimerID        int
	statusClearTimerID int
	pendingErrors      []config.ProfileParseError
	errorQueueActive   bool
	acknowledged       map[string]bool
	// sessionProfile is the profile the user explicitly switched to this
	// session; config reloads re-select it (unless a pivot lock wins).
	sessionProfile string
}

type Model struct {
	Selected      string
	SelectedBatch []string // cell names for a batch launch (read by app.go after quit)
	SelectedTabs  []int    // how those cells split into tabs, chosen in the dialog
	Quitting      bool
	ExitCode      int // process exit code the host should use once the TUI is down
	GlassrootMode bool
	Config        config.Config // effective config (read by app.go after quit)

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

	activeDetail *DetailState // Single source of truth for detail view
}

// clampCursor pulls the cursor back inside the grid bounds after the grid
// changes shape (e.g. a profile switch to a smaller layout).
func (g *gridNav) clampCursor() {
	if len(g.grid) == 0 {
		return
	}
	if g.cursorRow >= len(g.grid) {
		g.cursorRow = len(g.grid) - 1
	}
	if g.cursorRow < 0 {
		g.cursorRow = 0
	}
	if len(g.grid[0]) > 0 {
		if g.cursorCol >= len(g.grid[0]) {
			g.cursorCol = len(g.grid[0]) - 1
		}
		if g.cursorCol < 0 {
			g.cursorCol = 0
		}
	}
}

func (m *Model) applyConfig(cfg config.Config) {
	m.styles = BuildStyles(cfg)

	m.gridNav.grid = config.BuildGrid(cfg)
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

func (m *Model) applyBundle(bundle config.ConfigBundle) {
	m.profile.base = bundle.Base
	profiles := bundle.Profiles
	m.profile.profiles = profiles
	if bundle.ActiveIndex < 0 || bundle.ActiveIndex >= len(profiles) {
		m.profile.activeIndex = 0
	} else {
		m.profile.activeIndex = bundle.ActiveIndex
	}
	m.applyConfig(bundle.Config)
	m.profile.configDir = bundle.ConfigDir
	m.profile.pivotName = bundle.LockedName
	m.profile.locked = strings.TrimSpace(bundle.LockedName) != ""
}

// presentNextBrokenProfile pops the next pending broken profile error and configures infoMode to display it.
func (m Model) presentNextBrokenProfile() Model {
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
			// Trigger Rescue Mode if we just finished processing a queue
			rescueCfg := config.RescueConfig()
			rescueCfg.ApplyDefaults()
			m.applyConfig(rescueCfg)
		}

		// Safe reset to Grid Mode
		m.mode = gridMode
		m.activeDetail = nil
		m.profile.errorQueueActive = false
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
	if e.Name == paths.ConfigFileName || strings.HasSuffix(e.Path, paths.ConfigFileName) {
		// Specific message for the main config
		desc = "If your config.toml is invalid, **Rescue Mode** can be helpful.\n" +
			"**Warning:** Default keybindings are in effect. Your custom keys may not work (since the config.toml which stores them is invalid).\n\n" +
			"Available Actions:\n" +
			"• **Edit Config**: Open the config.toml in your default editor and fix the syntax error.\n" +
			"• **Reset Config**: The command `drako reset -c` will delete the config.toml. Drako reinstantiates it with default settings.\n" +
			"• **Documentation**: View default controls on Documentation website.\n\n" +
			"• **Exit Rescue Mode**: You can still keep using drako by exiting rescue mode (the top-left button, the error will keep showing up though)."

	} else if strings.Contains(e.Err, "empty profile file") {
		desc += "The file is completely empty. Either add valid TOML configuration or move/delete the file via Inventory (i).\n\n"
	} else if strings.Contains(e.Err, "no settings found") {
		desc += "The file exists but contains no configuration settings. Either add valid TOML configuration or move/delete the file via Inventory (i).\n\n"
	} else {
		desc += "The file has a TOML syntax error. Either fix the syntax error or move/delete the file via Inventory (i).\n\n"
	}
	desc += "Press any key to continue to the next error, or 'y' to copy error details to clipboard."

	m.activeDetail = &DetailState{
		Title:       fmt.Sprintf("Profile error: %s", e.Name),
		KeyLabel:    "Error",
		Value:       fmt.Sprintf("Path: %s\nError: %s", e.Path, strings.TrimSpace(e.Err)),
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
	m.previousMode = gridMode
	m.profile.errorQueueActive = false
	m.activeDetail = &DetailState{
		Title:    "Profile no longer available",
		KeyLabel: "Profile",
		Value:    name,
		Description: "This profile is no longer available — it was deleted or moved to " +
			"the inventory. drako has dropped into Rescue Mode (any lock on it has been " +
			"cleared).\n\n" +
			"To get back to work:\n" +
			"• Inventory (i): equip a profile.\n" +
			"• Cycle (o / p): jump to a still-equipped profile.\n" +
			"• Exit Rescue Mode: the top-left button of the grid.\n\n" +
			"Press any key to dismiss.",
		Meta: []DetailMeta{
			{Label: "CWD", Value: m.profile.configDir},
		},
	}
	m.mode = infoMode
	return m
}

// activeName returns the display name of the active profile, defaulting to
// "Rescue" when there are no profiles or the name is blank (no profiles on
// disk means the compiled-in rescue grid is what's showing).
func (p profileState) activeName() string {
	if len(p.profiles) == 0 {
		return "Rescue"
	}
	idx := p.activeIndex
	if idx < 0 || idx >= len(p.profiles) {
		idx = 0
	}
	name := p.profiles[idx].Name
	if strings.TrimSpace(name) == "" {
		return "Rescue"
	}
	return name
}

// ActiveProfileName is the name of the currently active profile, or "" when
// no profiles are loaded. The app layer exports it to spawned commands as
// DRAKO_PROFILE.
func (m Model) ActiveProfileName() string {
	p := m.profile
	if len(p.profiles) == 0 || p.activeIndex < 0 || p.activeIndex >= len(p.profiles) {
		return ""
	}
	return p.profiles[p.activeIndex].Name
}

// Session is where a run left off, handed to the next one. drako quits the TUI
// so a launched command gets a clean terminal, which means the model that knew
// your position is thrown away and rebuilt -- this is the value that survives
// that gap. It lives for the process and is never written down.
//
// Names, not coordinates. InitialModel re-reads the config every lap, so by the
// time you return, row 1 column 1 may be a different command: you edited a
// profile, or the command you ran switched one. Following the name lands you on
// the cell you actually ran, wherever it moved to.
type Session struct {
	Cell    string
	Profile string
}

// Session snapshots the position for Restore to re-establish once the config
// has been re-read.
func (m Model) Session() Session {
	s := Session{Profile: m.ActiveProfileName()}

	row, col := m.gridNav.cursorRow, m.gridNav.cursorCol
	if row >= 0 && row < len(m.gridNav.grid) && col >= 0 && col < len(m.gridNav.grid[row]) {
		s.Cell = m.gridNav.grid[row][col]
	}
	return s
}

// Restore puts the cursor back. The grid has been rebuilt from a config that may
// have changed since, so nothing carried over is trusted: both names are looked
// up afresh and anything missing is simply left alone.
//
// The profile goes first. Switching one replaces the grid, so a cursor set
// before the switch would index into a grid that no longer exists.
//
// Restore never touches Quitting or ExitCode. It must be called *after* the
// host's glassroot gate, so that a session glassroot decided to end cannot be
// revived by a carried value.
//
// A carried profile outranks the pivot lock, which is the one behaviour this
// changes. The pivot answers "which profile does a fresh process start on"
// (config.resolveRequested: override > pivot > DRAKO_PROFILE > config.toml), and
// it still does; a lap of the launch loop is not a fresh start, and an explicit
// mid-session switch is the more recent statement of intent. Nothing is granted
// by this: the pivot pins a starting point, it is not an access control, and
// glassroot already contracts that a session can reach every equipped profile.
func (m *Model) Restore(s Session) {
	if s.Profile != "" && s.Profile != m.ActiveProfileName() {
		for i, p := range m.profile.profiles {
			if p.Name != s.Profile {
				continue
			}
			// switchToProfileIndex is the real path: it re-applies the config
			// overlay and calls applyConfig. Setting activeIndex alone would
			// leave the model half-switched.
			if updated, ok := m.switchToProfileIndex(i); ok {
				*m = updated
			}
			break
		}
	}

	if s.Cell == "" {
		return
	}
	for row, cells := range m.gridNav.grid {
		for col, name := range cells {
			if name == s.Cell {
				m.gridNav.cursorRow, m.gridNav.cursorCol = row, col
				return
			}
		}
	}
}

func InitialModel(glassrootMode bool) Model {
	path, err := os.Getwd()
	if err != nil {
		path = "could not get path"
	}

	bundle, err := config.LoadConfig(nil)
	if err != nil {
		// No config directory means nothing to run from; hand the host a
		// model that is already done and let app.Run exit with the code.
		fmt.Fprintf(os.Stderr, "drako: %v\n", err)
		return Model{Quitting: true, ExitCode: 1, GlassrootMode: glassrootMode}
	}

	if glassrootMode && glassrootRejectsBundle(bundle) {
		// Glassroot never exposes TOML contents or Rescue Mode. Hand the host
		// a model that is already done; app.Run exits before starting the TUI.
		return Model{Quitting: true, ExitCode: 1, GlassrootMode: true}
	}

	s := newSpinner()
	m := Model{
		net: netStatus{
			meter:   core.TrafficMeter{WindowSeconds: 7.5},
			online:  "checking...",
			traffic: "calculating...",
		},
		path:    InitPathModel(path),
		mode:    gridMode,
		spinner: s,
		profile: profileState{
			base:         bundle.Base,
			acknowledged: make(map[string]bool),
		},
		lock: lockState{
			lastActivity: time.Now(),
			modeBefore:   gridMode,
			pumpGoal:     defaultLockPumpGoal,
		},
		GlassrootMode: glassrootMode,
	}
	m.applyBundle(bundle)
	if len(bundle.Broken) > 0 {
		m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
		m.profile.errorQueueActive = true
		m = m.presentNextBrokenProfile()
	} else if bundle.DroppedProfile != "" {
		// Relaunched after the active profile was deleted out from under us:
		// explain the rescue drop instead of showing a bare rescue grid.
		m = m.presentDroppedProfileNote(bundle.DroppedProfile)
	}

	return m
}

func (m *Model) scheduleStatusClearTimer() tea.Cmd {
	m.profile.nextTimerID++
	id := m.profile.nextTimerID
	m.profile.statusClearTimerID = id
	return tea.Tick(profileStatusDuration, func(time.Time) tea.Msg {
		return profileStatusClearMsg{id: id}
	})
}

func (m *Model) setProfileStatus(message string, positive bool) tea.Cmd {
	m.profile.statusMessage = message
	m.profile.statusPositive = positive
	if strings.TrimSpace(message) == "" {
		m.profile.statusClearTimerID = 0
		return nil
	}
	return m.scheduleStatusClearTimer()
}

func (m *Model) toggleProfileLock() tea.Cmd {
	if strings.TrimSpace(m.profile.configDir) == "" {
		return m.setProfileStatus("Pivot unavailable", false)
	}

	currentName := m.profile.activeName()
	normCurrent := profiles.NormalizeName(currentName)
	normPivot := profiles.NormalizeName(m.profile.pivotName)

	var err error
	var messageCmd tea.Cmd

	if m.profile.locked && normPivot == normCurrent && m.profile.pivotName != "" {
		err = config.DeletePivotProfile(m.profile.configDir)
		if err == nil {
			m.profile.locked = false
			m.profile.pivotName = ""
			messageCmd = m.setProfileStatus("Pivot cleared", false)
		}
	} else {
		err = config.WritePivotLocked(m.profile.configDir, currentName)
		if err == nil {
			m.profile.locked = true
			m.profile.pivotName = currentName
			messageCmd = m.setProfileStatus(fmt.Sprintf("Pinned %s", currentName), true)
		}
	}

	if err != nil {
		log.Printf("pivot update failed: %v", err)
		return m.setProfileStatus("Pivot error", false)
	}
	return messageCmd
}

func (m Model) enterLockedMode() Model {
	if m.mode != lockedMode {
		m.lock.modeBefore = m.mode
	}
	m.mode = lockedMode
	m.lock.progress = 0
	m.lock.lastDirection = 0
	return m
}

func (m Model) exitLockedMode() Model {
	if m.mode == lockedMode {
		m.mode = m.lock.modeBefore
	}
	m.lock.lastActivity = time.Now()
	m.lock.progress = 0
	m.lock.lastDirection = 0
	return m
}

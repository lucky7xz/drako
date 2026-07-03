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
}

type Model struct {
	Selected      string
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
	if len(profiles) == 0 {
		profiles = []config.ProfileInfo{{Name: "Core"}}
	}
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
	for len(m.profile.pendingErrors) > 0 {
		e := m.profile.pendingErrors[0]
		if m.profile.acknowledged[e.Path] { // Track by Path to be specific
			m.profile.pendingErrors = m.profile.pendingErrors[1:]
			continue
		}
		break
	}

	if len(m.profile.pendingErrors) == 0 {
		// Queue exhausted.
		if m.profile.errorQueueActive {
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
			"• **Exit Rescue Mode**: You can still keep using drako by exiting rescue mode (the button on the bottom, the error will keep showing up though)."

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

// activeName returns the display name of the active profile, defaulting to
// "Core" when there are no profiles or the name is blank.
func (p profileState) activeName() string {
	if len(p.profiles) == 0 {
		return "Core"
	}
	idx := p.activeIndex
	if idx < 0 || idx >= len(p.profiles) {
		idx = 0
	}
	name := p.profiles[idx].Name
	if strings.TrimSpace(name) == "" {
		return "Core"
	}
	return name
}

func InitialModel(glassrootMode bool) Model {
	path, err := os.Getwd()
	if err != nil {
		path = "could not get path"
	}

	bundle := config.LoadConfig(nil)

	if glassrootMode && len(bundle.Broken) > 0 {
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

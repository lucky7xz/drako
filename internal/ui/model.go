package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

const (
	profileStatusDuration     = 3 * time.Second
	defaultLockTimeoutMinutes = 5
	defaultLockPumpGoal       = 6
)

type Model struct {
	grid        [][]string
	cursorRow   int
	cursorCol   int
	termWidth   int
	termHeight  int
	Selected    string
	Quitting    bool
	mode        navMode
	spinner     spinner.Model
	inputBuffer string

	path PathModel

	onlineStatus      string
	traffic           string
	sentHistory       []uint64
	recvHistory       []uint64
	timeHistory       []time.Time
	trafficAvgSeconds float64

	baseConfig            config.Config
	Config                config.Config
	profiles              []config.ProfileInfo
	activeProfileIndex    int
	configDir             string
	pivotProfileName      string
	profileLocked         bool
	profileStatusMessage  string
	profileStatusPositive bool

	nextTimerID        int
	statusClearTimerID int

	navigationTimer *time.Timer

	inventory inventoryModel

	dropdownRow         int
	dropdownCol         int
	dropdownSelectedIdx int
	dropdownItems       []config.CommandItem

	previousMode navMode
	activeDetail *DetailState // Single source of truth for detail view

	pendingProfileErrors    []config.ProfileParseError
	profileErrorQueueActive bool

	lastActivityTime  time.Time
	lockTimeoutMins   int
	modeBeforeLock    navMode
	lockProgress      int
	lockPumpGoal      int
	lockLastDirection int

	acknowledgedErrors map[string]bool
}

func (m *Model) applyConfig(cfg config.Config) {
	config.ClampConfig(&cfg)
	applyThemeStyles(cfg)

	m.grid = config.BuildGrid(cfg)
	if len(m.grid) > 0 {
		if m.cursorRow >= len(m.grid) {
			m.cursorRow = len(m.grid) - 1
		}
		if m.cursorRow < 0 {
			m.cursorRow = 0
		}
		if len(m.grid[0]) > 0 {
			if m.cursorCol >= len(m.grid[0]) {
				m.cursorCol = len(m.grid[0]) - 1
			}
			if m.cursorCol < 0 {
				m.cursorCol = 0
			}
		}
	}
	m.Config = cfg
	m.inputBuffer = ""
	if m.spinner.Spinner.Frames == nil {
		m.spinner = spinner.New()
		m.spinner.Spinner = spinner.Line
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// Initialize lock timeout (default minutes if not set)
	if cfg.LockTimeoutMinutes != nil && *cfg.LockTimeoutMinutes > 0 {
		m.lockTimeoutMins = *cfg.LockTimeoutMinutes
	} else {
		m.lockTimeoutMins = defaultLockTimeoutMinutes
	}

	if m.lockPumpGoal <= 0 {
		m.lockPumpGoal = defaultLockPumpGoal
	}
}

func (m *Model) applyBundle(bundle config.ConfigBundle) {
	m.baseConfig = bundle.Base
	profiles := bundle.Profiles
	if len(profiles) == 0 {
		profiles = []config.ProfileInfo{{Name: "Core"}}
	}
	m.profiles = profiles
	if bundle.ActiveIndex < 0 || bundle.ActiveIndex >= len(profiles) {
		m.activeProfileIndex = 0
	} else {
		m.activeProfileIndex = bundle.ActiveIndex
	}
	m.applyConfig(bundle.Config)
	m.configDir = bundle.ConfigDir
	m.pivotProfileName = bundle.LockedName
	m.profileLocked = strings.TrimSpace(bundle.LockedName) != ""
}

// presentNextBrokenProfile pops the next pending broken profile error and configures infoMode to display it.
func (m Model) presentNextBrokenProfile() Model {
	// Filter out already acknowledged errors
	for len(m.pendingProfileErrors) > 0 {
		e := m.pendingProfileErrors[0]
		if m.acknowledgedErrors[e.Path] { // Track by Path to be specific
			m.pendingProfileErrors = m.pendingProfileErrors[1:]
			continue
		}
		break
	}

	if len(m.pendingProfileErrors) == 0 {
		// Queue exhausted.
		if m.profileErrorQueueActive {
			// Trigger Rescue Mode if we just finished processing a queue
			rescueCfg := config.RescueConfig()
			rescueCfg.ApplyDefaults()
			m.applyConfig(rescueCfg)
		}

		// Safe reset to Grid Mode
		m.mode = gridMode
		m.activeDetail = nil
		m.profileErrorQueueActive = false
		return m
	}

	e := m.pendingProfileErrors[0]
	m.pendingProfileErrors = m.pendingProfileErrors[1:]

	// Mark as acknowledged
	m.acknowledgedErrors[e.Path] = true

	// Capture previous mode only if we are transitioning FROM a valid mode.
	// If we are already in infoMode (chained errors), we keep the original previousMode.
	// However, since we now force return to Grid Mode at the end, this is less critical
	// but good for hygiene if we ever change the exit logic.
	if m.mode != infoMode {
		m.previousMode = m.mode
	}

	desc := "This profile has an error and was hidden from selection.\n\n"
	if strings.Contains(e.Err, "empty profile file") {
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
			{Label: "CWD", Value: m.configDir},
		},
	}
	m.mode = infoMode
	return m
}

func (m Model) activeProfileName() string {
	if len(m.profiles) == 0 {
		return "Core"
	}
	idx := m.activeProfileIndex
	if idx < 0 || idx >= len(m.profiles) {
		idx = 0
	}
	name := m.profiles[idx].Name
	if strings.TrimSpace(name) == "" {
		return "Core"
	}
	return name
}

func InitialModel() Model {
	path, err := os.Getwd()
	if err != nil {
		path = "could not get path"
	}

	bundle := config.LoadConfig(nil)

	s := spinner.New()
	s.Spinner = spinner.Line
	m := Model{
		cursorRow:          0,
		cursorCol:          0,
		trafficAvgSeconds:  7.5,
		onlineStatus:       "checking...",
		traffic:            "calculating...",
		path:               InitPathModel(path),
		mode:               gridMode,
		spinner:            s,
		baseConfig:         bundle.Base,
		lastActivityTime:   time.Now(),
		modeBeforeLock:     gridMode,
		lockPumpGoal:       defaultLockPumpGoal,
		acknowledgedErrors: make(map[string]bool),
	}
	m.applyBundle(bundle)
	if len(bundle.Broken) > 0 {
		m.pendingProfileErrors = append(m.pendingProfileErrors, bundle.Broken...)
		m.profileErrorQueueActive = true
		m = m.presentNextBrokenProfile()
	}

	return m
}

func (m *Model) scheduleStatusClearTimer() tea.Cmd {
	m.nextTimerID++
	id := m.nextTimerID
	m.statusClearTimerID = id
	return tea.Tick(profileStatusDuration, func(time.Time) tea.Msg {
		return profileStatusClearMsg{id: id}
	})
}

func (m *Model) setProfileStatus(message string, positive bool) tea.Cmd {
	m.profileStatusMessage = message
	m.profileStatusPositive = positive
	if strings.TrimSpace(message) == "" {
		m.statusClearTimerID = 0
		return nil
	}
	return m.scheduleStatusClearTimer()
}

func (m *Model) toggleProfileLock() tea.Cmd {
	if strings.TrimSpace(m.configDir) == "" {
		return m.setProfileStatus("Pivot unavailable", false)
	}

	currentName := m.activeProfileName()
	normCurrent := config.NormalizeProfileName(currentName)
	normPivot := config.NormalizeProfileName(m.pivotProfileName)

	var err error
	var messageCmd tea.Cmd

	if m.profileLocked && normPivot == normCurrent && m.pivotProfileName != "" {
		err = config.DeletePivotProfile(m.configDir)
		if err == nil {
			m.profileLocked = false
			m.pivotProfileName = ""
			messageCmd = m.setProfileStatus("Pivot cleared", false)
		}
	} else {
		err = config.WritePivotLocked(m.configDir, currentName)
		if err == nil {
			m.profileLocked = true
			m.pivotProfileName = currentName
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
		m.modeBeforeLock = m.mode
	}
	m.mode = lockedMode
	m.lockProgress = 0
	m.lockLastDirection = 0
	return m
}

func (m Model) exitLockedMode() Model {
	if m.mode == lockedMode {
		m.mode = m.modeBeforeLock
	}
	m.lastActivityTime = time.Now()
	m.lockProgress = 0
	m.lockLastDirection = 0
	return m
}

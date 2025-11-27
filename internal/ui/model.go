package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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

	pathComponents     []string
	selectedPathIndex  int
	childDirs          []string
	childDirsError     error
	selectedChildIndex int

	onlineStatus      string
	traffic           string
	sentHistory       []uint64
	recvHistory       []uint64
	timeHistory       []time.Time
	trafficAvgSeconds float64

	currentPath string

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

	previousMode    navMode
	infoTitle       string
	infoCommand     string
	infoDescription string
	infoExecMode    string
	infoAutoClose   bool
	infoCwd         string

	pendingProfileErrors    []config.ProfileParseError
	profileErrorQueueActive bool

	lastActivityTime  time.Time
	lockTimeoutMins   int
	modeBeforeLock    navMode
	lockProgress      int
	lockPumpGoal      int
	lockLastDirection int
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
		profiles = []config.ProfileInfo{{Name: "Default"}}
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
	if len(m.pendingProfileErrors) == 0 {
		return m
	}
	e := m.pendingProfileErrors[0]
	m.pendingProfileErrors = m.pendingProfileErrors[1:]

	m.previousMode = m.mode
	m.infoTitle = fmt.Sprintf("Profile error: %s", e.Name)
	// Put actionable details into infoCommand so users can copy with 'y'
	m.infoCommand = fmt.Sprintf("Path: %s\nError: %s", e.Path, strings.TrimSpace(e.Err))
	m.infoDescription = "This profile has an error and was hidden from selection.\n\n"
	if strings.Contains(e.Err, "empty profile file") {
		m.infoDescription += "The file is completely empty. Either add valid TOML configuration or move/delete the file via Inventory (i).\n\n"
	} else if strings.Contains(e.Err, "no settings found") {
		m.infoDescription += "The file exists but contains no configuration settings. Either add valid TOML configuration or move/delete the file via Inventory (i).\n\n"
	} else {
		m.infoDescription += "The file has a TOML syntax error. Either fix the syntax error or move/delete the file via Inventory (i).\n\n"
	}
	m.infoDescription += "Press any key to continue to the next error, or 'y' to copy error details to clipboard."
	m.infoExecMode = ""
	m.infoAutoClose = false
	m.infoCwd = m.configDir
	m.mode = infoMode
	return m
}

func (m Model) activeProfileName() string {
	if len(m.profiles) == 0 {
		return "Default"
	}
	idx := m.activeProfileIndex
	if idx < 0 || idx >= len(m.profiles) {
		idx = 0
	}
	name := m.profiles[idx].Name
	if strings.TrimSpace(name) == "" {
		return "Default"
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
		cursorRow:         0,
		cursorCol:         0,
		trafficAvgSeconds: 7.5,
		onlineStatus:      "checking...",
		traffic:           "calculating...",
		currentPath:       path,
		mode:              gridMode,
		spinner:           s,
		baseConfig:        bundle.Base,
		lastActivityTime:  time.Now(),
		modeBeforeLock:    gridMode,
		lockPumpGoal:      defaultLockPumpGoal,
	}
	m.applyBundle(bundle)
	if len(bundle.Broken) > 0 {
		m.pendingProfileErrors = append(m.pendingProfileErrors, bundle.Broken...)
		m.profileErrorQueueActive = true
		m = m.presentNextBrokenProfile()
	}
	m.updatePathComponents()
	m.listChildDirs()
	return m
}

func (m *Model) updatePathComponents() {
	home, err := os.UserHomeDir()
	path := m.currentPath
	if err == nil {
		if path == home {
			path = "~"
		} else if strings.HasPrefix(path, home+"/") {
			path = "~/" + strings.TrimPrefix(path, home+"/")
		}
	}

	var components []string
	if path == "/" {
		components = []string{"/"}
	} else {
		components = strings.Split(path, string(os.PathSeparator))
	}

	if len(components) > 1 && components[0] == "" {
		components[0] = "/"
	}

	m.pathComponents = components
	m.selectedPathIndex = len(m.pathComponents) - 1
}

func (m *Model) listChildDirs() {
	m.childDirs = []string{}
	m.childDirsError = nil
	path := m.buildPathFromComponents(m.selectedPathIndex)

	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("could not read directory %s: %v", path, err)
		m.childDirsError = err
		return
	}

	for _, f := range files {
		if f.IsDir() {
			m.childDirs = append(m.childDirs, f.Name())
		}
	}
	sort.Strings(m.childDirs)
}

func (m *Model) buildPathFromComponents(index int) string {
	home, _ := os.UserHomeDir()

	if len(m.pathComponents) == 0 {
		return m.currentPath
	}

	if len(m.pathComponents) == 1 && m.pathComponents[0] == "/" {
		return "/"
	}

	var pathToJoin []string
	if m.pathComponents[0] == "/" {
		pathToJoin = m.pathComponents[1 : index+1]
		return "/" + filepath.Join(pathToJoin...)
	} else if m.pathComponents[0] == "~" {
		pathToJoin = m.pathComponents[1 : index+1]
		return filepath.Join(home, filepath.Join(pathToJoin...))
	} else {
		pathToJoin = m.pathComponents[:index+1]
		return filepath.Join(pathToJoin...)
	}
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

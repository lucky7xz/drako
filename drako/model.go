package main

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
)

const (
	profileStatusDuration = 3 * time.Second
)

type model struct {
	grid        [][]string
	cursorRow   int
	cursorCol   int
	termWidth   int
	termHeight  int
	selected    string
	quitting    bool
	mode        navMode
	spinner     spinner.Model
	inputBuffer string

	pathComponents     []string
	selectedPathIndex  int
	childDirs          []string
	selectedChildIndex int

	onlineStatus      string
	traffic           string
	sentHistory       []uint64
	recvHistory       []uint64
	timeHistory       []time.Time
	trafficAvgSeconds float64

	currentPath string

	baseConfig            Config
	config                Config
	profiles              []ProfileInfo
	activeProfileIndex    int
	configDir             string
	pivotProfileName      string
	profileLocked         bool
	profileStatusMessage  string
	profileStatusPositive bool

	nextTimerID        int
	statusClearTimerID int

	inventory inventoryModel

	dropdownRow          int
	dropdownCol          int
	dropdownSelectedIdx  int
	dropdownItems        []CommandItem

	previousMode     navMode
	infoTitle        string
	infoCommand      string
	infoDescription  string
	infoInteractive  bool
	infoCwd          string
}

func (m *model) applyConfig(cfg Config) {
	clampConfig(&cfg)
	applyThemeStyles(cfg)

	m.grid = buildGrid(cfg)
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
	m.config = cfg
	m.inputBuffer = ""
	if m.spinner.Spinner.Frames == nil {
		m.spinner = spinner.New()
		m.spinner.Spinner = spinner.Line
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
}

func (m *model) applyBundle(bundle configBundle) {
	m.baseConfig = bundle.Base
	profiles := bundle.Profiles
	if len(profiles) == 0 {
		profiles = []ProfileInfo{{Name: "Default"}}
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

func (m model) activeProfileName() string {
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

func initialModel() model {
	path, err := os.Getwd()
	if err != nil {
		path = "could not get path"
	}

	bundle := loadConfig(nil)

	s := spinner.New()
	s.Spinner = spinner.Line
	m := model{
		cursorRow:         0,
		cursorCol:         0,
		trafficAvgSeconds: 7.5,
		onlineStatus:      "checking...",
		traffic:           "calculating...",
		currentPath:       path,
		mode:              gridMode,
		spinner:           s,
		baseConfig:        bundle.Base,
	}
	m.applyBundle(bundle)
	m.updatePathComponents()
	m.listChildDirs()
	return m
}

func (m *model) updatePathComponents() {
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

func (m *model) listChildDirs() {
	m.childDirs = []string{}
	path := m.buildPathFromComponents(m.selectedPathIndex)

	files, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() {
			m.childDirs = append(m.childDirs, f.Name())
		}
	}
	sort.Strings(m.childDirs)
}

func (m *model) buildPathFromComponents(index int) string {
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



func (m *model) scheduleStatusClearTimer() tea.Cmd {
	m.nextTimerID++
	id := m.nextTimerID
	m.statusClearTimerID = id
	return tea.Tick(profileStatusDuration, func(time.Time) tea.Msg {
		return profileStatusClearMsg{id: id}
	})
}


func (m *model) setProfileStatus(message string, positive bool) tea.Cmd {
	m.profileStatusMessage = message
	m.profileStatusPositive = positive
	if strings.TrimSpace(message) == "" {
		m.statusClearTimerID = 0
		return nil
	}
	return m.scheduleStatusClearTimer()
}


func (m *model) toggleProfileLock() tea.Cmd {
	if strings.TrimSpace(m.configDir) == "" {
		return m.setProfileStatus("Pivot unavailable", false)
	}

	currentName := m.activeProfileName()
	normCurrent := normalizeProfileName(currentName)
	normPivot := normalizeProfileName(m.pivotProfileName)

	var err error
	var messageCmd tea.Cmd

	if m.profileLocked && normPivot == normCurrent && m.pivotProfileName != "" {
		err = deletePivotProfile(m.configDir)
		if err == nil {
			m.profileLocked = false
			m.pivotProfileName = ""
			messageCmd = m.setProfileStatus("Pivot cleared", false)
		}
	} else {
		err = writePivotProfile(m.configDir, currentName)
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

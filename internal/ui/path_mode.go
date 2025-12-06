package ui

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

type PathModel struct {
	CurrentPath        string
	PathComponents     []string
	SelectedPathIndex  int
	ChildDirs          []string
	ChildDirsError     error
	SelectedChildIndex int
}

func InitPathModel(startPath string) PathModel {
	m := PathModel{
		CurrentPath: startPath,
	}
	m.UpdatePathComponents()
	m.ListChildDirs()
	return m
}

func (m *PathModel) UpdatePathComponents() {
	home, err := os.UserHomeDir()
	path := m.CurrentPath
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

	m.PathComponents = components
	m.SelectedPathIndex = len(m.PathComponents) - 1
}

func (m *PathModel) ListChildDirs() {
	m.ChildDirs = []string{}
	m.ChildDirsError = nil
	path := m.BuildPathFromComponents(m.SelectedPathIndex)

	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("could not read directory %s: %v", path, err)
		m.ChildDirsError = err
		return
	}

	for _, f := range files {
		// Basic visibility check: skip hidden files unless toggled (feature to add later)
		// For now, behave as before
		if f.IsDir() {
			m.ChildDirs = append(m.ChildDirs, f.Name())
		}
	}
	sort.Strings(m.ChildDirs)
}

func (m *PathModel) BuildPathFromComponents(index int) string {
	home, _ := os.UserHomeDir()

	if len(m.PathComponents) == 0 {
		return m.CurrentPath
	}

	if len(m.PathComponents) == 1 && m.PathComponents[0] == "/" {
		return "/"
	}

	var pathToJoin []string
	var result string

	if m.PathComponents[0] == "/" {
		pathToJoin = m.PathComponents[1 : index+1]
		result = "/" + filepath.Join(pathToJoin...)
	} else if m.PathComponents[0] == "~" {
		pathToJoin = m.PathComponents[1 : index+1]
		result = filepath.Join(home, filepath.Join(pathToJoin...))
	} else {
		pathToJoin = m.PathComponents[:index+1]
		result = filepath.Join(pathToJoin...)
	}

	// Windows Drive Root Fix
	if runtime.GOOS == "windows" && len(pathToJoin) == 1 && strings.HasSuffix(result, ":") {
		return result + string(os.PathSeparator)
	}

	return result
}

// Update handles key events when in PathMode
func (pm *PathModel) UpdatePathMode(msg tea.KeyMsg, cfg config.Config) (navMode, tea.Cmd) {
	switch {
	// Quit is handled by parent, usually
	case IsLeft(cfg.Keys, msg):
		if pm.SelectedPathIndex > 0 {
			pm.SelectedPathIndex--
			pm.ListChildDirs()
		}
	case IsRight(cfg.Keys, msg):
		if pm.SelectedPathIndex < len(pm.PathComponents)-1 {
			pm.SelectedPathIndex++
			pm.ListChildDirs()
		}
	case IsDown(cfg.Keys, msg):
		if len(pm.ChildDirs) > 0 {
			pm.SelectedChildIndex = 0
			return childMode, nil
		}
	case IsPathGridMode(cfg.Keys, msg):
		return gridMode, nil
	case IsConfirm(cfg.Keys, msg):
		targetPath := pm.BuildPathFromComponents(pm.SelectedPathIndex)
		if err := os.Chdir(targetPath); err == nil {
			pm.CurrentPath, _ = os.Getwd()
			return gridMode, func() tea.Msg { return pathChangedMsg{} }
		}
	}
	return pathMode, nil
}

// Update handles key events when in ChildMode
func (pm *PathModel) UpdateChildMode(msg tea.KeyMsg, cfg config.Config) (navMode, tea.Cmd) {
	switch {
	case IsUp(cfg.Keys, msg):
		if pm.SelectedChildIndex > 0 {
			pm.SelectedChildIndex--
		} else {
			return pathMode, nil
		}
	case IsDown(cfg.Keys, msg):
		if pm.SelectedChildIndex < len(pm.ChildDirs)-1 {
			pm.SelectedChildIndex++
		}
	case IsPathGridMode(cfg.Keys, msg):
		return gridMode, nil
	case IsConfirm(cfg.Keys, msg):
		parentPath := pm.BuildPathFromComponents(pm.SelectedPathIndex)
		targetPath := filepath.Join(parentPath, pm.ChildDirs[pm.SelectedChildIndex])
		if err := os.Chdir(targetPath); err == nil {
			pm.CurrentPath, _ = os.Getwd()
			return gridMode, func() tea.Msg { return pathChangedMsg{} }
		}
	}
	return childMode, nil
}

func (pm *PathModel) RenderPathBar(active bool) string {
	var renderedParts []string
	for i, component := range pm.PathComponents {
		var style lipgloss.Style
		if active && i == pm.SelectedPathIndex {
			style = selectedPathStyle
		} else {
			style = pathStyle
		}
		renderedParts = append(renderedParts, style.Render(component))
	}

	separator := pathSeparatorStyle.Render("/")
	return statusBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(renderedParts, separator)))
}

func (pm *PathModel) RenderChildDirs(mode navMode) string {
	if mode != childMode && mode != pathMode {
		return ""
	}
	if pm.ChildDirsError != nil {
		return offlineStyle.Render("  [cannot read directory: permission denied or path invalid]")
	}
	if len(pm.ChildDirs) == 0 {
		return helpStyle.Render("  [no sub-directories]")
	}

	var rows []string
	for i, dir := range pm.ChildDirs {
		if mode == childMode && i == pm.SelectedChildIndex {
			rows = append(rows, selectedChildDirStyle.Render("› "+dir))
		} else {
			rows = append(rows, childDirStyle.Render("  "+dir))
		}
	}

	maxVisible := 5
	start := 0
	if mode == childMode && pm.SelectedChildIndex >= maxVisible {
		start = pm.SelectedChildIndex - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(rows) {
		end = len(rows)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows[start:end]...)
}

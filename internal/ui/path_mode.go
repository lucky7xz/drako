package ui

import (
	"fmt"
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
	ShowHidden         bool
	Searching          bool
	Filter             string
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
		// Basic visibility check: skip hidden files unless toggled
		name := f.Name()
		if !m.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		// Search filter check
		if m.Filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(m.Filter)) {
			continue
		}
		if f.IsDir() {
			m.ChildDirs = append(m.ChildDirs, name)
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
	if pm.Searching {
		switch key := msg.String(); key {
		case "esc":
			pm.Searching = false
			pm.Filter = ""
			pm.ListChildDirs()
		case "enter":
			pm.Searching = false
			// Optionally keep filter or clear it? Cleared for now as we exit search mode vs applying selection.
			// Actually, Enter usually means "act on selection". If filtering, selection acts on filtered list.
			// Let's just exit search mode and let subsequent Enter handle action?
			// Or better: consume Enter to stop searching, user presses Enter again to Navigate.
		case "backspace":
			if len(pm.Filter) > 0 {
				pm.Filter = pm.Filter[:len(pm.Filter)-1]
				pm.ListChildDirs()
				pm.SelectedChildIndex = 0
			}
		default:
			// Basic filtering
			if len(key) == 1 {
				pm.Filter += key
				pm.ListChildDirs()
				pm.SelectedChildIndex = 0
			}
		}
		// While searching, limit navigation to arrow keys to avoid conflict with typing
		switch msg.Type {
		case tea.KeyDown:
			if len(pm.ChildDirs) > 0 {
				pm.SelectedChildIndex = 0
				return childMode, nil
			}
		}
		return pathMode, nil
	}

	switch {
	case msg.String() == "q" || msg.String() == "esc":
		return gridMode, nil // Return to grid mode (no brainer improvement)
	case msg.String() == "e":
		pm.Searching = true
		pm.Filter = ""
		pm.ListChildDirs() // Refresh logic just in case
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
	case msg.String() == ".":
		pm.ShowHidden = !pm.ShowHidden
		pm.ListChildDirs()
		// Reset child index if it became invalid (though ListChildDirs usually handles list rebuild)
		// If list became empty or shorter, we should clamp cursor
		if len(pm.ChildDirs) == 0 {
			pm.SelectedChildIndex = 0
		} else if pm.SelectedChildIndex >= len(pm.ChildDirs) {
			pm.SelectedChildIndex = len(pm.ChildDirs) - 1
		}
	}
	return pathMode, nil
}

// Update handles key events when in ChildMode
func (pm *PathModel) UpdateChildMode(msg tea.KeyMsg, cfg config.Config) (navMode, tea.Cmd) {
	if pm.Searching {
		switch key := msg.String(); key {
		case "esc":
			pm.Searching = false
			pm.Filter = ""
			pm.ListChildDirs()
			return pathMode, nil // Return to path mode to avoid accidental selection
		case "enter":
			pm.Searching = false
			// Act on selection immediately if Enter
			parentPath := pm.BuildPathFromComponents(pm.SelectedPathIndex)
			targetPath := filepath.Join(parentPath, pm.ChildDirs[pm.SelectedChildIndex])
			if err := os.Chdir(targetPath); err == nil {
				pm.CurrentPath, _ = os.Getwd()
				return gridMode, func() tea.Msg { return pathChangedMsg{} }
			}
		case "backspace":
			if len(pm.Filter) > 0 {
				pm.Filter = pm.Filter[:len(pm.Filter)-1]
				pm.ListChildDirs()
				pm.SelectedChildIndex = 0
			}
		default:
			if len(key) == 1 {
				pm.Filter += key
				pm.ListChildDirs()
				pm.SelectedChildIndex = 0
			}
		}
		// Allow navigation while searching, but STRICTLY limit to arrow keys
		switch msg.Type {
		case tea.KeyUp:
			if pm.SelectedChildIndex > 0 {
				pm.SelectedChildIndex--
			} else {
				return pathMode, nil
			}
		case tea.KeyDown:
			if pm.SelectedChildIndex < len(pm.ChildDirs)-1 {
				pm.SelectedChildIndex++
			}
		}
		return childMode, nil
	}

	switch {
	case msg.String() == "q" || msg.String() == "esc":
		return gridMode, nil // Return to grid mode
	case msg.String() == "e":
		pm.Searching = true
		pm.Filter = ""
		pm.ListChildDirs()
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
	case msg.String() == ".":
		pm.ShowHidden = !pm.ShowHidden
		pm.ListChildDirs()
		// Re-clamp cursor for child view
		if len(pm.ChildDirs) == 0 {
			pm.SelectedChildIndex = 0
		} else if pm.SelectedChildIndex >= len(pm.ChildDirs) {
			pm.SelectedChildIndex = len(pm.ChildDirs) - 1
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
	var content string

	if pm.ChildDirsError != nil {
		content = offlineStyle.Render("  [cannot read directory: permission denied or path invalid]")
	} else if len(pm.ChildDirs) == 0 {
		msg := "  [no sub-directories]"
		if pm.Filter != "" {
			msg = "  [no matches]"
		}
		content = helpStyle.Render(msg)
	} else {
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
		content = lipgloss.JoinVertical(lipgloss.Left, rows[start:end]...)
	}

	if pm.Searching {
		status := fmt.Sprintf("Search: %s_", pm.Filter)
		searchBar := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(status)
		return lipgloss.JoinVertical(lipgloss.Left, content, searchBar)
	}

	return content
}

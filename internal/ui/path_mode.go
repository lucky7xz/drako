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

	switch m.PathComponents[0] {
	case "/":
		pathToJoin = m.PathComponents[1 : index+1]
		result = "/" + filepath.Join(pathToJoin...)
	case "~":
		pathToJoin = m.PathComponents[1 : index+1]
		result = filepath.Join(home, filepath.Join(pathToJoin...))
	default:
		pathToJoin = m.PathComponents[:index+1]
		result = filepath.Join(pathToJoin...)
	}

	// Windows Drive Root Fix
	if runtime.GOOS == "windows" && len(pathToJoin) == 1 && strings.HasSuffix(result, ":") {
		return result + string(os.PathSeparator)
	}

	return result
}

// startSearch enters filter mode with an empty query.
func (pm *PathModel) startSearch() {
	pm.Searching = true
	pm.Filter = ""
	pm.ListChildDirs()
}

// clearSearch leaves filter mode and restores the unfiltered listing.
func (pm *PathModel) clearSearch() {
	pm.Searching = false
	pm.Filter = ""
	pm.ListChildDirs()
}

// editFilter applies one search keystroke — backspace or a single rune — to
// the active filter and refreshes the listing. Other keys are ignored.
func (pm *PathModel) editFilter(key string) {
	if key == "backspace" {
		if len(pm.Filter) > 0 {
			pm.Filter = pm.Filter[:len(pm.Filter)-1]
			pm.ListChildDirs()
			pm.SelectedChildIndex = 0
		}
		return
	}
	if len(key) == 1 {
		pm.Filter += key
		pm.ListChildDirs()
		pm.SelectedChildIndex = 0
	}
}

// selectedChild is the full path of the highlighted child directory. A filter
// or the hidden toggle can empty the listing under the cursor, so check ok.
func (pm *PathModel) selectedChild() (string, bool) {
	if pm.SelectedChildIndex < 0 || pm.SelectedChildIndex >= len(pm.ChildDirs) {
		return "", false
	}
	parent := pm.BuildPathFromComponents(pm.SelectedPathIndex)
	return filepath.Join(parent, pm.ChildDirs[pm.SelectedChildIndex]), true
}

// enterDir makes target the process working directory and rebuilds the
// breadcrumb and listing around it. Reading the path back with Getwd resolves
// "..", symlinks and relative components. Nothing changes on failure.
func (pm *PathModel) enterDir(target string) bool {
	if err := os.Chdir(target); err != nil {
		log.Printf("could not enter directory %s: %v", target, err)
		return false
	}
	pm.CurrentPath, _ = os.Getwd()
	pm.Searching, pm.Filter = false, "" // the filter was scoped to the directory we left
	pm.UpdatePathComponents()
	pm.ListChildDirs()
	pm.SelectedChildIndex = 0
	return true
}

// descendMode keeps the cursor on the child list so the next Enter descends
// again, or drops it to the breadcrumb when the new directory is a dead end.
func (pm *PathModel) descendMode() navMode {
	if len(pm.ChildDirs) == 0 {
		return pathMode
	}
	return childMode
}

// toggleHidden flips hidden-file visibility and clamps the child cursor to
// the resized listing.
func (pm *PathModel) toggleHidden() {
	pm.ShowHidden = !pm.ShowHidden
	pm.ListChildDirs()
	if len(pm.ChildDirs) == 0 {
		pm.SelectedChildIndex = 0
	} else if pm.SelectedChildIndex >= len(pm.ChildDirs) {
		pm.SelectedChildIndex = len(pm.ChildDirs) - 1
	}
}

// Update handles key events when in PathMode
func (pm *PathModel) UpdatePathMode(msg tea.KeyMsg, cfg config.Config) navMode {
	if pm.Searching {
		switch key := msg.String(); key {
		case "esc":
			pm.clearSearch()
		case "enter":
			// Consume Enter to leave search; the next Enter navigates.
			pm.Searching = false
		default:
			pm.editFilter(key)
		}
		// While searching, limit navigation to arrow keys to avoid conflict with typing
		switch msg.Type {
		case tea.KeyDown:
			if len(pm.ChildDirs) > 0 {
				pm.SelectedChildIndex = 0
				return childMode
			}
		}
		return pathMode
	}

	switch {
	case IsCancel(cfg.Keys, msg):
		return gridMode // Return to grid mode (no brainer improvement)
	case msg.String() == "e":
		pm.startSearch()
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
			return childMode
		}
	case IsPathGridMode(cfg.Keys, msg):
		return gridMode
	case IsConfirm(cfg.Keys, msg):
		pm.enterDir(pm.BuildPathFromComponents(pm.SelectedPathIndex))
	case msg.String() == ".":
		pm.toggleHidden()
	}
	return pathMode
}

// Update handles key events when in ChildMode
func (pm *PathModel) UpdateChildMode(msg tea.KeyMsg, cfg config.Config) navMode {
	if pm.Searching {
		switch key := msg.String(); key {
		case "esc":
			pm.clearSearch()
			return pathMode // Return to path mode to avoid accidental selection
		case "enter":
			pm.Searching = false
			// Act on selection immediately if Enter
			target, ok := pm.selectedChild()
			if !ok {
				return pathMode
			}
			if pm.enterDir(target) {
				return pm.descendMode()
			}
		default:
			pm.editFilter(key)
		}
		// Allow navigation while searching, but STRICTLY limit to arrow keys
		switch msg.Type {
		case tea.KeyUp:
			if pm.SelectedChildIndex > 0 {
				pm.SelectedChildIndex--
			} else {
				return pathMode
			}
		case tea.KeyDown:
			if pm.SelectedChildIndex < len(pm.ChildDirs)-1 {
				pm.SelectedChildIndex++
			}
		}
		return childMode
	}

	switch {
	case IsCancel(cfg.Keys, msg):
		return gridMode // Return to grid mode
	case msg.String() == "e":
		pm.startSearch()
	case IsUp(cfg.Keys, msg):
		if pm.SelectedChildIndex > 0 {
			pm.SelectedChildIndex--
		} else {
			return pathMode
		}
	case IsDown(cfg.Keys, msg):
		if pm.SelectedChildIndex < len(pm.ChildDirs)-1 {
			pm.SelectedChildIndex++
		}
	case IsPathGridMode(cfg.Keys, msg):
		return gridMode
	case IsConfirm(cfg.Keys, msg):
		target, ok := pm.selectedChild()
		if !ok {
			return pathMode
		}
		if pm.enterDir(target) {
			return pm.descendMode()
		}
	case msg.String() == ".":
		pm.toggleHidden()
	}
	return childMode
}

func (pm *PathModel) RenderPathBar(active bool, styles Styles) string {
	var renderedParts []string
	for i, component := range pm.PathComponents {
		var style lipgloss.Style
		if active && i == pm.SelectedPathIndex {
			style = styles.SelectedPath
		} else {
			style = styles.Path
		}
		renderedParts = append(renderedParts, style.Render(component))
	}

	separator := styles.PathSeparator.Render("/")
	return styles.StatusBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(renderedParts, separator)))
}

func (pm *PathModel) RenderChildDirs(mode navMode, styles Styles) string {
	if mode != childMode && mode != pathMode {
		return ""
	}
	var content string

	if pm.ChildDirsError != nil {
		content = styles.Offline.Render("  [cannot read directory: permission denied or path invalid]")
	} else if len(pm.ChildDirs) == 0 {
		msg := "  [no sub-directories]"
		if pm.Filter != "" {
			msg = "  [no matches]"
		}
		content = styles.Help.Render(msg)
	} else {
		var rows []string
		for i, dir := range pm.ChildDirs {
			if mode == childMode && i == pm.SelectedChildIndex {
				rows = append(rows, styles.SelectedChildDir.Render("› "+dir))
			} else {
				rows = append(rows, styles.ChildDir.Render("  "+dir))
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

package tui

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/glyph"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/settings"
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
	DeniedDir          string // dir the last Enter refused; cleared on the next keypress
}

func InitPathModel(startPath string) PathModel {
	m := PathModel{
		CurrentPath: startPath,
	}
	m.UpdatePathComponents()
	m.ListChildDirs()
	return m
}

// collapseHome rewrites a path under home to start with "~"; anything else,
// and an empty home, comes back unchanged. sep is a parameter so Windows stays
// testable from Linux, like launch.fallbackShell's goos.
func collapseHome(path, home string, sep rune) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(path, home+string(sep)); ok {
		return "~" + string(sep) + rest
	}
	return path
}

// splitPath breaks a path into breadcrumb components:
//
//	/home/lucky  ->  [/ home lucky]
//	C:\Users     ->  [C: Users]
func splitPath(path string, sep rune) []string {
	s := string(sep)
	if path == s {
		return []string{s}
	}
	parts := strings.Split(path, s)
	if len(parts) > 1 && parts[0] == "" {
		parts[0] = s
	}
	if n := len(parts); n > 1 && parts[n-1] == "" {
		parts = parts[:n-1]
	}
	return parts
}

func (pm *PathModel) UpdatePathComponents() {
	home := host.HomeDir()
	pm.PathComponents = splitPath(collapseHome(pm.CurrentPath, home, filepath.Separator), filepath.Separator)
	pm.SelectedPathIndex = len(pm.PathComponents) - 1
}

func (pm *PathModel) ListChildDirs() {
	pm.ChildDirs = []string{}
	pm.ChildDirsError = nil
	path := pm.BuildPathFromComponents(pm.SelectedPathIndex)

	dirs, err := host.Dirs(path)
	if err != nil {
		log.Printf("could not read directory %s: %v", path, err)
		pm.ChildDirsError = err
		return
	}

	for _, name := range dirs {
		// Basic visibility check: skip hidden files unless toggled
		if !pm.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		// Search filter check
		if pm.Filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(pm.Filter)) {
			continue
		}
		pm.ChildDirs = append(pm.ChildDirs, name)
	}
	sort.Strings(pm.ChildDirs)
}

func (pm *PathModel) BuildPathFromComponents(index int) string {
	home := host.HomeDir()
	root := string(filepath.Separator)

	if len(pm.PathComponents) == 0 {
		return pm.CurrentPath
	}

	if len(pm.PathComponents) == 1 && pm.PathComponents[0] == root {
		return root
	}

	var pathToJoin []string
	var result string

	switch pm.PathComponents[0] {
	case root:
		pathToJoin = pm.PathComponents[1 : index+1]
		result = root + filepath.Join(pathToJoin...)
	case "~":
		pathToJoin = pm.PathComponents[1 : index+1]
		result = filepath.Join(home, filepath.Join(pathToJoin...))
	default:
		pathToJoin = pm.PathComponents[:index+1]
		result = filepath.Join(pathToJoin...)
	}

	// Windows Drive Root Fix
	if runtime.GOOS == "windows" && len(pathToJoin) == 1 && strings.HasSuffix(result, ":") {
		return result + string(filepath.Separator)
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

// editFilter applies one search key event to the active filter: backspace,
// or typed/pasted runes (a fast burst arrives as one event). Others are ignored.
func (pm *PathModel) editFilter(msg tea.KeyMsg) {
	switch {
	case msg.Type == tea.KeyBackspace:
		if len(pm.Filter) == 0 {
			return
		}
		_, size := utf8.DecodeLastRuneInString(pm.Filter)
		pm.Filter = pm.Filter[:len(pm.Filter)-size]
	case (msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace) && !msg.Alt:
		pm.Filter += string(msg.Runes)
	default:
		return
	}
	pm.ListChildDirs()
	pm.SelectedChildIndex = 0
}

// selectedChild is the highlighted child's full path. A filter or the hidden
// toggle can empty the listing under the cursor, so check ok.
func (pm *PathModel) selectedChild() (string, bool) {
	if pm.SelectedChildIndex < 0 || pm.SelectedChildIndex >= len(pm.ChildDirs) {
		return "", false
	}
	parent := pm.BuildPathFromComponents(pm.SelectedPathIndex)
	return filepath.Join(parent, pm.ChildDirs[pm.SelectedChildIndex]), true
}

// enterDir chdirs to target and rebuilds the breadcrumb around it. Reading the
// path back with Getwd resolves "..", symlinks and relative components.
func (pm *PathModel) enterDir(target string) bool {
	entered, err := host.Enter(target)
	if err != nil {
		log.Printf("could not enter directory %s: %v", target, err)
		pm.DeniedDir = filepath.Base(target)
		return false
	}
	pm.CurrentPath = entered
	pm.Searching, pm.Filter = false, "" // the filter was scoped to the directory we left
	pm.UpdatePathComponents()
	pm.ListChildDirs()
	pm.SelectedChildIndex = 0
	return true
}

// descendMode keeps the cursor on the child list, or drops it to the
// breadcrumb when the new directory is a dead end.
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
func (pm *PathModel) UpdatePathMode(msg tea.KeyMsg, cfg settings.Config) navMode {
	pm.DeniedDir = "" // any keypress dismisses the last refusal
	if pm.Searching {
		switch key := msg.String(); key {
		case "esc":
			pm.clearSearch()
		case "enter":
			// Consume Enter to leave search; the next Enter navigates.
			pm.Searching = false
		default:
			pm.editFilter(msg)
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

	switch cfg.Screen(settings.PathBar).Act(msg.String()) {
	case settings.Back, settings.PathMode:
		return gridMode
	case settings.Search:
		pm.startSearch()
	case settings.MoveH:
		// Either way relists, and a filter belongs to the directory it was typed in.
		if IsLeft(cfg.Keys, msg) && pm.SelectedPathIndex > 0 {
			pm.SelectedPathIndex--
			pm.clearSearch()
		}
		if IsRight(cfg.Keys, msg) && pm.SelectedPathIndex < len(pm.PathComponents)-1 {
			pm.SelectedPathIndex++
			pm.clearSearch()
		}
	case settings.Descend:
		if len(pm.ChildDirs) > 0 {
			pm.SelectedChildIndex = 0
			return childMode
		}
	case settings.Confirm:
		pm.enterDir(pm.BuildPathFromComponents(pm.SelectedPathIndex))
	case settings.ToggleHidden:
		pm.toggleHidden()
	}
	return pathMode
}

// Update handles key events when in ChildMode
func (pm *PathModel) UpdateChildMode(msg tea.KeyMsg, cfg settings.Config) navMode {
	pm.DeniedDir = "" // any keypress dismisses the last refusal
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
			pm.editFilter(msg)
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

	switch cfg.Screen(settings.PathChildren).Act(msg.String()) {
	case settings.Back, settings.PathMode:
		return gridMode
	case settings.Search:
		pm.startSearch()
	case settings.MoveV:
		switch {
		case IsDown(cfg.Keys, msg):
			if pm.SelectedChildIndex < len(pm.ChildDirs)-1 {
				pm.SelectedChildIndex++
			}
		case pm.SelectedChildIndex > 0:
			pm.SelectedChildIndex--
		default:
			return pathMode
		}
	case settings.Confirm:
		target, ok := pm.selectedChild()
		if !ok {
			return pathMode
		}
		if pm.enterDir(target) {
			return pm.descendMode()
		}
	case settings.ToggleHidden:
		pm.toggleHidden()
	}
	return childMode
}

// pathWindow picks the run of components that fits budget, always including
// cursor and growing outward from it. Widths rather than strings: the
// components are styled, so only the caller can measure them.
func pathWindow(widths []int, sepWidth, ellipsisWidth, cursor, budget int) (int, int) {
	n := len(widths)
	if n == 0 {
		return 0, 0
	}
	cursor = max(0, min(cursor, n-1))

	// Counts the "…" markers only on the sides that actually elide.
	span := func(s, e int) int {
		w := (e - s - 1) * sepWidth
		for i := s; i < e; i++ {
			w += widths[i]
		}
		if s > 0 {
			w += ellipsisWidth + sepWidth
		}
		if e < n {
			w += ellipsisWidth + sepWidth
		}
		return w
	}

	if span(0, n) <= budget {
		return 0, n
	}

	// Always shown, even when it alone overflows; RenderPathBar truncates.
	start, end := cursor, cursor+1
	for {
		grew := false
		if end < n && span(start, end+1) <= budget {
			end++
			grew = true
		}
		if start > 0 && span(start-1, end) <= budget {
			start--
			grew = true
		}
		if !grew {
			return start, end
		}
	}
}

func (pm *PathModel) RenderPathBar(active bool, styles Styles, width int) string {
	if len(pm.PathComponents) == 0 {
		return ""
	}

	rendered := make([]string, len(pm.PathComponents))
	widths := make([]int, len(pm.PathComponents))
	for i, component := range pm.PathComponents {
		label := glyph.Label(component) // a folder name is drawn, never trusted
		rendered[i] = styles.Path.Render(label)
		if active && i == pm.SelectedPathIndex {
			// Brackets in the place of the padding: the selected folder shows
			// without colour, and no folder moves.
			rendered[i] = styles.SelectedPath.Padding(0).Render("[" + label + "]")
		}
		widths[i] = lipgloss.Width(rendered[i])
	}

	separator := styles.PathSeparator.Render(string(filepath.Separator))
	ellipsis := styles.Path.Render("…")
	start, end := pathWindow(widths, lipgloss.Width(separator), lipgloss.Width(ellipsis),
		pm.SelectedPathIndex, width)

	parts := make([]string, 0, end-start+2)
	if start > 0 {
		parts = append(parts, ellipsis)
	}
	parts = append(parts, rendered[start:end]...)
	if end < len(rendered) {
		parts = append(parts, ellipsis)
	}

	return styles.StatusBar.Render(truncateText(strings.Join(parts, separator), width))
}

// maxChildDirRows is how many folders path mode lists at once when there is
// room; the list scrolls with the selection.
const maxChildDirRows = 5

// RenderChildDirs lists the folders under the path bar, at most maxVisible of
// them, with the search line and the last refusal under it.
func (pm *PathModel) RenderChildDirs(mode navMode, styles Styles, width, maxVisible int) string {
	if mode != childMode && mode != pathMode {
		return ""
	}
	var content string

	if pm.ChildDirsError != nil {
		content = truncateText(styles.Offline.Render("  [cannot read directory: permission denied or path invalid]"), width)
	} else if len(pm.ChildDirs) == 0 {
		msg := "  [no sub-directories]"
		if pm.Filter != "" {
			msg = "  [no matches]"
		}
		content = truncateText(styles.Help.Render(msg), width)
	} else {
		var rows []string
		for i, dir := range pm.ChildDirs {
			style := styles.ChildDir
			marker := "  "
			if mode == childMode && i == pm.SelectedChildIndex {
				style, marker = styles.SelectedChildDir, "› "
			}
			rows = append(rows, truncateText(style.Render(marker+glyph.Label(dir)), width))
		}

		maxVisible = max(maxVisible, 1)
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

	// We are still in the old directory, so the refusal goes under its listing.
	if pm.DeniedDir != "" {
		refusal := fmt.Sprintf("  [cannot enter %s: permission denied or path invalid]", glyph.Label(pm.DeniedDir))
		content = lipgloss.JoinVertical(lipgloss.Left, content, truncateText(styles.Offline.Render(refusal), width))
	}

	if pm.Searching || pm.Filter != "" { // Enter ends the typing, not the filter
		status := "Search: " + pm.Filter
		if pm.Searching {
			status += "_"
		}
		searchBar := truncateText(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(status), width)
		return lipgloss.JoinVertical(lipgloss.Left, content, searchBar)
	}

	return content
}

package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/paths"
)

func (m Model) Init() tea.Cmd {
	configDir, _ := paths.ConfigDir()
	return tea.Batch(
		tea.EnterAltScreen,
		checkNetworkStatusCmd(),
		m.spinner.tick(),
		WatchConfigCmd(configDir),
		lockCheckTick(),
	)
}

func checkNetworkStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status := core.CheckNetworkStatus()
		return networkStatusMsg{online: status.Online, counters: status.Counters, t: status.Time, err: status.Err}
	}
}

func networkTick() tea.Cmd {
	return tea.Tick(2500*time.Millisecond, func(t time.Time) tea.Msg {
		return checkNetworkStatusCmd()()
	})
}

// lockCheckTick creates a command that checks for auto-lock every 30 seconds
func lockCheckTick() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return lockCheckMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil

	case pathChangedMsg:
		m.path.UpdatePathComponents()
		m.path.ListChildDirs()
		return m, nil

	case reloadProfilesMsg:
		bundle, err := config.ReloadConfig(m.profile.sessionProfile)
		if err != nil {
			log.Printf("config reload failed: %v", err)
			return m, m.setProfileStatus("Config reload failed", false)
		}
		if m.GlassrootMode && glassrootRejectsBundle(bundle) {
			return m.failGlassroot()
		}
		m.applyBundle(bundle)
		if len(bundle.Broken) > 0 {
			m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
			m.profile.errorQueueActive = true
			m = m.presentNextBrokenProfile()
			return m, nil
		}
		if bundle.DroppedProfile != "" {
			m = m.presentDroppedProfileNote(bundle.DroppedProfile)
			return m, nil
		}
		m.mode = gridMode
		return m, nil

	case ConfigChangedMsg:
		// Config file changed on disk, reload everything
		log.Printf("Config file change detected: %s", msg.Path)
		bundle, err := config.ReloadConfig(m.profile.sessionProfile)
		if err != nil {
			// Keep watching: the environment may recover before the next change.
			log.Printf("config reload failed: %v", err)
			configDir, _ := paths.ConfigDir()
			return m, tea.Batch(m.setProfileStatus("Config reload failed", false), WatchConfigCmd(configDir))
		}
		if m.GlassrootMode && glassrootRejectsBundle(bundle) {
			return m.failGlassroot()
		}
		m.applyBundle(bundle)
		if len(bundle.Broken) > 0 {
			m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
			m.profile.errorQueueActive = true
			m = m.presentNextBrokenProfile()
		} else if bundle.DroppedProfile != "" {
			m = m.presentDroppedProfileNote(bundle.DroppedProfile)
		}
		// Restart the watcher for the next change
		configDir, _ := paths.ConfigDir()
		return m, WatchConfigCmd(configDir)

	case inventoryErrorMsg:
		m.inventory.err = msg.err
		return m, nil

	case editorFinishedMsg:
		return m.afterEdit(msg)

	case spinnerTickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update()
		return m, cmd

	case tea.KeyMsg:
		key := msg.String()
		log.Printf("Key pressed: %q", key)

		// Global Emergency Exit: Ctrl+C should always quit (except in Locked Mode, handled below)
		if key == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}

		// Update last activity time for any key press (except in locked mode)
		if m.mode != lockedMode {
			m.lock.lastActivity = time.Now()
		}

		// Handle locked mode separately
		if m.mode == lockedMode {
			return m.updateLockedMode(msg)
		}

		// Locking on purpose has to work from wherever you are when you get
		// up — including a mode with a search field open, which is why this
		// sits above the text guard. It is a chord, so nothing was typing it.
		// Glassroot keeps it too: it hides the screen rather than exposing
		// anything, and the guest pumps straight back in.
		if IsSessionLock(m.Config.Keys, msg, m.Config.NumbModifier) {
			return m.enterLockedMode(), nil
		}

		// Glassroot gatekeeper: restricted keys are no-ops. The full policy
		// lives in glassroot.go.
		if m.glassrootBlocksKey(msg) {
			return m, nil
		}

		// Bindings that reach across modes are skipped while a text field is
		// collecting keystrokes, so a letter the user is typing isn't spent on
		// an action instead. Everything above this still wins: ctrl+c, the
		// locked-mode handoff, and the glassroot veto.
		if !m.capturingText() {
			// Both the lock and profile switching act on the *active* profile,
			// so they belong to the grid. Offering the lock from the inventory
			// would silently pin something other than the item under the cursor.
			if m.mode == gridMode || m.mode == childMode {
				if IsLock(m.Config.Keys, msg) {
					cmd := m.toggleProfileLock()
					return m, cmd
				}

				// Profile switching with configurable modifier + Number or ~ (Shift + `)
				if ok, target := IsProfileSwitch(m.Config.Keys, msg, m.Config.NumbModifier); ok {
					if target < len(m.profile.profiles) {
						if updated, ok := m.switchToProfileIndex(target); ok {
							m = updated
							return m, nil
						}
					}
					return m, nil
				}
				if IsProfilePrev(m.Config.Keys, msg) {
					return m.handleProfileCycle(-1)
				}
				if IsProfileNext(m.Config.Keys, msg) {
					return m.handleProfileCycle(1)
				}
			}
		}
		switch m.mode {
		case gridMode:
			return m.updateGridMode(msg)
		case pathMode:
			mode, cmd := m.path.UpdatePathMode(msg, m.Config)
			m.mode = mode
			return m, cmd
		case childMode:
			mode, cmd := m.path.UpdateChildMode(msg, m.Config)
			m.mode = mode
			return m, cmd
		case inventoryMode:
			return m.updateInventoryMode(msg)
		case dropdownMode:
			return m.updateDropdownMode(msg)
		case infoMode:
			return m.updateInfoMode(msg)
		case batchMode:
			return m.updateBatchMode(msg)
		}

	case networkStatusMsg:
		if msg.err != nil {
			m.net.traffic = m.styles.ThemeName.Render("error")
		} else {
			m.net.meter.Sample(msg.counters.BytesSent, msg.counters.BytesRecv, msg.t)

			isActive := false
			sentBps, recvBps, ok := m.net.meter.Rates()
			switch {
			case m.net.meter.Samples() <= 1:
				m.net.traffic = m.styles.ThemeName.Render("calculating...")
			case !ok:
				m.net.traffic = m.styles.ThemeName.Render("---")
			default:
				m.net.traffic = m.styles.ThemeName.Render(fmt.Sprintf("↓ %s ↑ %s", core.FormatTraffic(recvBps), core.FormatTraffic(sentBps)))
				if sentBps > 2*1024 || recvBps > 2*1024 {
					isActive = true
				}
			}

			if msg.online {
				if isActive {
					m.net.online = m.styles.Online.Render("online (active)")
				} else {
					m.net.online = m.styles.Online.Render("online (idle)")
				}
			} else {
				m.net.online = m.styles.Offline.Render("offline")
			}
		}
		return m, networkTick()

	case navTimeoutMsg:
		if m.gridNav.timer != nil {
			m.gridNav.timer.Stop()
		}
		m.gridNav.timer = nil
		return m, nil

	case profileStatusClearMsg:
		if msg.id != m.profile.statusClearTimerID {
			return m, nil
		}
		m.profile.statusClearTimerID = 0
		m.profile.statusMessage = ""
		return m, nil

	case leaderTimeoutMsg:
		m.disarmLeader()
		return m, nil

	case lockCheckMsg:
		// Check if we should auto-lock
		autoLockEnabled := m.Config.AutoLockEnabled == nil || *m.Config.AutoLockEnabled
		if autoLockEnabled && m.mode != lockedMode && m.lock.timeoutMins > 0 {
			elapsed := time.Since(m.lock.lastActivity)
			if elapsed >= time.Duration(m.lock.timeoutMins)*time.Minute {
				log.Printf("Auto-locking after %v of inactivity", elapsed)
				m = m.enterLockedMode()
			}
		}
		return m, lockCheckTick()

	}

	return m, nil
}

func (m Model) updateDropdownMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Leader sequence: inside a dropdown the only continuation is 'b'
	// (start marking this folder's items); everything else is swallowed.
	if m.leader.pending {
		m.disarmLeader()
		if key == "b" {
			return m.enterDropdownBatch()
		}
		return m, nil
	}
	if IsLeader(m.Config.Keys, msg) {
		return m.armLeader()
	}

	// An active dropdown batch owns space/enter/esc; other keys fall through
	// to the normal dropdown behavior below (digit jumps, nav, explain).
	if m.batch.dropdown {
		if next, cmd, handled := m.updateDropdownMarking(msg); handled {
			return next, cmd
		}
	}

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		targetIndex := num - 1 // Convert to 0-based index
		if len(m.dropdown.items) > 0 {
			// If the target is out of bounds, select the last item.
			if targetIndex >= len(m.dropdown.items) {
				m.dropdown.selectedIdx = len(m.dropdown.items) - 1
			} else {
				m.dropdown.selectedIdx = targetIndex
			}
		}
		return m, nil
	}

	switch {
	case IsCancel(m.Config.Keys, msg):
		// Close dropdown and return to grid mode
		m.mode = gridMode
		m.dropdown.items = nil
		return m, nil
	case Matches(m.Config.Keys, msg, "ctrl+c"):
		m.Quitting = true
		return m, tea.Quit
	case IsUp(m.Config.Keys, msg):
		if m.dropdown.selectedIdx > 0 {
			m.dropdown.selectedIdx--
		}
	case IsDown(m.Config.Keys, msg):
		if m.dropdown.selectedIdx < len(m.dropdown.items)-1 {
			m.dropdown.selectedIdx++
		}
	case IsExplain(m.Config.Keys, msg):
		if m.dropdown.selectedIdx >= 0 && m.dropdown.selectedIdx < len(m.dropdown.items) {
			item := m.dropdown.items[m.dropdown.selectedIdx]
			parent := ""
			if m.dropdown.row >= 0 && m.dropdown.col >= 0 && m.dropdown.row < len(m.gridNav.grid) && m.dropdown.col < len(m.gridNav.grid[0]) {
				parent = m.gridNav.grid[m.dropdown.row][m.dropdown.col]
			}
			m.previousMode = m.mode

			title := item.Name
			if strings.TrimSpace(parent) != "" {
				title = fmt.Sprintf("%s: %s", parent, item.Name)
			}

			cmdStr := item.Command
			if strings.TrimSpace(item.Command) == "" {
				cmdStr = "Error: no command configured"
			}

			m.activeDetail = newCommandDetail(title, cmdStr, item.Description, item.AutoCloseExecution, item.DebugExecution, m.path.CurrentPath)
			m.mode = infoMode
			return m, nil
		}
		return m, nil
	case IsConfirm(m.Config.Keys, msg):
		// Execute the selected dropdown item
		if m.dropdown.selectedIdx >= 0 && m.dropdown.selectedIdx < len(m.dropdown.items) {
			selectedItem := m.dropdown.items[m.dropdown.selectedIdx]
			m.Selected = selectedItem.Name
			// Store the command to execute
			// We need to create a temporary command entry for execution
			return m, tea.Quit
		}
	}
	return m, nil
}

// copyToClipboardCmd copies text to clipboard using the best available method
func copyToClipboardCmd(s string) tea.Cmd {
	return func() tea.Msg {
		CopyToClipboard(s)
		return nil
	}
}

func (m Model) updateInfoMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.profile.errorQueueActive {
		var cmds []tea.Cmd
		if key == "y" && m.allowCopy() && m.activeDetail != nil {
			cmds = append(cmds, copyToClipboardCmd(m.activeDetail.Value))
		}
		// Always delegate to presentNextBrokenProfile to handle queue exhaustion and Rescue Trigger
		m = m.presentNextBrokenProfile()
		return m, tea.Batch(cmds...)
	}
	switch key {
	case "pgup":
		if m.activeDetail != nil {
			_, viewportH, _ := m.infoScrollMetrics()
			m.activeDetail.ScrollOffset = max(0, m.activeDetail.ScrollOffset-(viewportH-1))
		}
		return m, nil
	case "pgdown":
		if m.activeDetail != nil {
			_, viewportH, maxOffset := m.infoScrollMetrics()
			m.activeDetail.ScrollOffset = min(maxOffset, m.activeDetail.ScrollOffset+(viewportH-1))
		}
		return m, nil
	case "y":
		if !m.allowCopy() {
			return m, nil
		}
		if m.activeDetail != nil {
			prev := m.previousMode
			m.mode = prev
			cmd := copyToClipboardCmd(m.activeDetail.Value)
			m.activeDetail = nil // Clear detail state
			return m, cmd
		}
		m.mode = m.previousMode
		m.activeDetail = nil
		return m, nil
	default:
		m.mode = m.previousMode
		m.activeDetail = nil // Clear detail state
		return m, nil
	}
}

// afterEdit reacts to the external editor closing: reload everything, let
// the broken-profile queue take over if the edit broke an equipped profile,
// otherwise return to the inventory view with a validity verdict.
func (m Model) afterEdit(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.inventory.status = "Edit failed: " + msg.err.Error()
		m.mode = inventoryMode
		return m, nil
	}

	// The edit may have changed any equipped profile — reload the bundle.
	bundle, err := config.ReloadConfig(m.profile.sessionProfile)
	if err != nil {
		m.inventory.status = "Reload failed: " + err.Error()
		m.mode = inventoryMode
		return m, nil
	}
	m.applyBundle(bundle)
	if len(bundle.Broken) > 0 {
		if m.GlassrootMode {
			return m.failGlassroot()
		}
		m.profile.pendingErrors = append(m.profile.pendingErrors, bundle.Broken...)
		m.profile.errorQueueActive = true
		return m.presentNextBrokenProfile(), nil
	}

	// Rebuild the inventory (validity may have changed), keep the cursor.
	prev := m.inventory
	m.inventory = InitInventoryModel(m.profile.configDir)
	m.inventory.focusedList = prev.focusedList
	m.inventory.cursor = prev.cursor
	if listPtr, err := m.inventory.State.GetList(prev.focusedList); err == nil && m.inventory.cursor >= len(*listPtr) {
		m.inventory.cursor = max(len(*listPtr)-1, 0)
	}

	name := filepath.Base(msg.path)
	if err := config.CheckProfileFile(msg.path); err != nil {
		m.inventory.status = fmt.Sprintf("⚠️ %s has errors: %v", name, err)
	} else {
		m.inventory.status = "✓ Saved: " + name
	}
	m.mode = inventoryMode
	return m, nil
}

// pump advances the unlock slider for one keypress: alternating directions
// fill it, repeating a direction drains it. Reports whether the goal was hit.
func (l *lockState) pump(dir int) bool {
	if l.lastDirection == dir {
		if l.progress > 0 {
			l.progress--
		}
		return false
	}
	l.lastDirection = dir
	if l.progress < l.pumpGoal {
		l.progress++
	}
	return l.progress >= l.pumpGoal
}

func (m Model) updateLockedMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Allow Ctrl+C to quit even when locked
	if key == "ctrl+c" {
		m.Quitting = true
		return m, tea.Quit
	}

	dir := core.PumpDirectionForKey(key)
	if dir == 0 {
		return m, nil
	}

	if m.lock.pump(dir) {
		log.Printf("Unlocking via pump sequence after %d steps", m.lock.progress)
		m = m.exitLockedMode()
	}
	return m, nil
}

// capturingText reports whether a mode is collecting typed characters right
// now — the delete confirmation, or a path search filter. Such a field owns
// the keyboard: profile names and directory names can contain any character,
// so no binding may claim one out from under it.
func (m Model) capturingText() bool {
	switch m.mode {
	case inventoryMode:
		return m.inventory.pending != nil
	case pathMode, childMode:
		return m.path.Searching
	}
	return false
}

func (m Model) switchToProfileIndex(target int) (Model, bool) {
	if len(m.profile.profiles) == 0 {
		return m, false
	}
	total := len(m.profile.profiles)
	if target < 0 || target >= total {
		target = ((target % total) + total) % total
	}

	selected := m.profile.profiles[target]

	// Check for existence
	if _, err := os.Stat(selected.Path); err != nil {
		log.Printf("skipping missing profile: %s", selected.Path)
		return m, false
	}

	updated := m
	updated.profile.activeIndex = target
	updated.profile.sessionProfile = selected.Name
	updated.Config = config.ApplyProfileOverlay(m.profile.base, selected.Profile)
	updated.applyConfig(updated.Config)

	return updated, true
}

func (m Model) handleProfileCycle(direction int) (tea.Model, tea.Cmd) {
	if len(m.profile.profiles) <= 1 {
		return m, nil
	}

	current := m.profile.activeIndex
	total := len(m.profile.profiles)
	// Only try up to 'total' times.
	// Start from 1 to avoid re-selecting the current profile immediately.
	for i := 1; i <= total; i++ {
		target := core.CalculateNextProfileIndex(current, direction*i, total)

		nextModel, ok := m.switchToProfileIndex(target)
		if ok {
			return nextModel, nil
		}
	}
	return m, nil
}

package tui

import (
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/settings"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		checkNetworkStatusCmd(),
		m.spinner.tick(),
		lockCheckTick(),
	)
}

func checkNetworkStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status := host.CheckNetworkStatus()
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

// Update handles a message, then gives any queued broken/dropped-profile
// notice a chance to surface — the single point where a protected mode
// (locked, or mid-capturing text) ending reconnects with data that arrived
// while it was up. See presentDeferredIfSafe.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.handleMsg(msg)
	nm, ok := next.(Model)
	if !ok {
		return next, cmd
	}
	return nm.presentDeferredIfSafe(), cmd
}

func (m Model) handleMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil

	case reloadProfilesMsg:
		next, cmd, applied := m.reloadAndApply()
		// From the inventory this is Apply's reload, and Apply means done. A
		// watcher reload (ConfigChangedMsg) leaves the inventory open instead.
		if applied && next.mode == inventoryMode {
			next.mode = gridMode
		}
		return next, cmd

	case ConfigChangedMsg:
		// Config file changed on disk, reload everything
		log.Printf("Config file change detected: %s", msg.Path)
		wasInventory := m.mode == inventoryMode
		next, cmd, applied := m.reloadAndApply()
		if applied && wasInventory {
			next = next.refreshInventory()
		}
		return next, cmd

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
		act := m.act(msg)

		// Fixed order, each step can fully short-circuit the ones below it:
		// 1. ctrl+c always quits, even locked.
		// 2. touch the idle timer (skipped while already locked).
		// 3. locked mode owns every other key.
		// 4. the session-lock chord works everywhere — mid-search, in glassroot.
		// 5. the key sheet: open, it owns every key; the help key opens it.
		// 6. glassroot vetoes its restricted keys before anything below acts.
		// 7. the deck keys (pin, switch), on the screens that have them.
		// 8. the mode-specific handler.
		// What each key means is the key table's to say (settings/keymap.go);
		// this order is the table's order too.

		// Global Emergency Exit: Ctrl+C should always quit (except in Locked Mode, handled below)
		if act == settings.Quit {
			m.quitting = true
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
		if act == settings.LockSession {
			return m.enterLockedMode(), nil
		}

		// The sheet floats over the screen it lists, so while it is open the
		// screen underneath gets no keys at all.
		if m.sheet != nil {
			return m.updateKeySheet(msg), nil
		}
		if act == settings.Help {
			m.sheet = &sheetState{}
			return m, nil
		}

		// Glassroot gatekeeper: restricted keys are no-ops. The full policy
		// lives in glassroot.go.
		if m.glassrootBlocksKey(msg) {
			return m, nil
		}

		if next, cmd, handled := m.handleDeckKeys(msg); handled {
			return next, cmd
		}

		switch m.mode {
		case gridMode:
			return m.updateGridMode(msg)
		case pathMode:
			m.mode = m.path.UpdatePathMode(msg, m.Config)
			return m, nil
		case childMode:
			m.mode = m.path.UpdateChildMode(msg, m.Config)
			return m, nil
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
				m.net.traffic = m.styles.ThemeName.Render(fmt.Sprintf("↓ %s ↑ %s", host.FormatTraffic(recvBps), host.FormatTraffic(sentBps)))
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
		if msg.seq == m.gridNav.navSeq {
			m.gridNav.navPending = false
		}
		return m, nil

	case rowFlashTimeoutMsg:
		if msg.seq == m.gridNav.flashSeq {
			m.gridNav.flashActive = false
		}
		return m, nil

	case profileStatusClearMsg:
		if msg.id != m.profile.statusClearTimerID {
			return m, nil
		}
		m.profile.statusClearTimerID = 0
		m.profile.statusMessage = ""
		return m, nil

	case leaderTimeoutMsg:
		if msg.seq == m.leader.seq {
			m.disarmLeader()
		}
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

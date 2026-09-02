package multiplex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// A pane's shell is still starting when the pane appears, and a shell that is
// not yet reading discards what was typed at it. How long that takes depends on
// what the login profile does — a quiet one is instant, one that prints a MOTD
// is not — so a send is confirmed and repeated rather than assumed.
//
// A landed send is visible almost at once, since exec replaces the shell in
// place; the window only costs time when the send was lost. Short window, more
// attempts, so a slow-starting shell recovers sooner.
const (
	herdrSendAttempts = 5
	herdrConfirmFor   = time.Second
	herdrConfirmEvery = 50 * time.Millisecond
)

// herdrSleep is a test seam so the retry loop costs nothing in tests.
var herdrSleep = time.Sleep

// Herdr launches a batch through herdr. Nothing that creates a pane there can
// also run something in it — not in the CLI and not in the socket API — so each
// pane is created, the id herdr's server minted is read from the reply, and the
// script is then sent to that id. Every pane in a tab exists before anything
// runs, so a split only ever carves up an idle shell.
//
// herdr is only used from inside herdr (it has no detached-create primitive),
// so a launch never hands over the terminal.
type Herdr struct{}

func NewHerdr() *Herdr { return &Herdr{} }

func (h *Herdr) Name() string { return herdrMux }

// session and env go unused: herdr has no named sessions to target, and a
// cell's environment is already baked into its script rather than passed to the
// multiplexer.
func (h *Herdr) Launch(session string, cmds []Command, tabs []int, paths, env []string) (bool, error) {
	cell := 0
	for t, panes := range tabs {
		// The batch is worth looking at, but only once: the rest of the tabs
		// are built behind the first.
		focus := "--no-focus"
		if t == 0 {
			focus = "--focus"
		}
		root, err := h.create("tab", "create", "--label", tabLabel(cmds[cell:cell+panes]), focus)
		if err != nil {
			return false, err
		}

		ids := make([]string, panes)
		ids[0] = root
		for i, s := range splitTree(panes) {
			id, err := h.create("pane", "split", "--pane", ids[s.parent], "--direction", s.dir, "--no-focus")
			if err != nil {
				return false, err
			}
			ids[i+1] = id
		}

		for i, id := range ids {
			if err := runCell(id, paths[cell+i]); err != nil {
				return false, err
			}
		}
		cell += panes
	}
	return false, nil
}

// runCell starts one cell in its pane and confirms it actually started.
//
// exec replaces the pane's shell, so the pane's process becomes the script and
// the pane closes when the cell finishes — as it does under tmux, and as
// auto_close_execution promises. It also gives us the acknowledgement: once the
// cell is running, the script is the pane's foreground process. Until it is,
// the send may have been typed at a shell that was not yet listening.
func runCell(id, path string) error {
	for range herdrSendAttempts {
		if _, err := herdrExec([]string{"herdr", "pane", "run", id, "exec", path}); err != nil {
			return err
		}
		started, err := confirmCell(id, path)
		if err != nil {
			return err
		}
		if started {
			return nil
		}
	}
	return fmt.Errorf("%s did not start in pane %s", path, id)
}

// confirmCell waits for path to become the pane's foreground process. A pane
// that has gone counts as started: the cell ran and finished quickly enough to
// close it.
func confirmCell(id, path string) (bool, error) {
	for waited := time.Duration(0); waited < herdrConfirmFor; waited += herdrConfirmEvery {
		out, err := herdrExec([]string{"herdr", "pane", "process-info", "--pane", id})
		if err != nil {
			return true, nil
		}
		running, err := foregroundHas(out, path)
		if err != nil {
			return false, err
		}
		if running {
			return true, nil
		}
		herdrSleep(herdrConfirmEvery)
	}
	return false, nil
}

// foregroundHas reports whether path is among the pane's foreground processes.
func foregroundHas(out []byte, path string) (bool, error) {
	var reply struct {
		Result struct {
			ProcessInfo struct {
				ForegroundProcesses []struct {
					Cmdline string `json:"cmdline"`
				} `json:"foreground_processes"`
			} `json:"process_info"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &reply); err != nil {
		return false, fmt.Errorf("could not read herdr's reply: %w", err)
	}
	for _, p := range reply.Result.ProcessInfo.ForegroundProcesses {
		if strings.Contains(p.Cmdline, path) {
			return true, nil
		}
	}
	return false, nil
}

// create runs a herdr call that mints a pane and returns its id.
func (h *Herdr) create(args ...string) (string, error) {
	out, err := herdrExec(append([]string{"herdr"}, args...))
	if err != nil {
		return "", err
	}
	return paneID(out)
}

// herdrExec runs one herdr invocation and returns its stdout. Test seam, like
// tmux's execStep. herdr reports failures as JSON on stderr with exit 1, so
// that is what a caller wants to see.
var herdrExec = func(argv []string) ([]byte, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %v: %s", strings.Join(argv[1:], " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// paneID reads the pane a create call minted: tab create names its root pane,
// pane split names the new one.
func paneID(out []byte) (string, error) {
	var reply struct {
		Result struct {
			Pane struct {
				PaneID string `json:"pane_id"`
			} `json:"pane"`
			RootPane struct {
				PaneID string `json:"pane_id"`
			} `json:"root_pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &reply); err != nil {
		return "", fmt.Errorf("could not read herdr's reply: %w", err)
	}
	if id := reply.Result.Pane.PaneID; id != "" {
		return id, nil
	}
	if id := reply.Result.RootPane.PaneID; id != "" {
		return id, nil
	}
	return "", fmt.Errorf("herdr's reply named no pane: %s", out)
}

// split creates one pane by splitting an existing one. Split i creates pane
// i+1 within its tab, so parent is always a pane that already exists.
type split struct {
	parent int    // index of the pane to split, within the tab
	dir    string // "right" or "down", as herdr's --direction takes them
}

// paneTree is every split a full tab needs, in order. herdr has no equivalent
// of tmux's `select-layout tiled`, so the geometry is drako's to choose: split
// the root right, then each column down, which leaves the panes in reading
// order — ① ② across the top, ③ ④ beneath. Each smaller tab is a prefix of
// this, so adding a pane never rearranges the ones already placed.
var paneTree = []split{{0, "right"}, {0, "down"}, {1, "down"}}

// splitTree is the splits that lay n panes out in one tab. Pane 0 is the tab's
// root and already exists, so a tab of n panes takes n-1 splits. n is capped at
// PanesPerTab, which checkTabs guarantees.
func splitTree(n int) []split {
	if n < 2 {
		return nil
	}
	return paneTree[:n-1]
}

package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

func brokenErr(name string) config.ProfileParseError {
	return config.ProfileParseError{
		Name: name,
		Path: "/tmp/" + name + ".profile.toml",
		Err:  "SYNTAX ERROR: boom",
	}
}

func queueModel(errs ...config.ProfileParseError) Model {
	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()
	return Model{
		mode:   gridMode,
		styles: BuildStyles(base),
		profile: profileState{
			pendingErrors:    errs,
			errorQueueActive: true,
			acknowledged:     map[string]bool{},
		},
	}
}

func TestBrokenQueuePresentsFirstError(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))

	m = m.presentNextBrokenProfile()

	if m.mode != infoMode {
		t.Fatalf("mode = %v, want infoMode", m.mode)
	}
	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "alpha") {
		t.Fatalf("detail should present alpha, got %+v", m.activeDetail)
	}
	if len(m.profile.pendingErrors) != 1 {
		t.Errorf("queue should have consumed one entry, %d left", len(m.profile.pendingErrors))
	}
	if !m.profile.acknowledged["/tmp/alpha.profile.toml"] {
		t.Error("presented error must be marked acknowledged")
	}
}

func TestBrokenQueueSkipsAcknowledged(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))
	m.profile.acknowledged["/tmp/alpha.profile.toml"] = true

	m = m.presentNextBrokenProfile()

	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "beta") {
		t.Fatalf("already-acknowledged alpha must be skipped, got %+v", m.activeDetail)
	}
}

func TestBrokenQueueExhaustionEntersRescue(t *testing.T) {
	m := queueModel() // active queue, nothing pending

	m = m.presentNextBrokenProfile()

	if m.mode != gridMode {
		t.Fatalf("mode = %v, want gridMode after exhaustion", m.mode)
	}
	if m.profile.errorQueueActive {
		t.Error("queue must deactivate once exhausted")
	}
	if len(m.Config.Commands) == 0 {
		t.Error("exhausting an active queue applies the rescue config (helper commands)")
	}
}

func TestBrokenQueueAckWalksToRescue(t *testing.T) {
	m := queueModel(brokenErr("alpha"), brokenErr("beta"))
	m = m.presentNextBrokenProfile()

	ack := tea.KeyMsg{Type: tea.KeyEnter}
	next, _ := m.updateInfoMode(ack)
	m = next.(Model)
	if m.activeDetail == nil || !strings.Contains(m.activeDetail.Title, "beta") {
		t.Fatalf("first ack should present beta, got %+v", m.activeDetail)
	}

	next, _ = m.updateInfoMode(ack)
	m = next.(Model)
	if m.mode != gridMode || m.profile.errorQueueActive {
		t.Fatalf("second ack should exhaust the queue into grid mode, mode=%v active=%v", m.mode, m.profile.errorQueueActive)
	}
}

// Exiting rescue mode reloads the config; with a broken config.toml the
// bundle still reports that error, but it was acknowledged on the way in. A
// drain that shows nothing must keep the config the caller just applied
// instead of bouncing straight back into rescue.
func TestBrokenQueueAllAcknowledgedKeepsReloadedConfig(t *testing.T) {
	m := queueModel(brokenErr("alpha"))
	m.profile.acknowledged["/tmp/alpha.profile.toml"] = true

	reloaded := config.Config{X: 1, Y: 1}
	reloaded.ApplyDefaults()
	reloaded.Commands = []config.Command{{Name: "my cell", Command: "true", Row: 0, Col: "a"}}
	m.applyConfig(reloaded)

	m = m.presentNextBrokenProfile()

	if len(m.Config.Commands) != 1 || m.Config.Commands[0].Name != "my cell" {
		t.Fatalf("nothing was shown, so the reloaded config must survive; got %+v", m.Config.Commands)
	}
	if m.profile.errorQueueActive {
		t.Error("queue must deactivate once exhausted")
	}
}

func TestApplyBundleAdoptsBundleState(t *testing.T) {
	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()
	effective := base
	effective.Commands = []config.Command{{Name: "noop", Command: "true", Row: 0, Col: "a"}}

	var m Model
	m.applyBundle(config.ConfigBundle{
		Base:        base,
		Config:      effective,
		Profiles:    []config.ProfileInfo{{Name: "alpha"}, {Name: "work"}},
		ActiveIndex: 1,
		ConfigDir:   "/tmp/cfg",
		LockedName:  "work",
	})

	if m.ActiveProfileName() != "work" {
		t.Errorf("active profile = %q, want work", m.ActiveProfileName())
	}
	if !m.profile.locked || m.profile.pivotName != "work" {
		t.Errorf("pivot lock not adopted: locked=%v name=%q", m.profile.locked, m.profile.pivotName)
	}
	if len(m.Config.Commands) != 1 {
		t.Errorf("effective config not applied: %+v", m.Config.Commands)
	}
}

func TestApplyBundleClampsBadActiveIndex(t *testing.T) {
	base := config.Config{X: 1, Y: 1}
	base.ApplyDefaults()

	var m Model
	m.applyBundle(config.ConfigBundle{
		Base:        base,
		Config:      base,
		Profiles:    []config.ProfileInfo{{Name: "only"}},
		ActiveIndex: 7,
	})

	if m.profile.activeIndex != 0 {
		t.Errorf("out-of-range index must clamp to 0, got %d", m.profile.activeIndex)
	}
}

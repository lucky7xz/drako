package ui

import "github.com/lucky7xz/drako/internal/launch"

// Setup is what the host tells the TUI when it creates it. The TUI looks none
// of this up for itself.
type Setup struct {
	ConfigDir string   // drako's config directory
	Targets   []string // the platform's variant keys, most specific first (host.Targets)
	Profile   string   // the profile asked for from outside (DRAKO_PROFILE); "" for none
	Glassroot bool     // serve the sealed surface
}

// Action is what the TUI asks its host to do once it is off the screen.
type Action int

const (
	Resume Action = iota // nothing asked: show the TUI again
	Quit                 // end drako with ExitCode
	Run                  // run Plans[0] in the terminal
	Batch                // launch Plans together, split into Tabs
)

// Outcome is the TUI's whole message to its host: one action and what that
// action needs. The host never looks inside the model for anything else.
type Outcome struct {
	Action    Action
	ExitCode  int           // Quit
	Message   string        // Quit: why, for the host to print; usually empty
	Plans     []launch.Plan // Run: exactly one. Batch: one per mark, runnable or not
	Tabs      []int         // Batch: the layout chosen in the dialog; nil for the default
	ForceTmux bool          // Batch: the batch_force_tmux setting
}

// Outcome reads the finished model. Quitting outranks a batch, which outranks
// a single run.
func (m Model) Outcome() Outcome {
	profile := m.ActiveProfileName()
	switch {
	case m.quitting:
		return Outcome{Action: Quit, ExitCode: m.exitCode, Message: m.exitMessage}
	case len(m.selectedBatch) > 0:
		plans := make([]launch.Plan, len(m.selectedBatch))
		for i, sel := range m.selectedBatch {
			plans[i] = launch.For(m.Config.DefaultShell, m.Config.Commands, sel, profile)
		}
		return Outcome{Action: Batch, Plans: plans, Tabs: m.selectedTabs, ForceTmux: m.Config.BatchForceTmux}
	case m.selected.Cell != "":
		return Outcome{Action: Run, Plans: []launch.Plan{launch.For(m.Config.DefaultShell, m.Config.Commands, m.selected, profile)}}
	}
	return Outcome{}
}

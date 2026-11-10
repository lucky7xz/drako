// Package launch decides what runs for a selection: the command line, its
// environment and how the run ends. It only decides; starting the process is
// the caller's job.
package launch

import (
	"strings"

	"github.com/lucky7xz/drako/internal/config"
)

// Plan is everything needed to run one entry. Making one does nothing.
type Plan struct {
	Name      string   // the entry's name as shown on screen
	Argv      []string // the command line; empty when there is nothing to run
	Profile   string   // the active profile, exported as DRAKO_PROFILE
	AutoClose bool     // after success, return to drako without a keypress
	Debug     bool     // capture the output and show it once the command ends
}

// Runnable reports whether the plan has a command to run. A cell with no
// command for this platform, or a selection the config no longer has, has none.
func (p Plan) Runnable() bool { return len(p.Argv) > 0 }

// For plans the run of sel as cfg defines it: a cell runs with its own
// command and flags, an item with the item's.
func For(cfg config.Config, sel Selection, profile string) Plan {
	name := sel.Name(cfg)
	cell, item, found := sel.Resolve(cfg)
	if !found {
		return Plan{Name: name, Profile: profile}
	}
	command, autoClose, debug := cell.Command, cell.AutoCloseExecution, cell.DebugExecution
	if item != nil {
		command, autoClose, debug = item.Command, item.AutoCloseExecution, item.DebugExecution
	}
	plan := Shell(cfg.DefaultShell, name, command, profile)
	plan.AutoClose = boolOrDefault(autoClose, true)
	plan.Debug = boolOrDefault(debug, false)
	return plan
}

// Shell plans running command through shell, with a cell's default flags. A
// blank command leaves the plan with nothing to run.
func Shell(shell, name, command, profile string) Plan {
	plan := Plan{Name: name, Profile: profile, AutoClose: true}
	if strings.TrimSpace(command) != "" {
		plan.Argv = ShellArgv(shell, command)
	}
	return plan
}

// boolOrDefault resolves an optional config flag. Config keeps these as *bool
// so that "not set" (nil) can be told apart from false.
func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

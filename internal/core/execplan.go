package core

import (
	"os/exec"

	"github.com/lucky7xz/drako/internal/config"
)

// planKind says what RunCommand should do with a selection.
type planKind int

const (
	// planShell runs commandStr through the configured shell.
	planShell planKind = iota
	// planDirect runs path as a bare binary (no shell) — the PATH fallback
	// for selections that match no configured cell.
	planDirect
	// planNoCommand is a configured cell with no command string (e.g. no
	// variant for this platform); inform the user, never fall through to PATH.
	planNoCommand
	// planNotFound matches nothing: no cell, no binary in PATH.
	planNotFound
)

// execPlan is the resolved decision for one selection: what to run, how, and
// with which flags. Building it is pure — no execution, no process state.
type execPlan struct {
	kind       planKind
	shell      string
	commandStr string
	path       string
	autoClose  bool
	debug      bool
}

// buildExecPlan resolves sel: a cell or item runs via the shell with its own flags,
// one without a command is a dead end, and an unmatched bare cell name tries PATH.
func buildExecPlan(cfg config.Config, sel Selection, lookPath func(string) (string, error)) execPlan {
	plan := execPlan{
		kind:      planNotFound,
		shell:     cfg.DefaultShell,
		autoClose: true,
	}

	parentCmd, itemCfg, found := sel.Resolve(cfg)
	if found {
		commandStr := parentCmd.Command
		autoClosePtr := parentCmd.AutoCloseExecution
		debugPtr := parentCmd.DebugExecution
		if itemCfg != nil {
			commandStr = itemCfg.Command
			autoClosePtr = itemCfg.AutoCloseExecution
			debugPtr = itemCfg.DebugExecution
		}
		if commandStr == "" {
			plan.kind = planNoCommand
			return plan
		}
		plan.kind = planShell
		plan.commandStr = commandStr
		plan.autoClose = boolOrDefault(autoClosePtr, true)
		plan.debug = boolOrDefault(debugPtr, false)
		return plan
	}

	if sel.Item != 0 {
		return plan // a stale item must never run its folder's name as a binary
	}
	if path, err := lookPath(sel.Cell); err == nil {
		plan.kind = planDirect
		plan.path = path
		return plan
	}
	return plan
}

// assembleCmd turns a runnable plan into an exec.Cmd with the prepared child
// environment attached — the same env for every run mode, debug included.
// Non-runnable plans (noCommand/notFound) return nil.
func assembleCmd(plan execPlan, env []string) *exec.Cmd {
	var cmd *exec.Cmd
	switch plan.kind {
	case planShell:
		cmd = buildShellCmd(plan.shell, plan.commandStr)
	case planDirect:
		cmd = commandFn(plan.path)
	default:
		return nil
	}
	cmd.Env = env
	return cmd
}

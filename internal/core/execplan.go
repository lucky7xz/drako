package core

import (
	"os/exec"
	"strings"

	"github.com/lucky7xz/drako/internal/config"
)

// planKind says what RunCommand should do with a selection.
type planKind int

const (
	// planShell runs commandStr through the configured shell.
	planShell planKind = iota
	// planNoCommand has nothing to run: a cell with no command (e.g. no
	// variant for this platform) or a selection the config no longer has.
	planNoCommand
)

// execPlan is the resolved decision for one selection: what to run, how, and
// with which flags. Building it is pure — no execution, no process state.
type execPlan struct {
	kind       planKind
	shell      string
	commandStr string
	autoClose  bool
	debug      bool
}

// buildExecPlan resolves sel: a cell or item runs via the shell with its own
// flags; a blank command, or a selection the config no longer has, is a dead end.
func buildExecPlan(cfg config.Config, sel Selection) execPlan {
	parentCmd, itemCfg, found := sel.Resolve(cfg)
	if !found {
		return execPlan{kind: planNoCommand}
	}
	commandStr := parentCmd.Command
	autoClosePtr := parentCmd.AutoCloseExecution
	debugPtr := parentCmd.DebugExecution
	if itemCfg != nil {
		commandStr = itemCfg.Command
		autoClosePtr = itemCfg.AutoCloseExecution
		debugPtr = itemCfg.DebugExecution
	}
	if strings.TrimSpace(commandStr) == "" {
		return execPlan{kind: planNoCommand}
	}
	return execPlan{
		kind:       planShell,
		shell:      cfg.DefaultShell,
		commandStr: commandStr,
		autoClose:  boolOrDefault(autoClosePtr, true),
		debug:      boolOrDefault(debugPtr, false),
	}
}

// assembleCmd turns a planShell plan into an exec.Cmd with the prepared child
// environment attached — the same env for every run mode, debug included.
func assembleCmd(plan execPlan, env []string) *exec.Cmd {
	cmd := buildShellCmd(plan.shell, plan.commandStr)
	cmd.Env = env
	return cmd
}

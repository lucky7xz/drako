package core

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func boolPtr(b bool) *bool { return &b }

func planTestConfig() config.Config {
	return config.Config{
		DefaultShell: "bash",
		Commands: []config.Command{
			{Name: "hello", Command: "echo hi"},
			{
				Name:               "careful",
				Command:            "sudo reboot",
				AutoCloseExecution: boolPtr(false),
				DebugExecution:     boolPtr(true),
			},
			{Name: "empty-cell"},
			{
				Name: "folder",
				Items: []config.CommandItem{
					{Name: "child", Command: "ls -la", AutoCloseExecution: boolPtr(false)},
					{Name: "bare-child"},
				},
			},
		},
	}
}

func TestBuildExecPlan_TopLevelCommand(t *testing.T) {
	plan := buildExecPlan(planTestConfig(), Selection{Cell: "hello"})
	if plan.kind != planShell {
		t.Fatalf("kind = %v, want planShell", plan.kind)
	}
	if plan.commandStr != "echo hi" || plan.shell != "bash" {
		t.Errorf("plan = %+v, want echo hi via bash", plan)
	}
	if !plan.autoClose || plan.debug {
		t.Errorf("defaults: autoClose=%v debug=%v, want true/false", plan.autoClose, plan.debug)
	}
}

func TestBuildExecPlan_FlagOverrides(t *testing.T) {
	plan := buildExecPlan(planTestConfig(), Selection{Cell: "careful"})
	if plan.autoClose || !plan.debug {
		t.Errorf("overrides: autoClose=%v debug=%v, want false/true", plan.autoClose, plan.debug)
	}
}

func TestBuildExecPlan_DropdownItem(t *testing.T) {
	plan := buildExecPlan(planTestConfig(), Selection{Cell: "folder", Item: 1})
	if plan.kind != planShell || plan.commandStr != "ls -la" {
		t.Fatalf("plan = %+v, want ls -la via shell", plan)
	}
	if plan.autoClose {
		t.Error("item-level autoClose override not honored")
	}
}

func TestBuildExecPlan_ConfiguredButEmpty(t *testing.T) {
	for _, sel := range []Selection{{Cell: "empty-cell"}, {Cell: "folder", Item: 2}} {
		plan := buildExecPlan(planTestConfig(), sel)
		if plan.kind != planNoCommand {
			t.Errorf("%+v: kind = %v, want planNoCommand ", sel, plan.kind)
		}
	}
}

func TestAssembleCmd_ShellPlanCarriesEnv(t *testing.T) {
	plan := execPlan{kind: planShell, shell: "sh", commandStr: "echo hi"}
	env := []string{"PATH=/usr/bin", "DRAKO_PROFILE=work"}
	cmd := assembleCmd(plan, env)
	if cmd == nil {
		t.Fatal("shell plan must produce a command")
	}
	if len(cmd.Env) != 2 || cmd.Env[1] != "DRAKO_PROFILE=work" {
		t.Errorf("the command env must be attached before any run mode, got %v", cmd.Env)
	}
}

func TestBuildExecPlan_NotFound(t *testing.T) {
	// A cell the config no longer has, or an item past the folder's end.
	for _, sel := range []Selection{{Cell: "ghost"}, {Cell: "folder", Item: 9}} {
		if plan := buildExecPlan(planTestConfig(), sel); plan.kind != planNoCommand {
			t.Errorf("%+v: kind = %v, want planNoCommand", sel, plan.kind)
		}
	}
}

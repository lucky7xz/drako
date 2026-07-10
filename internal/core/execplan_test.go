package core

import (
	"errors"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func boolPtr(b bool) *bool { return &b }

func noLookPath(string) (string, error) { return "", errors.New("not found") }

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
	plan := buildExecPlan(planTestConfig(), "hello", noLookPath)
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
	plan := buildExecPlan(planTestConfig(), "careful", noLookPath)
	if plan.autoClose || !plan.debug {
		t.Errorf("overrides: autoClose=%v debug=%v, want false/true", plan.autoClose, plan.debug)
	}
}

func TestBuildExecPlan_DropdownItem(t *testing.T) {
	plan := buildExecPlan(planTestConfig(), "child", noLookPath)
	if plan.kind != planShell || plan.commandStr != "ls -la" {
		t.Fatalf("plan = %+v, want ls -la via shell", plan)
	}
	if plan.autoClose {
		t.Error("item-level autoClose override not honored")
	}
}

func TestBuildExecPlan_ConfiguredButEmpty(t *testing.T) {
	for _, name := range []string{"empty-cell", "bare-child"} {
		plan := buildExecPlan(planTestConfig(), name, noLookPath)
		if plan.kind != planNoCommand {
			t.Errorf("%s: kind = %v, want planNoCommand (never fall through to PATH)", name, plan.kind)
		}
	}
}

func TestBuildExecPlan_PathFallback(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "htop" {
			return "/usr/bin/htop", nil
		}
		return "", errors.New("not found")
	}
	plan := buildExecPlan(planTestConfig(), "htop", lookPath)
	if plan.kind != planDirect || plan.path != "/usr/bin/htop" {
		t.Fatalf("plan = %+v, want direct /usr/bin/htop", plan)
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
		t.Errorf("sanitized env must be attached before any run mode, got %v", cmd.Env)
	}
}

func TestAssembleCmd_DirectPlanCarriesEnv(t *testing.T) {
	plan := execPlan{kind: planDirect, path: "/bin/true"}
	cmd := assembleCmd(plan, []string{"A=1"})
	if cmd == nil || cmd.Path != "/bin/true" {
		t.Fatalf("direct plan should exec the resolved path, got %+v", cmd)
	}
	if len(cmd.Env) != 1 || cmd.Env[0] != "A=1" {
		t.Errorf("sanitized env must be attached, got %v", cmd.Env)
	}
}

func TestBuildExecPlan_NotFound(t *testing.T) {
	plan := buildExecPlan(planTestConfig(), "ghost", noLookPath)
	if plan.kind != planNotFound {
		t.Fatalf("kind = %v, want planNotFound", plan.kind)
	}
}

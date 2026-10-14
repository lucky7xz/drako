package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestAgentsBriefNamesTheProtocol(t *testing.T) {
	for _, want := range []string{
		"drako ls", "drako explain <addr>", "drako run <addr>",
		"danger", "work:B2", "B2.1",
	} {
		if !strings.Contains(agentsBrief, want) {
			t.Errorf("brief does not mention %q", want)
		}
	}
}

// The brief promises an exit code; it must be the one drako run returns.
func TestAgentsBriefMatchesRunExitCode(t *testing.T) {
	if !strings.Contains(agentsBrief, fmt.Sprintf("Exit %d", exitRunFailed)) {
		t.Errorf("brief does not document exit %d", exitRunFailed)
	}
}

// Static means no config at all: it works on a machine drako never ran on,
// and never bootstraps files as a side effect.
func TestAgentsTouchesNoConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	if code := HandleAgentsCommand([]string{"drako", "agents"}); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("drako agents created files: %v", entries)
	}
}

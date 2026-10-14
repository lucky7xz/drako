package cli

// drako agents prints a brief for AI agents and scripts: how to drive the
// deck through ls, explain and run. Static, so it can live in AGENTS.md.

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed agents_brief.md
var agentsBrief string

// HandleAgentsCommand processes 'drako agents'. Returns the process exit code.
func HandleAgentsCommand(args []string) int {
	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "Usage: drako agents\n")
		return 1
	}
	fmt.Print(agentsBrief)
	return 0
}

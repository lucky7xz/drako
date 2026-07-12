package config

import (
	"strings"
	"testing"
)

// The core profile's windows variants run under powershell.exe: cmd-style
// %VAR% expansion and retired subcommands must not creep back in.
func TestCoreProfileWindowsVariants(t *testing.T) {
	raw, err := bootstrapFS.ReadFile("bootstrap/core.profile.toml")
	if err != nil {
		t.Fatalf("embedded core profile missing: %v", err)
	}
	for _, banned := range []string{"%APPDATA%", "drako internal"} {
		if strings.Contains(string(raw), banned) {
			t.Errorf("core.profile.toml contains %q, which does not work under powershell", banned)
		}
	}
}

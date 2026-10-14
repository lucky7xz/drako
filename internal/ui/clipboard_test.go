package ui

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

// The copied text must reach PowerShell only as inert base64, never as
// script source it would expand or execute.
func TestPowerShellClipboardArgsKeepTextInert(t *testing.T) {
	payload := "echo \"$(Remove-Item ~\\x)\"; calc; `n 'q' $HOME Übung"
	args := powerShellClipboardArgs(payload)
	script := args[len(args)-1]

	if strings.Contains(script, "Remove-Item") || strings.Contains(script, "$HOME") {
		t.Fatalf("payload leaked into the script: %s", script)
	}
	m := regexp.MustCompile(`FromBase64String\('([A-Za-z0-9+/=]*)'\)`).FindStringSubmatch(script)
	if m == nil {
		t.Fatalf("no base64 literal in script: %s", script)
	}
	got, err := base64.StdEncoding.DecodeString(m[1])
	if err != nil || string(got) != payload {
		t.Errorf("round trip = %q, %v; want %q", got, err, payload)
	}
}

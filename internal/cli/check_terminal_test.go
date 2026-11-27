package cli

import (
	"strings"
	"testing"
)

func TestCheckTerminalNamesTheEnvironment(t *testing.T) {
	t.Setenv("TERM", "xterm-test")
	t.Setenv("COLORTERM", "")
	var code int
	out := captureStdout(t, func() { code = HandleCheckTerminalCommand(nil) })
	if code != 0 || !strings.Contains(out, "TERM=xterm-test") || !strings.Contains(out, "COLORTERM=(not set)") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

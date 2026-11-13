package launch

import (
	"slices"
	"strings"
	"testing"
)

func TestShellArgv(t *testing.T) {
	tests := []struct {
		shell string
		want  []string
	}{
		{"bash", []string{"bash", "-lc", "echo hi"}},
		{"sh", []string{"sh", "-c", "echo hi"}},
		{"zsh", []string{"zsh", "-lc", "echo hi"}},
		{"fish", []string{"fish", "-c", "echo hi"}},
		{"pwsh", []string{"pwsh", "-NoLogo", "-NoProfile", "-Command", "echo hi"}},
		{"powershell", []string{"powershell", "-NoLogo", "-NoProfile", "-Command", "echo hi"}},
		{"powershell.exe", []string{"powershell", "-NoLogo", "-NoProfile", "-Command", "echo hi"}},
		{"cmd", []string{"cmd", "/C", "echo hi"}},
	}
	for _, tc := range tests {
		t.Run(tc.shell, func(t *testing.T) {
			if got := ShellArgv(tc.shell, "echo hi"); !slices.Equal(got, tc.want) {
				t.Fatalf("argv=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestFallbackShell(t *testing.T) {
	if got := fallbackShell("windows"); got != "powershell" {
		t.Fatalf("windows fallback=%q, want powershell", got)
	}
	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		if got := fallbackShell(goos); got != "bash" {
			t.Fatalf("%s fallback=%q, want bash", goos, got)
		}
	}
}

// One list: every shell check accepts is one ShellArgv runs as itself.
func TestKnownShellsRunAsThemselves(t *testing.T) {
	for _, name := range ShellNames() {
		argv := ShellArgv(name, "echo hi")
		if !KnownShell(name) || argv[len(argv)-1] != "echo hi" || !strings.HasPrefix(name, argv[0]) {
			t.Errorf("%s: argv = %q", name, argv)
		}
	}
	if KnownShell("zhs") {
		t.Error("zhs is a typo, not a shell")
	}
}

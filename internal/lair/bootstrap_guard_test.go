package lair

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/profile"
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

// Install common tools says by its exit status whether everything is ready:
// 0 when every tool installed, non-zero when one is missing. Its summary used
// to end on `[ -n "$skipped" ] && …`, which made a full success exit 1.
func TestInstallCommonToolsExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs the Linux variants under bash")
	}
	raw, err := bootstrapFS.ReadFile("bootstrap/core.profile.toml")
	if err != nil {
		t.Fatalf("embedded core profile missing: %v", err)
	}
	script := func(target string) string {
		pf, _, err := profile.Decode(raw, []string{target})
		if err != nil {
			t.Fatalf("%s: %v", target, err)
		}
		for _, c := range pf.Commands {
			for _, it := range c.Items {
				if it.Name == "Install common tools" {
					return it.Command
				}
			}
		}
		t.Fatalf("%s: no Install common tools item", target)
		return ""
	}
	// run gives the script a PATH of stubs only: sudo runs its arguments, and
	// the package manager (pacman, or nix for the NixOS variant) fails for
	// the tool named in failTool and succeeds otherwise.
	run := func(target, manager, failTool string) (int, string) {
		dir := t.TempDir()
		stub := func(name, body string) {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		stub("sudo", `exec "$@"`)
		stub(manager, `case "$*" in *`+failTool+`*) exit 1;; esac; exit 0`)
		cmd := exec.Command("/bin/bash", "-c", script(target))
		cmd.Env = []string{"PATH=" + dir, "HOME=" + t.TempDir()}
		out, _ := cmd.CombinedOutput()
		return cmd.ProcessState.ExitCode(), string(out)
	}

	for _, c := range []struct{ target, manager string }{
		{"linux_generic", "pacman"},
		{"linux_nixos", "nix"},
	} {
		if code, out := run(c.target, c.manager, "no-such-tool"); code != 0 {
			t.Errorf("%s, everything installs: exit %d, want 0\n%s", c.target, code, out)
		}
		if code, out := run(c.target, c.manager, "duf"); code == 0 || !strings.Contains(out, "Missing: duf") {
			t.Errorf("%s, duf fails: exit %d, want non-zero and a Missing line\n%s", c.target, code, out)
		}
	}
}

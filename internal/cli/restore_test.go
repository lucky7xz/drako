package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateConfigDir points the config dir at a fresh temp dir so restore
// commands never touch the real environment. Returns the drako config dir.
func isolateConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	return filepath.Join(tmp, "drako")
}

func TestHandleRestoreBootstrapCommand(t *testing.T) {
	dir := isolateConfigDir(t)

	// Fresh dir: restores the full set, exit 0.
	if code := HandleRestoreBootstrapCommand(); code != 0 {
		t.Fatalf("first restore exit = %d, want 0", code)
	}
	for _, f := range []string{"core.profile.toml", "inventory/ssh-101.profile.toml", "config.toml"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s not restored: %v", f, err)
		}
	}

	// Everything present: still exit 0, nothing to do.
	if code := HandleRestoreBootstrapCommand(); code != 0 {
		t.Errorf("idempotent restore exit = %d, want 0", code)
	}
}

func TestRestoreDispatch_NewNameAndLegacyAlias(t *testing.T) {
	for _, verb := range []string{"restore-bootstrap", "restore-core"} {
		t.Run(verb, func(t *testing.T) {
			isolateConfigDir(t)
			handled, code := HandleCLI([]string{"drako", verb})
			if !handled || code != 0 {
				t.Errorf("HandleCLI(%q) = (%v, %d), want (true, 0)", verb, handled, code)
			}
		})
	}
}

// A command that works in drako's folder does not start on a linked
// subfolder: one message, nothing restored, nothing written through the link.
// A command that does not need the folder still runs.
func TestHandleCLIRefusesALinkedSubfolder(t *testing.T) {
	dir := isolateConfigDir(t)
	outside := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "inventory")); err != nil {
		t.Fatal(err)
	}
	var code int
	stderr := captureStderr(t, func() { _, code = HandleCLI([]string{"drako", "restore-bootstrap"}) })
	if code != 1 || !strings.HasPrefix(stderr, "inventory is a linked folder") {
		t.Errorf("exit = %d, stderr = %q; want 1 and the layout message", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err == nil {
		t.Error("restored files although the layout is unsupported")
	}
	if left, _ := os.ReadDir(outside); len(left) != 0 {
		t.Errorf("wrote through the link: %v", left)
	}
	if _, code := HandleCLI([]string{"drako", "version"}); code != 0 {
		t.Errorf("version exit = %d, want 0", code)
	}

	// check reads the folder only when given no paths.
	stderr = captureStderr(t, func() { _, code = HandleCLI([]string{"drako", "check"}) })
	if code != 1 || !strings.HasPrefix(stderr, "inventory is a linked folder") {
		t.Errorf("check without paths: exit = %d, stderr = %q; want the layout message", code, stderr)
	}
	deck := filepath.Join(t.TempDir(), "a.profile.toml")
	if err := os.WriteFile(deck, []byte(validDeck), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr = captureStderr(t, func() { _, code = HandleCLI([]string{"drako", "check", deck}) })
	if strings.Contains(stderr, "linked folder") {
		t.Errorf("check with a path met the layout check: %q", stderr)
	}
}

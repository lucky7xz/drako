package cli

import (
	"os"
	"path/filepath"
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

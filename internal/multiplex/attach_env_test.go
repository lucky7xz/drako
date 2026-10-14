package multiplex

import (
	"os"
	"path/filepath"
	"testing"
)

// The attach client must find the server the setup steps started, so both
// run in the same environment; a cell's own env is baked into its script.
func TestExecStep_AttachSeesTheSetupEnvironment(t *testing.T) {
	t.Setenv("TMUX_TMPDIR", "/tmp/drako-sockets")
	dir := t.TempDir()
	probe := func(attach bool, out string) string {
		argv := []string{"sh", "-c", `printf %s "$TMUX_TMPDIR" > "` + filepath.Join(dir, out) + `"`}
		if err := execStep(argv, attach); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(filepath.Join(dir, out))
		return string(b)
	}
	setup, attach := probe(false, "setup"), probe(true, "attach")
	if attach != setup {
		t.Errorf("setup steps see TMUX_TMPDIR=%q but attach sees %q", setup, attach)
	}
}

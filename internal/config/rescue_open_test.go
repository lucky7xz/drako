package config

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lucky7xz/drako/internal/paths"
)

// The rescue cells that open the config must hand drako open one argument,
// whatever the path holds (macOS: "Application Support"; names like O'Brien).
func TestRescueOpenCellsPassThePathWhole(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("parses the command with bash, the rescue shell off Windows")
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "Application Support", "o'brien"))
	dir, err := paths.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Edit Config": paths.ConfigFile(dir), "Open Config Dir": dir}

	for _, c := range RescueConfig().Commands {
		path, ok := want[c.Name]
		if !ok {
			continue
		}
		out, err := exec.Command("bash", "-c", `drako() { printf '%s\n' "$@"; }; `+c.Command).CombinedOutput()
		if got := string(out); err != nil || got != "open\n"+path+"\n" {
			t.Errorf("%s: drako received %q (%v), want open + %q", c.Name, got, err, path)
		}
		delete(want, c.Name)
	}
	if len(want) > 0 {
		t.Fatalf("rescue cells not found: %v", want)
	}
}

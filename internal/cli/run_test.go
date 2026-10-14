package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func runBundle() config.ConfigBundle {
	return config.ConfigBundle{
		ActiveIndex: 0,
		Profiles: []config.ProfileInfo{
			{Name: "core", Profile: config.ProfileFile{X: 3, Y: 3, Commands: []config.Command{
				{Name: "Hello", Command: "echo hi", Col: "a", Row: 0},
				{Name: "🚨 Nuke", Command: "rm -rf /tmp/x", Col: "a", Row: 1, Danger: true},
				{Name: "Mixed", Col: "a", Row: 2, Items: []config.CommandItem{
					{Name: "Calm", Command: "echo calm"},
					{Name: "🚨 Wipe", Command: "echo wipe", Danger: true},
					{Name: "Ghost"},
				}},
				{Name: "🚨 Danger Folder", Col: "b", Row: 0, Danger: true, Items: []config.CommandItem{
					{Name: "Looks Calm", Command: "echo inherited"},
				}},
				{Name: "NoVariant", Col: "b", Row: 1},
				{Name: "Both", Command: "echo own", Col: "b", Row: 2, Items: []config.CommandItem{
					{Name: "Child", Command: "echo child"},
				}},
			}}},
			{Name: "work", Profile: config.ProfileFile{X: 1, Y: 1, Commands: []config.Command{
				{Name: "Deploy", Command: "make deploy", Col: "a", Row: 0},
			}}},
		},
	}
}

func TestResolveRunTargets(t *testing.T) {
	tests := []struct {
		ref, wantName, wantCmd, wantProfile string
	}{
		{"A0", "Hello", "echo hi", "core"},
		{"a0", "Hello", "echo hi", "core"},
		{"A2.1", "Calm", "echo calm", "core"},
		{"B2", "Both", "echo own", "core"},
		{"B2.1", "Child", "echo child", "core"},
		{"work:A0", "Deploy", "make deploy", "work"},
		{"WORK:a0", "Deploy", "make deploy", "work"},
	}
	for _, tt := range tests {
		target, _, err := resolveRun(runBundle(), tt.ref)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tt.ref, err)
			continue
		}
		name, cmd, p := target.name(), target.command(), target.p
		if name != tt.wantName || cmd != tt.wantCmd || p.Name != tt.wantProfile {
			t.Errorf("%s = (%q, %q, %s), want (%q, %q, %s)",
				tt.ref, name, cmd, p.Name, tt.wantName, tt.wantCmd, tt.wantProfile)
		}
	}
}

func TestResolveRunRefusals(t *testing.T) {
	tests := []struct{ ref, wantErr string }{
		{"A1", "A1 (🚨 Nuke) is marked danger"},
		{"A2.2", "A2.2 (🚨 Wipe) is marked danger"},
		{"B0.1", "B0.1 (Looks Calm) is marked danger"}, // a danger folder covers its items
		{"A2", "is a folder; address one of its items (e.g. A2.1)"},
		{"B0", "is a folder"},
		{"A2.3", "A2.3 (Ghost) has no command"},
		{"B1", "B1 (NoVariant) has no command"},
		{"A2.4", "no item 4 in cell A2 (3 items)"},
		{"A0.1", "no item 1 in cell A0 (0 items)"},
		{"C2", "no cell C2 in profile core"},
		{"nope:A0", `no equipped profile "nope"`},
		{"0A", "bad cell address"},
		{"A", "bad cell address"},
		{"A1.x", "bad cell address"},
		{"", "bad cell address"},
	}
	for _, tt := range tests {
		_, _, err := resolveRun(runBundle(), tt.ref)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%q: err = %v, want it to contain %q", tt.ref, err, tt.wantErr)
		}
	}
}

func TestResolveRunDangerBeatsEmptyCommand(t *testing.T) {
	b := runBundle()
	b.Profiles[0].Profile.Commands[1].Command = ""
	_, _, err := resolveRun(b, "A1")
	if err == nil || !strings.Contains(err.Error(), "marked danger") {
		t.Errorf("danger must be reported first, got %v", err)
	}
}

func TestResolveRunMatchesTheGrid(t *testing.T) {
	b := runBundle()
	core := &b.Profiles[0].Profile
	core.Commands = append(core.Commands,
		config.Command{Name: "Shadow", Command: "echo shadow", Col: "a", Row: 0},
		config.Command{Name: "Last", Command: "echo last", Col: "c", Row: -1},
	)
	for ref, want := range map[string]string{"A0": "Shadow", "c-1": "Last"} {
		target, _, err := resolveRun(b, ref)
		if err != nil || target.name() != want {
			t.Errorf("%s = (%q, %v), want %q: the cell the grid shows", ref, target.name(), err, want)
		}
	}
}

func TestResolveRunRefusesInvalidGrid(t *testing.T) {
	b := runBundle()
	work := &b.Profiles[1].Profile
	work.Commands = append(work.Commands, config.Command{Name: "Off grid", Command: "true", Col: "c", Row: 0})
	if _, _, err := resolveRun(b, "work:A0"); err == nil || !strings.Contains(err.Error(), "invalid grid") {
		t.Errorf("work:A0 err = %v, want an invalid-grid refusal", err)
	}
	// Broken profile 0 still sits at ActiveIndex 0 while the TUI shows rescue.
	b.Profiles[0], b.Profiles[1] = b.Profiles[1], b.Profiles[0]
	b.ActiveIndex = 0
	if _, _, err := resolveRun(b, "A0"); err == nil || !strings.Contains(err.Error(), "invalid grid") {
		t.Errorf("bare A0 err = %v, want an invalid-grid refusal", err)
	}
}

// runFixture writes a real config dir with one profile whose cells leave
// sentinel files behind, so a test can prove whether a command really ran.
func runFixture(t *testing.T) (dir string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("DRAKO_PROFILE", "")
	dir = filepath.Join(tmp, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("config.toml", "theme = \"default\"\ndefault_shell = \"sh\"\n")
	write("solo.profile.toml", fmt.Sprintf(`x = 2
y = 3
[[commands]]
name = "safe"
command = "echo $DRAKO_PROFILE > %[1]s/safe.ran"
col = "a"
row = 0
[[commands]]
name = "🚨 nuke"
command = "touch %[1]s/nuke.ran"
col = "a"
row = 1
danger = true
[[commands]]
name = "seven"
command = "exit 7"
col = "a"
row = 2
[[commands]]
name = "folder"
col = "b"
row = 0
items = [
    { name = "calm", command = "touch %[1]s/calm.ran" },
    { name = "🚨 wipe", command = "touch %[1]s/wipe.ran", danger = true },
]
[[commands]]
name = "🚨 danger folder"
col = "b"
row = 1
danger = true
items = [ { name = "child", command = "touch %[1]s/child.ran" } ]
`, dir))
	return dir
}

func ran(dir, sentinel string) bool {
	_, err := os.Stat(filepath.Join(dir, sentinel))
	return err == nil
}

func TestRunExecutesSafeCells(t *testing.T) {
	dir := runFixture(t)
	if code := HandleRunCommand([]string{"drako", "run", "solo:A0"}); code != 0 {
		t.Fatalf("solo:A0 exit = %d, want 0", code)
	}
	b, err := os.ReadFile(filepath.Join(dir, "safe.ran"))
	if err != nil {
		t.Fatal("safe cell did not run")
	}
	if strings.TrimSpace(string(b)) != "solo" {
		t.Errorf("DRAKO_PROFILE in child = %q, want solo", b)
	}

	if code := HandleRunCommand([]string{"drako", "run", "solo:B0.1"}); code != 0 || !ran(dir, "calm.ran") {
		t.Errorf("solo:B0.1 exit = %d, ran = %v; want 0, true", code, ran(dir, "calm.ran"))
	}
}

func TestRunPassesExitCodeThrough(t *testing.T) {
	runFixture(t)
	if code := HandleRunCommand([]string{"drako", "run", "solo:A2"}); code != 7 {
		t.Errorf("exit = %d, want the child's 7", code)
	}
}

func TestRunRefusesDanger(t *testing.T) {
	dir := runFixture(t)
	for ref, sentinel := range map[string]string{
		"solo:A1":   "nuke.ran",  // danger cell
		"solo:B0.2": "wipe.ran",  // danger item
		"solo:B1.1": "child.ran", // item of a danger folder
	} {
		if code := HandleRunCommand([]string{"drako", "run", ref}); code != 125 {
			t.Errorf("%s exit = %d, want 125", ref, code)
		}
		if ran(dir, sentinel) {
			t.Errorf("%s ran despite danger", ref)
		}
	}
}

func TestRunWritesHistory(t *testing.T) {
	dir := runFixture(t)
	HandleRunCommand([]string{"drako", "run", "solo:A0"})
	b, err := os.ReadFile(filepath.Join(dir, "history.log"))
	if err != nil || !strings.Contains(string(b), "safe") {
		t.Errorf("history.log missing the run: %q, %v", b, err)
	}
}

func TestRunUsage(t *testing.T) {
	runFixture(t)
	for _, args := range [][]string{
		{"drako", "run"},
		{"drako", "run", "solo:A0", "solo:A2"},
	} {
		var code int
		stderr := captureStderr(t, func() { code = HandleRunCommand(args) })
		if code != 125 || !strings.HasPrefix(stderr, "drako run: ") {
			t.Errorf("%v = (%d, %q), want 125 and a drako run: line", args, code, stderr)
		}
	}
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	f()
	os.Stderr = orig
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

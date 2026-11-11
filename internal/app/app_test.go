package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/profile"
)

func TestCwdFileArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"absent", []string{"drako"}, ""},
		{"equals form", []string{"drako", "--cwd-file=/tmp/x"}, "/tmp/x"},
		{"separate form", []string{"drako", "--cwd-file", "/tmp/x"}, "/tmp/x"},
		{"alongside glassroot", []string{"drako", "--glassroot", "--cwd-file=/tmp/x"}, "/tmp/x"},
		{"empty value is absent", []string{"drako", "--cwd-file="}, ""},
		// A trailing bare flag has nothing to consume; treat it as absent
		// rather than reading past the end of the arguments.
		{"trailing bare flag", []string{"drako", "--cwd-file"}, ""},
		// The wrapper quotes "$tmp", so a path with spaces arrives as one arg.
		{"path with spaces", []string{"drako", "--cwd-file", "/tmp/a b"}, "/tmp/a b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cwdFileArg(tt.args); got != tt.want {
				t.Errorf("cwdFileArg(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestWriteCwdFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	// Getwd resolves symlinks (macOS /var -> /private/var), so compare against
	// what the process actually reports rather than against dir.
	want, _ := os.Getwd()

	out := filepath.Join(orig, "cwd.out")
	t.Cleanup(func() { _ = os.Remove(out) })
	writeCwdFile(out)

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("wrote %q, want %q", got, want)
	}
}

// The shell owns the file; drako refusing to exit over it would be worse than
// a missing cd.
func TestWriteCwdFile_UnwritablePathIsSurvivable(t *testing.T) {
	writeCwdFile(filepath.Join(t.TempDir(), "no-such-dir", "cwd.out"))
}

// A batch pane stays open after success exactly when a single run would pause:
// auto_close_execution = false or debug_execution = true, item flags for items.
func TestBatchCommands_KeepOpen(t *testing.T) {
	yes, no := true, false
	cfg := config.Config{Commands: []profile.Command{
		{Name: "plain", Command: "ls"},
		{Name: "stay", Command: "ls", AutoCloseExecution: &no},
		{Name: "debug", Command: "ls", DebugExecution: &yes},
		{Name: "folder", Command: "ls", DebugExecution: &yes, Items: []profile.CommandItem{
			{Name: "quiet", Command: "ls"},
			{Name: "loud", Command: "ls", DebugExecution: &yes},
		}},
	}}
	sels := []launch.Selection{{Cell: "plain"}, {Cell: "stay"}, {Cell: "debug"}, {Cell: "folder", Item: 1}, {Cell: "folder", Item: 2}}
	cmds := batchCommands(plansFor(cfg, sels))
	if len(cmds) != len(sels) {
		t.Fatalf("got %d cells for %d marks", len(cmds), len(sels))
	}
	for i, want := range []bool{false, true, true, false, true} {
		if cmds[i].KeepOpen != want {
			t.Errorf("%s: KeepOpen = %v, want %v", cmds[i].Name, cmds[i].KeepOpen, want)
		}
	}
}

// A marked entry with nothing to run is left out by the same rule a single run
// uses: gone from the config, no command, or a command of only spaces.
func TestBatchCommands_SkipsNothingToRun(t *testing.T) {
	cfg := config.Config{Commands: []profile.Command{
		{Name: "real", Command: "ls"},
		{Name: "empty"},
		{Name: "blank", Command: "   "},
	}}
	sels := []launch.Selection{{Cell: "empty"}, {Cell: "real"}, {Cell: "blank"}, {Cell: "gone"}}
	cmds := batchCommands(plansFor(cfg, sels))
	if len(cmds) != 1 || cmds[0].Name != "real" {
		t.Fatalf("got %+v, want only the cell with a command", cmds)
	}
}

func plansFor(cfg config.Config, sels []launch.Selection) []launch.Plan {
	plans := make([]launch.Plan, len(sels))
	for i, sel := range sels {
		plans[i] = launch.For(cfg.DefaultShell, cfg.Commands, sel, "work")
	}
	return plans
}

// A plan with nothing to run says so and waits for a key. That it starts
// nothing is proc's promise (TestNothingToRunIsRefused).
func TestRunPlan_NothingToRunPauses(t *testing.T) {
	old := pauseFn
	defer func() { pauseFn = old }()

	cfg := config.Config{Commands: []profile.Command{{Name: "test", Command: ""}, {Name: "blank", Command: "   "}}}
	for _, sel := range []launch.Selection{{Cell: "test"}, {Cell: "blank"}, {Cell: "gone"}} {
		paused := false
		pauseFn = func(string) { paused = true }
		runPlan(nil, launch.For(cfg.DefaultShell, cfg.Commands, sel, "")) // nothing to run, so nothing is logged
		if !paused {
			t.Errorf("%s: expected a pause", sel.Cell)
		}
	}
}

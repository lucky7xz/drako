package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
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
	cfg := config.Config{Commands: []config.Command{
		{Name: "plain", Command: "ls"},
		{Name: "stay", Command: "ls", AutoCloseExecution: &no},
		{Name: "debug", Command: "ls", DebugExecution: &yes},
		{Name: "folder", Command: "ls", DebugExecution: &yes, Items: []config.CommandItem{
			{Name: "quiet", Command: "ls"},
			{Name: "loud", Command: "ls", DebugExecution: &yes},
		}},
	}}
	sels := []core.Selection{{Cell: "plain"}, {Cell: "stay"}, {Cell: "debug"}, {Cell: "folder", Item: 1}, {Cell: "folder", Item: 2}}
	cmds := batchCommands(cfg, sels, "work")
	if len(cmds) != len(sels) {
		t.Fatalf("got %d cells for %d marks", len(cmds), len(sels))
	}
	for i, want := range []bool{false, true, true, false, true} {
		if cmds[i].KeepOpen != want {
			t.Errorf("%s: KeepOpen = %v, want %v", cmds[i].Name, cmds[i].KeepOpen, want)
		}
	}
}

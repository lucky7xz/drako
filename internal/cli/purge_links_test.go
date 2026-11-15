package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/lair"
)

// linkedLair is a config folder where config.toml, history.log and
// work.profile.toml are links into a dotfiles folder, next to plain files.
func linkedLair(t *testing.T) (dir, dots string) {
	t.Helper()
	base := t.TempDir()
	dir, dots = filepath.Join(base, "drako"), filepath.Join(base, "dots")
	for _, d := range []string{dir, dots} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"config.toml", "history.log", "work.profile.toml"} {
		if err := os.WriteFile(filepath.Join(dots, name), []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join("..", "dots", name), filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"plain.profile.toml", "drako.log"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, dots
}

// The prompt says what is a link and that the file it points to stays.
func TestPurgePromptNamesLinks(t *testing.T) {
	dir, dots := linkedLair(t)
	tests := []struct {
		name string
		opts PurgeOptions
		want string
	}{
		{"config", PurgeOptions{TargetConfig: true}, "This will:\n" +
			"   ⚠️  reset your Core Configuration\n" +
			"       config.toml is a link to " + dots + "/config.toml.\n" +
			"       The link goes to trash and a fresh config.toml takes its place.\n" +
			"       The linked file stays as it is; drako stops reading it.\n" +
			"Proceed?"},
		{"logs", PurgeOptions{TargetLogs: true}, "This will:\n" +
			"   💀 PERMANENTLY DELETE your log files (history.log, drako.log)\n" +
			"       history.log is a link: only the link is removed, " + dots + "/history.log stays\n" +
			"Proceed?"},
		{"profiles", PurgeOptions{TargetProfiles: []string{"work.profile.toml", "plain.profile.toml"}}, "This will:\n" +
			"   ⚠️  move 2 profile(s) to trash: work.profile.toml, plain.profile.toml\n" +
			"       work.profile.toml is a link: the link goes to trash, " + dots + "/work.profile.toml stays\n" +
			"Proceed?"},
		{"everything", PurgeOptions{DestroyEverything: true}, "💀 This will DESTROY EVERYTHING in " + dir + ".\n" +
			"   NO UNDO. NO TRASH.\n" +
			"   3 files in it are links: those lose only the link, the files they point to stay.\n" +
			"   Are you absolutely sure?"},
	}
	for _, tt := range tests {
		if got := purgeConfirmMessage(dir, tt.opts); got != tt.want {
			t.Errorf("%s:\n got: %q\nwant: %q", tt.name, got, tt.want)
		}
	}
}

// Without links every prompt is word for word what it was.
func TestPurgePromptWithoutLinks(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		opts PurgeOptions
		want string
	}{
		{PurgeOptions{TargetConfig: true}, "This will:\n   ⚠️  reset your Core Configuration (config.toml goes to trash)\nProceed?"},
		{PurgeOptions{TargetLogs: true}, "This will:\n   💀 PERMANENTLY DELETE your log files (history.log, drako.log)\nProceed?"},
		{PurgeOptions{TargetProfiles: []string{"a.profile.toml"}}, "This will:\n   ⚠️  move 1 profile(s) to trash: a.profile.toml\nProceed?"},
		{PurgeOptions{DestroyEverything: true}, "💀 This will DESTROY EVERYTHING in " + dir + ".\n   NO UNDO. NO TRASH.\n   Are you absolutely sure?"},
	}
	for _, tt := range tests {
		if got := purgeConfirmMessage(dir, tt.opts); got != tt.want {
			t.Errorf("got: %q\nwant: %q", got, tt.want)
		}
	}
}

// A config folder that is itself a link: only the link goes.
func TestPurgePromptLinkedFolder(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "drako")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	want := "💀 " + link + " is a link to " + real + ".\n" +
		"   This removes the link only. Everything in " + real + " stays.\n" +
		"   Proceed?"
	if got := purgeConfirmMessage(link, PurgeOptions{DestroyEverything: true}); got != want {
		t.Errorf("got: %q\nwant: %q", got, want)
	}
	if got, want := destroyedLine(link, real), "✓ Removed the link "+link+" ("+real+" and its files are untouched)"; got != want {
		t.Errorf("got: %q\nwant: %q", got, want)
	}
	if got, want := destroyedLine(link, ""), "✓ Full destruction completed - "+link+" has been deleted"; got != want {
		t.Errorf("got: %q\nwant: %q", got, want)
	}
}

func TestPurgeResultLines(t *testing.T) {
	tests := []struct{ got, want string }{
		{trashedLine("work.profile.toml", ""), "  ✓ Moved work.profile.toml to trash"},
		{trashedLine("work.profile.toml", "/d/work.profile.toml"), "  ✓ Moved the link work.profile.toml to trash (/d/work.profile.toml is untouched)"},
		{deletedLine("history.log", ""), "  💀 Deleted history.log"},
		{deletedLine("history.log", "/d/history.log"), "  💀 Removed the link history.log (/d/history.log is untouched)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got: %q\nwant: %q", tt.got, tt.want)
		}
	}
}

// The purge itself: links go, the files they point to stay as they were.
func TestPurgeLeavesLinkedFilesAlone(t *testing.T) {
	dir, dots := linkedLair(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))
	opts := PurgeOptions{TargetConfig: true, TargetLogs: true, TargetProfiles: []string{"work.profile.toml"}}
	if err := PurgeConfig(dir, opts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.toml", "history.log", "work.profile.toml"} {
		if data, err := os.ReadFile(filepath.Join(dots, name)); err != nil || string(data) != "mine" {
			t.Errorf("%s in the dotfiles folder: %q, %v; want it untouched", name, data, err)
		}
	}
	if lair.LinkTarget(dir, "config.toml") != "" {
		t.Error("config.toml should be a fresh ordinary file")
	}
	if entries, _ := os.ReadDir(filepath.Join(dir, "trash")); len(entries) != 2 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("trash holds %s; want the two links", strings.Join(names, ", "))
	}
}

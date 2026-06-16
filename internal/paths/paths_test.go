package paths

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// TestConfigDir pins the resolution contract the rest of the codebase
// relies on: XDG_CONFIG_HOME wins when set, HOME/.config is the normal
// Linux path, and an unresolvable platform directory is an error — never
// a guessed location.
func TestConfigDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("pins Linux (XDG) behavior only")
	}

	cases := []struct {
		name    string
		xdg     string // XDG_CONFIG_HOME ("" = unset)
		home    string // HOME ("" = unset)
		want    string
		wantErr bool
	}{
		{
			name: "XDG_CONFIG_HOME set: used directly",
			xdg:  "/fake/xdg",
			home: "/fake/home",
			want: "/fake/xdg/drako",
		},
		{
			name: "XDG unset: falls back to HOME/.config",
			xdg:  "",
			home: "/fake/home",
			want: "/fake/home/.config/drako",
		},
		{
			name:    "XDG and HOME both unset: error",
			xdg:     "",
			home:    "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tc.xdg)
			t.Setenv("HOME", tc.home)

			got, err := ConfigDir()

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != filepath.FromSlash(tc.want) {
				t.Errorf("ConfigDir() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The join helpers are pure string builders; one table covers them all.
// If a well-known name ever changes, exactly one constant and one line
// here change with it.
func TestWellKnownLocations(t *testing.T) {
	const cfg = "/fake/config/drako"

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"inventory dir", InventoryDir(cfg), cfg + "/inventory"},
		{"specs dir", SpecsDir(cfg), cfg + "/specs"},
		{"assets dir", AssetsDir(cfg, "docker"), cfg + "/assets/docker"},
		{"trash dir", TrashDir(cfg), cfg + "/trash"},
		{"config file", ConfigFile(cfg), cfg + "/config.toml"},
		{"pivot file", PivotFile(cfg), cfg + "/pivot.toml"},
		{"themes file", ThemesFile(cfg), cfg + "/themes.toml"},
		{"log file", LogFile(cfg), cfg + "/drako.log"},
		{"log archive", LogArchive(cfg), cfg + "/drako.log.old"},
		{"history file", HistoryFile(cfg), cfg + "/history.log"},
		{"history archive", HistoryArchive(cfg), cfg + "/history.log.old"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != filepath.FromSlash(tc.want) {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

// TestLogFilesIncludesArchives locks the coupling that the .old fix was
// about: "drako purge --logs" must delete the rotated backups, not just the
// live logs. If an archive is ever dropped from this list, the rotated file
// it names would survive a purge — this test fails before that ships.
func TestLogFilesIncludesArchives(t *testing.T) {
	const cfg = "/fake/config/drako"

	want := []string{
		LogFile(cfg),
		LogArchive(cfg),
		HistoryFile(cfg),
		HistoryArchive(cfg),
	}

	got := LogFiles(cfg)
	if !slices.Equal(got, want) {
		t.Errorf("LogFiles() = %v, want %v", got, want)
	}
}

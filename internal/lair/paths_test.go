package lair

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestConfigDir pins ~/.config/drako on every OS, an absolute XDG_CONFIG_HOME
// over it, and an error rather than a guessed location.
func TestConfigDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows finds home through USERPROFILE, not HOME")
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
			name:    "relative XDG_CONFIG_HOME: error",
			xdg:     "relative/xdg",
			home:    "/fake/home",
			wantErr: true,
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

// One table pins every well-known location, so a renamed one changes one
// constant and one line here.
func TestWellKnownLocations(t *testing.T) {
	const cfg = "/fake/config/drako"

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"inventory dir", InventoryDir(cfg), cfg + "/inventory"},
		{"specs dir", SpecsDir(cfg), cfg + "/specs"},
		{"trash dir", TrashDir(cfg), cfg + "/trash"},
		{"config file", ConfigFile(cfg), cfg + "/config.toml"},
		{"pivot file", PivotFile(cfg), cfg + "/pivot.toml"},
		{"themes file", ThemesFile(cfg), cfg + "/themes.toml"},
		{"log file", LogFile(cfg), cfg + "/drako.log"},
		// Opened by name only (rotation, purge --logs): the name is the location.
		{"log archive", LogArchiveName, "drako.log.old"},
		{"history file", HistoryFileName, "history.log"},
		{"history archive", HistoryArchiveName, "history.log.old"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != filepath.FromSlash(tc.want) {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

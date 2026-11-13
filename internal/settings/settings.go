// Package settings says what config.toml means: its fields and keybindings,
// the defaults that fill its gaps, the rescue grid drako falls back to, the
// theme palettes, and how the active profile is laid over it all. It works
// on bytes and values only; reading files is the caller's job.
package settings

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/profile"
)

// FileName is the settings file's name inside the config directory.
const FileName = "config.toml"

// Config is the settings in effect: what config.toml says, with the active
// profile laid over it (its grid size and commands always; its theme, header
// art and shell when it sets them).
type Config struct {
	Theme              string
	HeaderArt          *string
	DefaultShell       string
	NumbModifier       string
	X                  int
	Y                  int
	Profile            string
	LockTimeoutMinutes *int
	AutoLockEnabled    *bool
	BatchForceTmux     bool
	Keys               InputConfig
	Commands           []profile.Command
}

// file is config.toml as written.
type file struct {
	DefaultShell       string      `toml:"default_shell"`
	NumbModifier       string      `toml:"numb_modifier"`
	Profile            string      `toml:"profile"`
	LockTimeoutMinutes *int        `toml:"lock_timeout_minutes"`
	AutoLockEnabled    *bool       `toml:"auto_lock_enabled"`
	BatchForceTmux     bool        `toml:"batch_force_tmux"`
	Theme              string      `toml:"theme"` // Global Fallback Theme
	Keys               InputConfig `toml:"keys"`
}

// Decode reads config.toml into the base Config: no commands, those only ever
// come from a profile. unknown lists every key path that matched no setting,
// e.g. "keys.lader"; err is a TOML error.
func Decode(raw []byte) (base Config, unknown []string, err error) {
	var written file
	meta, err := toml.Decode(string(raw), &written)
	if err != nil {
		return Config{}, nil, err
	}
	for _, k := range meta.Undecoded() {
		unknown = append(unknown, k.String())
	}
	return Config{
		DefaultShell:       written.DefaultShell,
		NumbModifier:       written.NumbModifier,
		Profile:            written.Profile,
		LockTimeoutMinutes: written.LockTimeoutMinutes,
		AutoLockEnabled:    written.AutoLockEnabled,
		BatchForceTmux:     written.BatchForceTmux,
		Theme:              written.Theme,
		Keys:               written.Keys,
		Commands:           []profile.Command{},
	}, unknown, nil
}

// Lint checks raw config.toml: what sends the loader to rescue mode (an
// error), keys it ignores (a typo, or a setting drako no longer has), and
// bindings that cannot work as written (KeyFindings).
func Lint(src []byte) []profile.Finding {
	base, unknown, err := Decode(src)
	if err != nil {
		return []profile.Finding{{Level: "error", Msg: fmt.Sprintf("%v", err)}}
	}
	var fs []profile.Finding
	if len(unknown) > 0 {
		fs = append(fs, profile.Finding{Level: "warning", Msg: fmt.Sprintf("unrecognized key(s): %s (a typo, or a setting drako no longer has)", strings.Join(unknown, ", "))})
	}
	base.ApplyDefaults()
	return append(fs, KeyFindings(base)...)
}

// ApplyProfileOverlay layers a profile on top of the base settings: grid
// size and commands always come from the profile; theme, header art and
// shell only when the profile sets them.
func ApplyProfileOverlay(base Config, pf profile.File) Config {
	cfg := base

	cfg.X = pf.X
	cfg.Y = pf.Y
	if strings.TrimSpace(pf.Theme) != "" {
		cfg.Theme = pf.Theme
	}
	if pf.HeaderArt != nil {
		cfg.HeaderArt = pf.HeaderArt
	}
	if pf.Shell != nil {
		cfg.DefaultShell = *pf.Shell
	}
	cfg.Commands = profile.CopyCommands(pf.Commands)

	return cfg
}

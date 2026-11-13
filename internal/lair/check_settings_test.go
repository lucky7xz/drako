package lair

import (
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/settings"
)

// The shipped template must lint clean, or every new install starts with a warning.
func TestCheckSettings_TemplateIsClean(t *testing.T) {
	src, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		t.Fatal(err)
	}
	if fs := settings.Lint(src); len(fs) != 0 {
		t.Errorf("the settings template has findings: %+v", fs)
	}
}

// A setting the template shows commented out must work when its # is removed
// where it stands: a top-level setting below [keys] would land in that table.
func TestSettingsTemplate_EverySettingWorksWhereItStands(t *testing.T) {
	src, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(string(src), "\n") {
		set := strings.TrimPrefix(line, "#")
		key, _, isSetting := strings.Cut(set, " = ")
		if strings.HasPrefix(line, "[") {
			seen = map[string]bool{} // a new table: its own keys
		}
		if !isSetting || strings.ContainsAny(key, " #") {
			out = append(out, line)
			continue
		}
		if seen[key] { // an alternative value for a setting already set (the themes)
			continue
		}
		seen[key] = true
		out = append(out, set)
	}
	if fs := settings.Lint([]byte(strings.Join(out, "\n"))); len(fs) != 0 {
		t.Errorf("with every setting uncommented where it stands: %+v", fs)
	}
}

// A key the loader ignores is a typo or a removed setting: one warning names them all.
func TestCheckSettings_NamesTheKeysTheLoaderIgnores(t *testing.T) {
	src := "env_whitelist = [\"PATH\"]\nthme = \"nord\"\ntheme = \"nord\"\n[keys]\nlader = \"m\"\n"
	fs := settings.Lint([]byte(src))
	if len(fs) != 1 || fs[0].Level != "warning" {
		t.Fatalf("want one warning, got %+v", fs)
	}
	for _, key := range []string{"env_whitelist", "thme", "keys.lader"} {
		if !strings.Contains(fs[0].Msg, key) {
			t.Errorf("warning %q does not name %s", fs[0].Msg, key)
		}
	}
}

// What sends the loader to rescue mode is an error.
func TestCheckSettings_WhatTheLoaderRejectsIsAnError(t *testing.T) {
	for _, src := range []string{"theme = \n", "batch_force_tmux = \"yes\"\n"} {
		if fs := settings.Lint([]byte(src)); len(fs) != 1 || fs[0].Level != "error" {
			t.Errorf("%q: want one error, got %+v", src, fs)
		}
	}
}

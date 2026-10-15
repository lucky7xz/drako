package config

import (
	"strings"
	"testing"
)

// The shipped template must lint clean, or every new install starts with a warning.
func TestCheckSettings_TemplateIsClean(t *testing.T) {
	src, err := bootstrapFS.ReadFile("bootstrap/settings_template.toml")
	if err != nil {
		t.Fatal(err)
	}
	if fs := CheckSettings(src); len(fs) != 0 {
		t.Errorf("the settings template has findings: %+v", fs)
	}
}

// A key the loader ignores is a typo or a removed setting: one warning names them all.
func TestCheckSettings_NamesTheKeysTheLoaderIgnores(t *testing.T) {
	src := "env_whitelist = [\"PATH\"]\nthme = \"nord\"\ntheme = \"nord\"\n[keys]\nlader = \"m\"\n"
	fs := CheckSettings([]byte(src))
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
		if fs := CheckSettings([]byte(src)); len(fs) != 1 || fs[0].Level != "error" {
			t.Errorf("%q: want one error, got %+v", src, fs)
		}
	}
}

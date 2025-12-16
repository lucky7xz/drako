package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefactorBootstrap(t *testing.T) {
	// Create temp dir for XDG_CONFIG_HOME
	tempDir, err := os.MkdirTemp("", "drako_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set env var to redirect config dir
	// GetConfigDir uses os.UserConfigDir which uses XDG_CONFIG_HOME on Linux
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	// Also unset HOME to forces dependency on XDG on some systems, or set HOME to tempDir
	t.Setenv("HOME", tempDir)

	// Subdir expected: $XDG_CONFIG_HOME/drako
	expectedDir := filepath.Join(tempDir, "drako")

	// 1. Run LoadConfig (Should bootstrap)
	bundle := LoadConfig(nil)

	// 2. Verify config.toml exists and has no commands
	configPath := filepath.Join(expectedDir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config.toml was not created at %s", configPath)
	}
	content, _ := os.ReadFile(configPath)
	if strings.Contains(string(content), "[[commands]]") {
		t.Errorf("config.toml should NOT contain [[commands]]")
	}
	// Theme is now allowed (and expected) in config.toml
	if !strings.Contains(string(content), "theme =") {
		t.Errorf("config.toml SHOULD contain theme")
	}

	// 3. Verify core.profile.toml exists and HAS commands
	profilePath := filepath.Join(expectedDir, "core.profile.toml")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Errorf("core.profile.toml was not created at %s", profilePath)
	}
	pContent, _ := os.ReadFile(profilePath)
	if !strings.Contains(string(pContent), "[[commands]]") {
		t.Errorf("core.profile.toml SHOULD contain [[commands]]")
	}
	if !strings.Contains(string(pContent), "theme =") {
		t.Errorf("core.profile.toml SHOULD contain theme")
	}

	// 4. Verify loaded bundle
	if len(bundle.Config.Commands) == 0 {
		t.Errorf("Bundle.Config should have commands loaded from profile, got 0")
	}

	// Check if defaults are loaded
	if bundle.Config.Theme != "dracula" {
		t.Errorf("Expected theme dracula, got %s", bundle.Config.Theme)
	}

	// Check multiline handling (implicit check via bootstrap success, but could inspect file)
	// If the weaver was broken, the TOML decode above (or in app) would likely fail or produce truncated strings.
	// We trust that if LoadConfig succeeded with non-empty commands, we are good.
}

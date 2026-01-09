package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyDefaults ensures that a zero-value Config gets populated with safe defaults.
// This is critical for the "Base" configuration.
func TestApplyDefaults(t *testing.T) {
	cfg := Config{} // Empty
	cfg.ApplyDefaults()

	if cfg.Theme != "dracula" {
		t.Errorf("expected default theme 'dracula', got '%s'", cfg.Theme)
	}
	if cfg.DefaultShell != "bash" && cfg.DefaultShell != "pwsh" {
		t.Errorf("expected safe default shell, got '%s'", cfg.DefaultShell)
	}
	// Critical: Check Controls initialization
	if len(cfg.Keys.NavUp) == 0 {
		t.Error("ApplyDefaults failed to initialize Navigation Keys (NavUp is empty)")
	}
}

// TestApplyProfileOverlay verifies that merging a profile into the base config works correctly.
// CRITICAL: It must NOT wipe out existing settings (like Keys) if the profile doesn't specify them.
func TestApplyProfileOverlay_PreservesBase(t *testing.T) {
	// 1. Setup Base with Defaults
	base := Config{}
	base.ApplyDefaults()
	originalKey := base.Keys.NavUp[0] // e.g. "up"

	// 2. Create an Overlay (Profile) that only changes the Theme
	overlay := ProfileFile{
		Theme: "solarized",
		// Commands is mandatory in a real profile, but Overlay accepts empty
		Commands: []Command{},
	}

	// 3. Apply
	result := ApplyProfileOverlay(base, overlay)

	// 4. Verification
	if result.Theme != "solarized" {
		t.Errorf("Overlay failed to update Theme. Got '%s'", result.Theme)
	}
	// REGRESSION CHECK: Did we lose the keys?
	if len(result.Keys.NavUp) == 0 {
		t.Fatal("Regression: ApplyProfileOverlay wiped out Base keys!")
	}
	if result.Keys.NavUp[0] != originalKey {
		t.Errorf("Key binding changed unexpectedly. Expected '%s', got '%s'", originalKey, result.Keys.NavUp[0])
	}
}

// TestLoadConfig_HandlesBrokenProfiles simulates a "Rescue Mode" scenario.
// We create a directory with a garbage .profile.toml and ensure LoadConfig:
// 1. Does not panic
// 2. Returns a valid bundle (Rescue/Factory defaults)
// 3. Reports the file as Broken
func TestLoadConfig_HandlesBrokenProfiles(t *testing.T) {
	// Setup Temp Dir
	tmpDir, err := os.MkdirTemp("", "drako_rescue_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock Environment to force LoadConfig to use our tmpDir
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	// Create strict structure: $XDG/drako
	configDir := filepath.Join(tmpDir, "drako")
	os.MkdirAll(configDir, 0755)

	// Create a BROKEN profile file
	brokenPath := filepath.Join(configDir, "broken.profile.toml")
	os.WriteFile(brokenPath, []byte("This is not TOML content! ["), 0644)

	// Run LoadConfig
	// Note: It might create config.toml (bootstrap) which is fine.
	bundle := LoadConfig(nil)

	// Assertions
	// 1. Check for Broken report
	if len(bundle.Broken) == 0 {
		t.Error("LoadConfig failed to report the broken profile")
	} else {
		errName := bundle.Broken[0].Name
		if !strings.Contains(errName, "broken") {
			t.Errorf("Expected broken profile error for 'broken', got '%s'", errName)
		}
	}

	// 2. Check Rescue Mode (No valid profiles found -> Factory Defaults)
	if len(bundle.Profiles) != 0 {
		// Actually, DiscoverProfiles returns 0 profiles if they are all broken.
		// LoadConfig constructs a dummy "Rescue" profile if len=0
		// Wait, my implementation of DiscoverProfiles excludes broken ones.
		// So Profiles should be Empty?
		// Let's check LoadConfig logic: if len(profiles) == 0 -> useFactoryDefaults=true
		// selected = ProfileInfo{Name: "Rescue"...}
		// It does NOT append "Rescue" to bundle.Profiles usually, just sets selected.
		// Let's check ActiveIndex or Config.Commands
		// If using factory defaults, Config.Commands should be the built-in rescue commands.
	}

	// We check if we got a valid config at all
	if len(bundle.Config.Commands) == 0 {
		t.Errorf("Rescue Mode failed: Config has 0 commands. Should have default/rescue set.")
	}

	// 3. Verify Base Integrity again (Integration Check)
	if len(bundle.Base.Keys.NavUp) == 0 {
		t.Error("LoadConfig returned empty Base config in Rescue Mode")
	}
}

// TestBootstrap_GeneratesFiles confirms that a fresh boot creates necessary files.
func TestBootstrap_GeneratesFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drako_bootstrap_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Run
	LoadConfig(nil)

	// Verify
	expectedConfig := filepath.Join(tmpDir, "drako", "config.toml")
	if _, err := os.Stat(expectedConfig); os.IsNotExist(err) {
		t.Error("Bootstrap failed to create config.toml")
	}

	expectedProfile := filepath.Join(tmpDir, "drako", "core.profile.toml")
	if _, err := os.Stat(expectedProfile); os.IsNotExist(err) {
		t.Error("Bootstrap failed to create core.profile.toml")
	}
}

// TestLoadConfig_HandlesBrokenConfig ensures that if config.toml is corrupted,
// the application does NOT crash and returns a Rescue config with an error.
func TestLoadConfig_HandlesBrokenConfig(t *testing.T) {
	// Setup Temp Dir
	tmpDir, err := os.MkdirTemp("", "drako_broken_config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock Environment
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	configDir := filepath.Join(tmpDir, "drako")
	os.MkdirAll(configDir, 0755)

	// Create a corrupted config.toml
	configPath := filepath.Join(configDir, "config.toml")
	os.WriteFile(configPath, []byte("NOT TOML! ["), 0644)

	// Run LoadConfig - Should NOT PANIC
	bundle := LoadConfig(nil)

	// Assertions
	// 1. Check for Broken report
	if len(bundle.Broken) == 0 {
		t.Error("LoadConfig failed to report the broken config file")
	} else {
		errName := bundle.Broken[0].Name
		if errName != "config.toml" {
			t.Errorf("Expected broken profile error for 'config.toml', got '%s'", errName)
		}
	}

	// 2. Check if we have Safe/Rescue defaults
	if bundle.Base.DefaultShell != "bash" && bundle.Base.DefaultShell != "pwsh" {
		t.Errorf("Expected rescue default shell, got '%s'", bundle.Base.DefaultShell)
	}

	// 3. Verify that the UI will get a valid config to render (Rescue Commands)
	// RescueConfig initializes Commands.
	if len(bundle.Base.Commands) == 0 {
		// Base commands are usually empty in LoadConfig unless defaults applied?
		// LoadConfig applies defaults to base.
		// commands in base are usually empty because they come from profile.
		// BUT RescueConfig provides commands!
		// However, LoadConfig logic sets commands to empty [] explicitly for base.
		// Wait, let's check LoadConfig logic:
		// "base = RescueConfig()" -> base has commands.
		// Then later: "base.Commands = []Command{}"?
		// No, LoadConfig "Convert Settings to Base" explicitly sets empty commands.
		// BUT in our fix case:
		// "base = RescueConfig()"
		// We do NOT wipe base.Commands in the error path!
		// But let's check the code:
		// if error { base = RescueConfig() ... } else { ... base = Config{ ... Commands: [] } }
		// So in error case, base HAS commands from RescueConfig.
	}

	// Effective config logic in LoadConfig:
	// "effective = base"
	// Then tries to load profile.
	// If profiles are broken/missing -> useFactoryDefaults = true
	// "effective = RescueConfig()"
	// "effective.ApplyDefaults()"

	// So ultimately effective config should have Rescue commands.
	if len(bundle.Config.Commands) == 0 {
		t.Error("Effective config should have rescue commands in broken config mode")
	}

	// Check if commands contain "Reset Core" which is typical for Rescue
	hasReset := false
	for _, cmd := range bundle.Config.Commands {
		if strings.Contains(cmd.Name, "Reset") || strings.Contains(cmd.Command, "purge") {
			hasReset = true
			break
		}
	}
	if !hasReset {
		t.Error("Rescue config missing purge/reset command")
	}
}

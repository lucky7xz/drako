package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPurgeConfig_TargetConfig(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "drako_purge_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config.toml
	configFile := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configFile, []byte("theme = 'dracula'"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy profile just to ensure it's NOT touched
	profileFile := filepath.Join(tmpDir, "git.profile.toml")
	if err := os.WriteFile(profileFile, []byte("x = 3"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := PurgeOptions{
		TargetConfig: true,
	}

	// Run Purge
	if err := PurgeConfig(tmpDir, opts); err != nil {
		t.Fatalf("PurgeConfig failed: %v", err)
	}

	// Check config.toml is gone (moved to trash)
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Error("config.toml should have been moved to trash")
	}

	// Check profile file still exists
	if _, err := os.Stat(profileFile); os.IsNotExist(err) {
		t.Error("git.profile.toml should NOT have been touched")
	}

	// Verify it's in trash
	trashDir := filepath.Join(tmpDir, "trash")
	entries, _ := os.ReadDir(trashDir)
	found := false
	for _, e := range entries {
		// Timestamped filename check
		if len(e.Name()) > len("config.toml") { // roughly check
			found = true
			break
		}
	}
	if !found {
		t.Error("config.toml not found in trash")
	}
}

func TestPurgeConfig_TargetProfile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drako_purge_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config.toml (should stay)
	configFile := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configFile, []byte("theme = 'dracula'"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create core.profile.toml (target)
	coreProfile := filepath.Join(tmpDir, "core.profile.toml")
	if err := os.WriteFile(coreProfile, []byte("x = 3"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := PurgeOptions{
		TargetProfile: "core",
	}

	if err := PurgeConfig(tmpDir, opts); err != nil {
		t.Fatalf("PurgeConfig failed: %v", err)
	}

	// Check core.profile.toml is gone
	if _, err := os.Stat(coreProfile); !os.IsNotExist(err) {
		t.Error("core.profile.toml should have been moved to trash")
	}

	// Check config.toml still exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("config.toml should NOT have been touched")
	}
}

func TestPurgeConfig_SafetyCheck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drako_purge_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.toml")
	os.WriteFile(configFile, []byte(""), 0644)

	profileFile := filepath.Join(tmpDir, "git.profile.toml")
	os.WriteFile(profileFile, []byte(""), 0644)

	opts := PurgeOptions{
		TargetProfile: "",
		TargetConfig:  false,
		// No args = Safety check (should fail or do nothing)
	}

	// It should return an error now because no target is specified
	if err := PurgeConfig(tmpDir, opts); err == nil {
		t.Error("PurgeConfig should return error when no target is specified")
	}

	// Nothing should be touched
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("config.toml should persist")
	}
	if _, err := os.Stat(profileFile); os.IsNotExist(err) {
		t.Error("git.profile.toml should persist")
	}
}

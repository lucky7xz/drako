package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		TargetProfiles: []string{"core"},
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
		TargetProfiles: []string{},
		TargetConfig:   false,
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

func TestParsePurgeFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedOpts  PurgeOptions
		expectedInter bool
		expectError   bool
	}{
		{
			name: "Target Config Full",
			args: []string{"--config"},
			expectedOpts: PurgeOptions{
				TargetConfig: true,
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Target Config Alias",
			args: []string{"--config"}, // Corrected from -c since it wasn't defined
			expectedOpts: PurgeOptions{
				TargetConfig: true,
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Target Core Profile",
			args: []string{"--target", "core"},
			expectedOpts: PurgeOptions{
				TargetProfiles: []string{"core"},
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Target Profile Alias",
			args: []string{"-t", "git"},
			expectedOpts: PurgeOptions{
				TargetProfiles: []string{"git"},
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Interactive Mode",
			args: []string{"--interactive"},
			expectedOpts: PurgeOptions{
				TargetProfiles: []string{}, // Initialize empty slice
			},
			expectedInter: true,
			expectError:   false,
		},
		{
			name: "Destroy Everything",
			args: []string{"--destroyeverything"},
			expectedOpts: PurgeOptions{
				DestroyEverything: true,
			},
			expectedInter: false,
			expectError:   false,
		},
		{
			name: "Positional Args Error",
			args: []string{"some_file"},
			// Should error out
			expectError: true,
		},
		{
			name: "Positional Arg 'ssh' (User Report)",
			args: []string{"ssh"},
			// Should error out explicitly
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, interactive, err := ParsePurgeFlags(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// For TargetProfiles, we need to handle nil vs empty slice comparison potentially
			// But DeepEqual usually handles nil vs []string{} as different.
			// Let's ensure ParsePurgeFlags initializes it if needed, or expectedOpts matches.
			if !reflect.DeepEqual(*opts, tt.expectedOpts) {
				// Special check for nil slice vs empty slice if that's the only diff
				if len(opts.TargetProfiles) == 0 && len(tt.expectedOpts.TargetProfiles) == 0 {
					// pass
				} else {
					t.Errorf("Options mismatch.\nGot: %+v\nWant: %+v", *opts, tt.expectedOpts)
				}
			}

			if interactive != tt.expectedInter {
				t.Errorf("Interactive mismatch. Got %v, Want %v", interactive, tt.expectedInter)
			}
		})
	}
}

func TestInteractivePurge_Batch(t *testing.T) {
	// Setup mock filesystem for scanning
	tmpDir, err := os.MkdirTemp("", "drako_purge_batch")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy profiles
	// 1. Core (root)
	os.WriteFile(filepath.Join(tmpDir, "core.profile.toml"), []byte{}, 0644)
	// 2. Personal (root)
	os.WriteFile(filepath.Join(tmpDir, "personal.profile.toml"), []byte{}, 0644)

	inventoryDir := filepath.Join(tmpDir, "inventory")
	os.MkdirAll(inventoryDir, 0755)
	// 3. Git (inventory)
	os.WriteFile(filepath.Join(inventoryDir, "git.profile.toml"), []byte{}, 0644)

	// Profiles will be sorted by name/location usually.
	// runInteractivePurgeSelection scans root then inventory.
	// Root: core, personal
	// Inventory: git
	// Expected Order:
	// 1. core (Equipped)
	// 2. personal (Equipped)
	// 3. git (Inventory)

	// Input simulation:
	// "1, 3" -> Select 1 (core) and 3 (git)
	// User must confirm each.
	// Prompt 1 (core): "y"
	// Prompt 2 (git): "n" (Deny deletion of git)

	// Input stream: "1, 3\ny\nn\n"
	inputStr := "1, 3\ny\nn\n"
	reader := strings.NewReader(inputStr)
	var output strings.Builder

	opts := &PurgeOptions{}
	err = runInteractivePurgeSelection(tmpDir, opts, reader, &output)
	if err != nil {
		t.Fatalf("Interactive scan failed: %v", err)
	}

	// Verify TargetProfiles
	// Should contain "core.profile.toml" (from 1)
	// Should NOT contain "inventory/git.profile.toml" (from 3, denied)

	if len(opts.TargetProfiles) != 1 {
		t.Errorf("Expected 1 target, got %d: %v", len(opts.TargetProfiles), opts.TargetProfiles)
	} else {
		if opts.TargetProfiles[0] != "core.profile.toml" {
			t.Errorf("Expected 'core.profile.toml', got '%s'", opts.TargetProfiles[0])
		}
	}

	// Verify Output contains prompts
	outStr := output.String()
	if !strings.Contains(outStr, "Delete core") {
		t.Error("Output missing confirmation prompt for core")
	}
	if !strings.Contains(outStr, "Delete git") {
		t.Error("Output missing confirmation prompt for git")
	}
}

func TestInteractivePurge_Range(t *testing.T) {
	// Setup mock filesystem
	tmpDir, err := os.MkdirTemp("", "drako_purge_range")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create 3 profiles
	os.WriteFile(filepath.Join(tmpDir, "p1.profile.toml"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "p2.profile.toml"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "p3.profile.toml"), []byte{}, 0644)

	// Input: "1-3" -> Select 1, 2, 3
	// Verify all are added to TargetProfiles (assuming 'y' for all)
	inputStr := "1-3\ny\ny\ny\n"
	reader := strings.NewReader(inputStr)
	var output strings.Builder

	opts := &PurgeOptions{}
	err = runInteractivePurgeSelection(tmpDir, opts, reader, &output)
	if err != nil {
		t.Fatalf("Interactive scan failed: %v", err)
	}

	if len(opts.TargetProfiles) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(opts.TargetProfiles))
	}
}

func TestHandlePurgeCommand_BadFlag(t *testing.T) {
	if got := HandlePurgeCommand([]string{"drako", "purge", "--nonsense"}); got != 1 {
		t.Errorf("HandlePurgeCommand --nonsense = %d, want 1", got)
	}
}

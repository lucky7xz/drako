package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/config"
)

// Spec defines a named set of visible profiles.
type Spec struct {
	Profiles []string `toml:"profiles"`
}

func HandleSpecCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako spec <name>\n")
		fmt.Fprintf(os.Stderr, "  Loads a profile specification from ~/.config/drako/specs/<name>.toml\n")
		os.Exit(1)
	}

	specName := os.Args[2]
	// Handle .toml extension if provided or not
	if !strings.HasSuffix(specName, ".toml") {
		specName += ".toml"
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		log.Fatalf("could not get config dir: %v", err)
	}

	specsDir := filepath.Join(configDir, "specs")
	specPath := filepath.Join(specsDir, specName)

	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		// Try checking if it's just a name without dir? No, we expect it in specs/
		fmt.Fprintf(os.Stderr, "Spec not found: %s\n", specPath)
		fmt.Fprintf(os.Stderr, "Please create a spec file in %s\n", specsDir)
		fmt.Fprintf(os.Stderr, "Example content:\n")
		fmt.Fprintf(os.Stderr, "profiles = [\"git\", \"work\"]\n")
		os.Exit(1)
	}

	var spec Spec
	if _, err := toml.DecodeFile(specPath, &spec); err != nil {
		log.Fatalf("failed to parse spec: %v", err)
	}

	if err := ApplySpec(configDir, spec.Profiles); err != nil {
		log.Fatalf("failed to apply spec: %v", err)
	}

	fmt.Printf("✓ Spec '%s' applied successfully.\n", strings.TrimSuffix(specName, ".toml"))
	os.Exit(0)
}

func ApplySpec(configDir string, targetProfiles []string) error {
	inventoryDir := filepath.Join(configDir, "inventory")
	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		return err
	}

	// Normalize target list
	targetSet := make(map[string]bool)
	for _, p := range targetProfiles {
		targetSet[config.NormalizeProfileName(p)] = true
	}

	// 1. Move profiles from Inventory to Visible (if in target)
	invEntries, err := os.ReadDir(inventoryDir)
	if err == nil {
		for _, entry := range invEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".profile.toml") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".profile.toml")
			norm := config.NormalizeProfileName(name)

			if targetSet[norm] {
				src := filepath.Join(inventoryDir, entry.Name())
				dst := filepath.Join(configDir, entry.Name())
				if err := os.Rename(src, dst); err != nil {
					return fmt.Errorf("failed to move %s to visible: %w", entry.Name(), err)
				}
				fmt.Printf("  + Equipped: %s\n", name)
			}
		}
	}

	// 2. Move profiles from Visible to Inventory (if NOT in target)
	visEntries, err := os.ReadDir(configDir)
	if err != nil {
		return err
	}
	for _, entry := range visEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".profile.toml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".profile.toml")
		norm := config.NormalizeProfileName(name)

		// Skip Core/Default
		if norm == "core" || norm == "default" {
			continue
		}

		if !targetSet[norm] {
			src := filepath.Join(configDir, entry.Name())
			dst := filepath.Join(inventoryDir, entry.Name())
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("failed to move %s to inventory: %w", entry.Name(), err)
			}
			fmt.Printf("  - Stored: %s\n", name)
		}
	}

	// 3. Update Pivots (Equipped Order)
	// Ensure Core is in the list for safety
	finalOrder := make([]string, 0, len(targetProfiles)+1)
	hasCore := false
	for _, p := range targetProfiles {
		if config.NormalizeProfileName(p) == "core" {
			hasCore = true
		}
		finalOrder = append(finalOrder, p)
	}
	if !hasCore {
		// Prepend Core
		finalOrder = append([]string{"Core"}, finalOrder...)
	}

	return config.WritePivotEquippedOrder(configDir, finalOrder)
}


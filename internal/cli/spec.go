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

func HandleSpecCommand(args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako spec <name>\n")
		fmt.Fprintf(os.Stderr, "  Loads a profile specification from ~/.config/drako/specs/<name>.spec.toml\n")
		os.Exit(1)
	}

	specName := args[2]

	configDir, err := config.GetConfigDir()
	if err != nil {
		log.Fatalf("could not get config dir: %v", err)
	}

	specsDir := filepath.Join(configDir, "specs")
	// Try resolve the spec path
	specPath, err := resolveSpecPath(specsDir, specName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Spec not found: %s\n", specName)
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

	fmt.Printf("✓ Spec '%s' applied successfully.\n", strings.TrimSuffix(filepath.Base(specPath), ".spec.toml"))
	os.Exit(0)
}

func HandleStashCommand(args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako stash <name>\n")
		fmt.Fprintf(os.Stderr, "  Stashes profiles listed in ~/.config/drako/specs/<name>.spec.toml to inventory\n")
		os.Exit(1)
	}

	specName := args[2]

	configDir, err := config.GetConfigDir()
	if err != nil {
		log.Fatalf("could not get config dir: %v", err)
	}

	specsDir := filepath.Join(configDir, "specs")
	specPath, err := resolveSpecPath(specsDir, specName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Spec not found: %s\n", specName)
		os.Exit(1)
	}

	var spec Spec
	if _, err := toml.DecodeFile(specPath, &spec); err != nil {
		log.Fatalf("failed to parse spec: %v", err)
	}

	if err := StashSpec(configDir, spec.Profiles); err != nil {
		log.Fatalf("failed to stash spec: %v", err)
	}

	fmt.Printf("✓ Spec '%s' stashed successfully.\n", strings.TrimSuffix(filepath.Base(specPath), ".spec.toml"))
	os.Exit(0)
}

func HandleStripCommand(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: drako strip\n")
		fmt.Fprintf(os.Stderr, "  Moves ALL profiles (except Core) to inventory.\n")
		os.Exit(1)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		log.Fatalf("could not get config dir: %v", err)
	}

	if err := StripAllProfiles(configDir); err != nil {
		log.Fatalf("failed to strip profiles: %v", err)
	}

	fmt.Printf("✓ All profiles stripped successfully.\n")
	os.Exit(0)
}

// resolveSpecPath attempts to find a spec file with .spec.toml or .toml extension
// It returns the full path to the found file, or an error if not found.
func resolveSpecPath(specsDir, name string) (string, error) {
	// Force .spec.toml extension
	if !strings.HasSuffix(name, ".spec.toml") {
		name += ".spec.toml"
	}

	specPath := filepath.Join(specsDir, name)
	if _, err := os.Stat(specPath); err == nil {
		return specPath, nil
	}

	return "", fmt.Errorf("spec file not found: %s", specPath)
}

func StashSpec(configDir string, targetProfiles []string) error {
	inventoryDir := filepath.Join(configDir, "inventory")
	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		return err
	}

	// Read current pivot/lock state
	pf, err := config.ReadPivotProfile(configDir)
	if err != nil {
		log.Printf("Warning: could not read pivot profile: %v", err)
	}

	// Normalize target list
	targetSet := make(map[string]bool)
	for _, p := range targetProfiles {
		targetSet[config.NormalizeProfileName(p)] = true
	}

	// Scan Visible profiles and move them to Inventory if they are in the target set
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

		if targetSet[norm] {
			// Check if this profile is currently locked
			if config.NormalizeProfileName(pf.Locked) == norm {
				fmt.Printf("  ! Unlocking profile: %s\n", name)
				if err := config.WritePivotLocked(configDir, ""); err != nil {
					log.Printf("Warning: failed to unlock profile %s: %v", name, err)
				}
			}

			src := filepath.Join(configDir, entry.Name())
			dst := filepath.Join(inventoryDir, entry.Name())
			if err := moveFileSafe(src, dst); err != nil {
				log.Printf("Warning: skipped stashing %s: %v", name, err)
			} else {
				fmt.Printf("  - Stashed: %s\n", name)
			}
		}
	}
	return nil
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
				if err := moveFileSafe(src, dst); err != nil {
					log.Printf("Warning: skipped moving %s: %v", name, err)
				} else {
					fmt.Printf("  + Equipped: %s\n", name)
				}
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
			if err := moveFileSafe(src, dst); err != nil {
				log.Printf("Warning: skipped moving %s: %v", name, err)
			} else {
				fmt.Printf("  - Stored: %s\n", name)
			}
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

func StripAllProfiles(configDir string) error {
	inventoryDir := filepath.Join(configDir, "inventory")
	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		return err
	}

	// Read current pivot/lock state
	pf, err := config.ReadPivotProfile(configDir)
	if err != nil {
		log.Printf("Warning: could not read pivot profile: %v", err)
	}
	// Unlock if locked
	if pf.Locked != "" {
		if err := config.WritePivotLocked(configDir, ""); err != nil {
			log.Printf("Warning: failed to unlock profile: %v", err)
		}
	}

	// Move all visible profiles to inventory
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

		src := filepath.Join(configDir, entry.Name())
		dst := filepath.Join(inventoryDir, entry.Name())
		if err := moveFileSafe(src, dst); err != nil {
			log.Printf("Warning: skipped moving %s: %v", name, err)
		} else {
			fmt.Printf("  - Stored: %s\n", name)
		}
	}

	// Reset Pivot Order to just Core
	return config.WritePivotEquippedOrder(configDir, []string{"Core"})
}

func moveFileSafe(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	return os.Rename(src, dst)
}

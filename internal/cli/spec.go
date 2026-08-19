package cli

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// Spec defines a named set of visible profiles.
type Spec struct {
	Profiles []string `toml:"profiles"`
}

func HandleSpecCommand(args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako spec <name>\n")
		fmt.Fprintf(os.Stderr, "       drako spec list\n")
		fmt.Fprintf(os.Stderr, "  Loads a profile specification from ~/.config/drako/specs/<name>.spec.toml\n")
		return 1
	}

	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}
	specsDir := paths.SpecsDir(configDir)

	if args[2] == "list" {
		if err := ListSpecs(specsDir, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "failed to list specs: %v\n", err)
			return 1
		}
		return 0
	}

	specName := args[2]

	// Try resolve the spec path
	specPath, err := resolveSpecPath(specsDir, specName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Spec not found: %s\n", specName)
		fmt.Fprintf(os.Stderr, "Please create a spec file in %s\n", specsDir)
		fmt.Fprintf(os.Stderr, "Example content:\n")
		fmt.Fprintf(os.Stderr, "profiles = [\"git\", \"work\"]\n")
		return 1
	}

	var spec Spec
	if _, err := toml.DecodeFile(specPath, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse spec: %v\n", err)
		return 1
	}

	// A spec declares a whole configuration, so it may legitimately want more
	// profiles than the leader chords can address. Say what that costs and let
	// the user decide, rather than refusing to express what the file says.
	overCapConfirmed := false
	planned, err := profiles.PlannedEquippedCount(configDir, spec.Profiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to inspect spec: %v\n", err)
		return 1
	}
	if planned > profiles.MaxEquipped {
		fmt.Printf("\n⚠️  This spec equips %d profiles, over drako's limit of %d.\n", planned, profiles.MaxEquipped)
		fmt.Printf("    Profiles %d-%d will be reachable only by cycling (o / p),\n", profiles.MaxEquipped+1, planned)
		fmt.Printf("    not by the leader + 1-9 chord.\n\n")
		if !ConfirmAction("Proceed anyway?") {
			fmt.Println("Cancelled.")
			return 1
		}
		overCapConfirmed = true
	}

	if err := ApplySpec(configDir, spec.Profiles, overCapConfirmed); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply spec: %v\n", err)
		return 1
	}

	fmt.Printf("✓ Spec '%s' applied successfully.\n", strings.TrimSuffix(filepath.Base(specPath), profiles.SpecSuffix))
	return 0
}

func HandleStashCommand(args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako stash <name>\n")
		fmt.Fprintf(os.Stderr, "  Stashes profiles listed in ~/.config/drako/specs/<name>.spec.toml to inventory\n")
		return 1
	}

	specName := args[2]

	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}

	specsDir := paths.SpecsDir(configDir)
	specPath, err := resolveSpecPath(specsDir, specName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Spec not found: %s\n", specName)
		return 1
	}

	var spec Spec
	if _, err := toml.DecodeFile(specPath, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse spec: %v\n", err)
		return 1
	}

	if err := StashSpec(configDir, spec.Profiles); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stash spec: %v\n", err)
		return 1
	}

	fmt.Printf("✓ Spec '%s' stashed successfully.\n", strings.TrimSuffix(filepath.Base(specPath), profiles.SpecSuffix))
	return 0
}

func HandleStripCommand(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: drako strip\n")
		fmt.Fprintf(os.Stderr, "  Moves ALL profiles to inventory.\n")
		return 1
	}

	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}

	if err := StripAllProfiles(configDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to strip profiles: %v\n", err)
		return 1
	}

	fmt.Printf("✓ All profiles stripped successfully.\n")
	return 0
}

// ListSpecs writes the name and profiles of every *.spec.toml file in specsDir.
// The name shown is the filename with the .spec.toml suffix removed — exactly
// what "drako spec <name>" expects.
func ListSpecs(specsDir string, out io.Writer) error {
	entries, err := profiles.ListSpecs(specsDir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintf(out, "No specs found. Create one in %s\n", specsDir)
		return nil
	}

	// Resolve each spec's profiles into a printable cell before drawing, so we
	// can size the columns to the widest content.
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		var spec Spec
		profilesCol := "(unreadable)"
		if _, derr := toml.DecodeFile(filepath.Join(specsDir, e.File), &spec); derr == nil {
			profilesCol = strings.Join(spec.Profiles, ", ")
		}
		rows = append(rows, []string{e.Name, profilesCol})
	}

	table(out, []string{"Spec", "Profiles"}, rows)
	return nil
}

// resolveSpecPath finds the spec file for name (forcing the .spec.toml suffix).
// It returns the full path to the found file, or an error if not found.
func resolveSpecPath(specsDir, name string) (string, error) {
	// Force .spec.toml extension
	if !strings.HasSuffix(name, profiles.SpecSuffix) {
		name += profiles.SpecSuffix
	}

	specPath := filepath.Join(specsDir, name)
	if _, err := os.Stat(specPath); err == nil {
		return specPath, nil
	}

	return "", fmt.Errorf("spec file not found: %s", specPath)
}

func StashSpec(configDir string, targetProfiles []string) error {
	// Read current pivot/lock state
	pf, err := config.ReadPivotProfile(configDir)
	if err != nil {
		log.Printf("Warning: could not read pivot profile: %v", err)
	}

	// Normalize target list
	targetSet := make(map[string]bool)
	for _, p := range targetProfiles {
		targetSet[profiles.NormalizeName(p)] = true
	}

	// Scan Visible profiles and move them to Inventory if they are in the target set
	visEntries, err := profiles.List(configDir)
	if err != nil {
		return err
	}

	for _, e := range visEntries {
		if !targetSet[e.Norm] {
			continue
		}
		// Check if this profile is currently locked
		if profiles.NormalizeName(pf.Locked) == e.Norm {
			fmt.Printf("  ! Unlocking profile: %s\n", e.Name)
			if err := config.WritePivotLocked(configDir, ""); err != nil {
				log.Printf("Warning: failed to unlock profile %s: %v", e.Name, err)
			}
		}

		if err := profiles.Move(configDir, e.File, profiles.Equipped, profiles.Inventory); err != nil {
			log.Printf("Warning: skipped stashing %s: %v", e.Name, err)
		} else {
			fmt.Printf("  - Stashed: %s\n", e.Name)
		}
	}
	return nil
}

// ApplySpec equips targetProfiles and stashes the rest. allowOverCap lifts the
// MaxEquipped ceiling and must only be set once the user has confirmed the
// overflow — see HandleSpecCommand.
func ApplySpec(configDir string, targetProfiles []string, allowOverCap bool) error {
	// Equip the targets, stash the rest — file moves live in the profiles
	// package; the CLI keeps the pivot bookkeeping and feedback.
	reconcile := profiles.Reconcile
	if allowOverCap {
		reconcile = profiles.ReconcileOverCap
	}
	res, err := reconcile(configDir, targetProfiles)
	if err != nil {
		return err
	}
	for _, n := range res.Equipped {
		fmt.Printf("  + Equipped: %s\n", n)
	}
	for _, n := range res.Stashed {
		fmt.Printf("  - Stored: %s\n", n)
	}

	return config.WritePivotEquippedOrder(configDir, targetProfiles)
}

func StripAllProfiles(configDir string) error {
	// Read current pivot/lock state; unlock if locked.
	pf, err := config.ReadPivotProfile(configDir)
	if err != nil {
		log.Printf("Warning: could not read pivot profile: %v", err)
	}
	if pf.Locked != "" {
		if err := config.WritePivotLocked(configDir, ""); err != nil {
			log.Printf("Warning: failed to unlock profile: %v", err)
		}
	}

	// Empty desired set ⇒ stash everything.
	res, err := profiles.Reconcile(configDir, nil)
	if err != nil {
		return err
	}
	for _, n := range res.Stashed {
		fmt.Printf("  - Stored: %s\n", n)
	}

	// Empty pivot order: next launch lands in rescue mode.
	return config.WritePivotEquippedOrder(configDir, nil)
}

package cli

// Summon fetches untrusted profile files into the inventory (quarantine).
// This file is the orchestration: source classification, user confirmation,
// and the per-file summon flow. The moving parts live next door:
//   summon_transport.go — git/HTTP transports, URL parsing, file copying
//   summon_assets.go    — asset plan/copy with limits and path-safety checks
//   summon_validate.go  — validation of everything that arrives

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucky7xz/drako/internal/paths"
)

// FileDownloader defines the interface for downloading a file
type FileDownloader interface {
	DownloadFile(url, destPath string) error
}

// RepoCloner defines the interface for cloning a git repository
type RepoCloner interface {
	CloneRepo(url, destDir string) error
	CheckGitAvailable() error
}

// UIInterface abstraction for user confirmation
type UIInterface interface {
	Confirm(prompt string) bool
}

// Summoner handles the logic for summoning profiles
type Summoner struct {
	ConfigDir  string
	Downloader FileDownloader
	Cloner     RepoCloner
	UI         UIInterface
}

// NewSummoner creates a new Summoner with real dependencies
func NewSummoner(configDir string) *Summoner {
	return &Summoner{
		ConfigDir:  configDir,
		Downloader: &HTTPDownloader{},
		Cloner:     &GitCloner{},
		UI:         &RealUI{},
	}
}

// SummonProfile Entry Point (Legacy Wrapper)
func SummonProfile(sourceURL, configDir string) error {
	summoner := NewSummoner(configDir)
	return summoner.Summon(sourceURL)
}

// Summon executes the summoning logic
func (s *Summoner) Summon(sourceURL string) error {
	inventoryDir := paths.InventoryDir(s.ConfigDir)
	if err := os.MkdirAll(inventoryDir, 0o755); err != nil {
		return fmt.Errorf("failed to create inventory directory: %w", err)
	}

	if isGitURL(sourceURL) {
		if err := s.Cloner.CheckGitAvailable(); err != nil {
			return err
		}

		if isSSHURL(sourceURL) {
			warnIfNoSSHKeys()
		}

		fmt.Printf("\nYou are about to clone a git repository:\n")
		fmt.Printf("  Source: %s\n", sourceURL)
		fmt.Printf("  Destination: %s\n", inventoryDir)
		fmt.Printf("  Action: Find and copy all .profile.toml files\n\n")

		if !s.UI.Confirm("Proceed with cloning?") {
			return fmt.Errorf("operation cancelled by user")
		}

		return s.summonFromGit(sourceURL, inventoryDir)
	}

	// HTTP/HTTPS Download
	filename := extractFilenameFromURL(sourceURL)
	if filename == "" || !strings.HasSuffix(filename, ".profile.toml") {
		filename = "personal.profile.toml"
	}

	// Safety: Check Equipped Collision
	if err := s.checkEquippedCollision(filename); err != nil {
		return err
	}

	dstPath := filepath.Join(inventoryDir, filename)
	fmt.Printf("\nYou are about to download a profile:\n")
	fmt.Printf("  Source: %s\n", sourceURL)
	fmt.Printf("  Destination: %s\n", dstPath)

	if _, err := os.Stat(dstPath); err == nil {
		fmt.Printf("  ⚠️  Warning: %s already exists and will be overwritten\n", filename)
	}
	fmt.Println()

	if !s.UI.Confirm("Proceed with download?") {
		return fmt.Errorf("operation cancelled by user")
	}

	return s.summonFromHTTP(sourceURL, inventoryDir)
}

// checkEquippedCollision ensures we don't summon a profile that conflicts with an actively equipped one (in root)
func (s *Summoner) checkEquippedCollision(filename string) error {
	equippedPath := filepath.Join(s.ConfigDir, filename)
	if _, err := os.Stat(equippedPath); err == nil {
		return fmt.Errorf("safety violation: '%s' is currently EQUIPPED (in root). Cannot overwrite active profile from inventory summon. Please unequip or stash it first", filename)
	}
	return nil
}

// summonOutcome classifies what happened to one candidate file.
type summonOutcome int

const (
	outcomeSummoned summonOutcome = iota
	outcomeSkipped
	outcomeCancelled
	outcomeFailed // copy failed after confirmation; reported but not counted
)

// summonFromGit clones a profile repository and walks the user through
// summoning every profile and spec file it contains.
func (s *Summoner) summonFromGit(repoURL, inventoryDir string) error {
	// Create a temporary directory for cloning
	tempDir := filepath.Join(inventoryDir, ".summon-temp")
	defer os.RemoveAll(tempDir)

	// Clone the repository using injected Cloner
	if err := s.Cloner.CloneRepo(repoURL, tempDir); err != nil {
		return err
	}

	profileFiles, specFiles, err := findRepoFiles(tempDir)
	if err != nil {
		return fmt.Errorf("failed to search for files: %w", err)
	}
	if len(profileFiles) == 0 && len(specFiles) == 0 {
		return fmt.Errorf("no .profile.toml or .spec.toml files found in repository")
	}
	printFoundFiles(profileFiles, specFiles)

	summoned := 0
	skipped := 0
	cancelled := 0
	tally := func(o summonOutcome) {
		switch o {
		case outcomeSummoned:
			summoned++
		case outcomeSkipped:
			skipped++
		case outcomeCancelled:
			cancelled++
		}
	}

	// 1. Process Profile Files
	for _, srcPath := range profileFiles {
		outcome, err := s.summonRepoProfile(srcPath, tempDir, inventoryDir)
		if err != nil {
			return err
		}
		tally(outcome)
	}

	// 2. Process Spec Files
	if len(specFiles) > 0 {
		configDir, _ := paths.ConfigDir()
		specsDir := paths.SpecsDir(configDir)
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			fmt.Printf("⚠️  Failed to create specs directory: %v\n", err)
		} else {
			fmt.Printf("\nProcessing spec files...\n")
			for _, srcPath := range specFiles {
				tally(s.summonRepoSpec(srcPath, specsDir))
			}
		}
	}

	// Summary
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("✓ Summoned: %d\n", summoned)
	if cancelled > 0 {
		fmt.Printf("⊘ Cancelled: %d\n", cancelled)
	}
	if skipped > 0 {
		fmt.Printf("⚠️  Skipped: %d (validation errors)\n", skipped)
	}

	if summoned == 0 {
		if cancelled > 0 {
			return fmt.Errorf("no items summoned (all cancelled by user)")
		}
		return fmt.Errorf("no valid items found in repository (%d skipped)", skipped)
	}

	log.Printf("Successfully summoned %d item(s) from repository: %s (skipped: %d, cancelled: %d)", summoned, repoURL, skipped, cancelled)
	return nil
}

// findRepoFiles walks a cloned repo for .profile.toml and .spec.toml files.
func findRepoFiles(tempDir string) (profileFiles, specFiles []string, err error) {
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".profile.toml") {
			profileFiles = append(profileFiles, path)
		} else if strings.HasSuffix(path, ".spec.toml") {
			specFiles = append(specFiles, path)
		}
		return nil
	})
	return profileFiles, specFiles, err
}

// printFoundFiles shows the user what the repository contains.
func printFoundFiles(profileFiles, specFiles []string) {
	fmt.Printf("\nFound %d file(s) in repository:\n", len(profileFiles)+len(specFiles))
	foundRows := make([][]string, 0, len(profileFiles)+len(specFiles))
	for _, srcPath := range profileFiles {
		foundRows = append(foundRows, []string{filepath.Base(srcPath), "profile"})
	}
	for _, srcPath := range specFiles {
		foundRows = append(foundRows, []string{filepath.Base(srcPath), "spec"})
	}
	table(os.Stdout, []string{"File", "Type"}, foundRows)
	fmt.Println()
}

// summonRepoProfile validates one profile from the cloned repo, shows the
// user what summoning it would do (including its asset plan), and copies it
// plus its assets on confirmation. A copy failure after confirmation is
// fatal for the whole summon (err != nil).
func (s *Summoner) summonRepoProfile(srcPath, tempDir, inventoryDir string) (summonOutcome, error) {
	dstName := filepath.Base(srcPath)
	dstPath := filepath.Join(inventoryDir, dstName)

	// Validate filename
	if err := validateFilename(dstName); err != nil {
		fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
		return outcomeSkipped, nil
	}

	// Safety Check: Equipped Collision
	// Ensure this profile doesn't conflict with what is actively equipped in root
	if err := s.checkEquippedCollision(dstName); err != nil {
		fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
		return outcomeSkipped, nil
	}

	// Validate file content before copying
	fmt.Printf("\nValidating %s...\n", dstName)
	if err := validateProfileFile(srcPath); err != nil {
		fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
		return outcomeSkipped, nil
	}

	// Parse overlay to detect assets
	assets, perr := readAssetsFromProfile(srcPath)
	if perr != nil {
		fmt.Printf("⚠️  Warning: could not read assets from %s: %v (continuing)\n", dstName, perr)
		assets = nil
	}

	// Get file info for size display
	info, _ := os.Stat(srcPath)
	size := info.Size()

	// Ask for confirmation (include assets plan if any)
	fmt.Printf("  File: %s\n", dstName)
	fmt.Printf("  Size: %d bytes (%.1f KB)\n", size, float64(size)/1024)
	fmt.Printf("  Destination: %s\n", inventoryDir)
	if _, err := os.Stat(dstPath); err == nil {
		fmt.Printf("  ⚠️  Warning: Will overwrite existing file\n")
	}
	profileName := strings.TrimSuffix(dstName, ".profile.toml")
	if len(assets) > 0 {
		printAssetsPlan(tempDir, filepath.Dir(srcPath), assets, profileName)
	}

	if !ConfirmAction(fmt.Sprintf("Summon %s?", dstName)) {
		fmt.Printf("⊘ Cancelled: %s\n", dstName)
		return outcomeCancelled, nil
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		return outcomeFailed, fmt.Errorf("failed to copy %s: %w", dstName, err)
	}
	fmt.Printf("✓ Summoned: %s\n", dstName)

	// Handle assets (git-only feature)
	if len(assets) > 0 {
		aCopied, aSkipped, aMissing, aBytes := copyAssetsList(tempDir, filepath.Dir(srcPath), assets, profileName)
		fmt.Printf("  Assets: copied=%d, skipped=%d, missing=%d, total=%.1f MB\n",
			aCopied, aSkipped, aMissing, float64(aBytes)/(1024*1024))
		log.Printf("Assets for %s: copied=%d, skipped=%d, missing=%d, bytes=%d", dstName, aCopied, aSkipped, aMissing, aBytes)
	}
	return outcomeSummoned, nil
}

// printAssetsPlan shows what the profile's declared assets would copy, so the
// user confirms with the full picture.
func printAssetsPlan(tempDir, srcDir string, assets []string, profileName string) {
	fmt.Printf("  Assets declared: %d\n", len(assets))
	fmt.Printf("  Plan (destination under ~/.config/drako/assets/%s/):\n", profileName)
	plans := planAssetsList(tempDir, srcDir, assets)
	totalPlannedBytes := int64(0)
	totalPlannedFiles := 0
	missingPlanned := 0
	for _, p := range plans {
		status := "file"
		if p.IsDir {
			status = "dir"
		}
		dest := filepath.Join("~/.config/drako/assets", profileName, p.DestRel)
		if p.Missing {
			fmt.Printf("    - %s (%s) -> %s [missing]\n", p.AssetRel, status, dest)
			missingPlanned++
		} else {
			if p.IsDir {
				fmt.Printf("    - %s (%s, %d files, %.1f MB) -> %s\n", p.AssetRel, status, p.FileCount, float64(p.Bytes)/(1024*1024), dest)
			} else {
				fmt.Printf("    - %s (%s, %.1f MB) -> %s\n", p.AssetRel, status, float64(p.Bytes)/(1024*1024), dest)
			}
			totalPlannedBytes += p.Bytes
			totalPlannedFiles += p.FileCount
		}
	}
	fmt.Printf("  Assets summary: planned_files=%d, planned_bytes=%.1f MB, missing=%d\n",
		totalPlannedFiles, float64(totalPlannedBytes)/(1024*1024), missingPlanned)
	fmt.Printf("  Note: No per-asset prompts. Missing assets will be warned and skipped. Limits: %d files, total ≤ %d MB, per-file ≤ %d MB.\n",
		assetMaxFileCount, assetMaxTotalBytes/(1024*1024), assetMaxFileBytes/(1024*1024))
}

// summonRepoSpec validates and copies one spec file on confirmation. Unlike
// profiles, a spec copy failure is reported and the summon continues.
func (s *Summoner) summonRepoSpec(srcPath, specsDir string) summonOutcome {
	dstName := filepath.Base(srcPath)
	dstPath := filepath.Join(specsDir, dstName)

	// Validate spec file content
	if err := validateSpecFile(srcPath); err != nil {
		fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
		return outcomeSkipped
	}

	info, _ := os.Stat(srcPath)
	size := info.Size()

	fmt.Printf("  File: %s\n", dstName)
	fmt.Printf("  Size: %d bytes\n", size)
	fmt.Printf("  Destination: %s\n", specsDir)
	if _, err := os.Stat(dstPath); err == nil {
		fmt.Printf("  ⚠️  Warning: Will overwrite existing file\n")
	}

	if !ConfirmAction(fmt.Sprintf("Summon spec %s?", dstName)) {
		fmt.Printf("⊘ Cancelled: %s\n", dstName)
		return outcomeCancelled
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		fmt.Printf("Failed to copy %s: %v\n", dstName, err)
		return outcomeFailed
	}
	fmt.Printf("✓ Summoned spec: %s\n", dstName)
	return outcomeSummoned
}

// summonFromHTTP downloads a single profile file from an HTTP/HTTPS URL,
// validates it, and moves it into the inventory.
func (s *Summoner) summonFromHTTP(sourceURL, inventoryDir string) error {
	// Extract filename from URL or use default
	filename := extractFilenameFromURL(sourceURL)
	if filename == "" || !strings.HasSuffix(filename, ".profile.toml") {
		filename = "personal.profile.toml"
	}

	// Validate filename before proceeding
	if err := validateFilename(filename); err != nil {
		return err
	}

	dstPath := filepath.Join(inventoryDir, filename)

	// Download to a temp path so validation happens before anything lands in
	// the inventory.
	tempFile, err := os.CreateTemp("", "drako-summon-*.toml")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	tempFile.Close() // close immediately, downloader will open
	defer os.Remove(tempPath)

	if err := s.Downloader.DownloadFile(sourceURL, tempPath); err != nil {
		return err
	}

	// Validate the downloaded file
	fmt.Printf("Validating profile...\n")
	if err := validateProfileFile(tempPath); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Move temp file to final destination
	// We need to copy/move it manually since it's cross-device potentially (tmp vs home)
	if err := copyFile(tempPath, dstPath); err != nil {
		return fmt.Errorf("failed to finalize file: %w", err)
	}

	fmt.Printf("✓ Summoned: %s\n", filename)
	log.Printf("Successfully summoned profile: %s from %s", filename, sourceURL)
	return nil
}

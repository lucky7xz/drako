package cli

// Summon fetches untrusted profile files into the config dir. A deck replaces
// the copy you already have, wherever it is; a new one is equipped while there
// is room, else it waits in the inventory (placeFor). Only the top level of a
// folder or repo is searched.
// This file is the orchestration: source classification, user confirmation,
// and the per-file summon flow. The moving parts live next door:
//   summon_transport.go — git/HTTP transports, URL parsing, file copying
//   summon_assets.go    — asset plan/copy with limits and path-safety checks
//   summon_validate.go  — validation of everything that arrives

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/glyph"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/summon"
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

	// Verification pins, already format-validated by the CLI layer.
	// SHA256 pins a single-file (HTTP) summon to a payload hash;
	// Rev pins a git summon to a full commit hash. Empty = unverified
	// (allowed for compatibility, warned about).
	SHA256 string
	Rev    string

	// Show lists every command a deck holds before the user confirms it.
	Show bool
}

// NewSummoner creates a new Summoner with real dependencies
func NewSummoner(configDir string) *Summoner {
	return &Summoner{
		ConfigDir:  configDir,
		Downloader: &summon.HTTPDownloader{},
		Cloner:     &summon.GitCloner{},
		UI:         &RealUI{},
	}
}

// Summon executes the summoning logic
func (s *Summoner) Summon(sourceURL string) error {
	if err := lair.MakeInventory(s.ConfigDir); err != nil {
		return fmt.Errorf("failed to create inventory directory: %w", err)
	}

	kind := summon.Classify(sourceURL)
	if kind == summon.Folder {
		return s.summonFromFolder(sourceURL)
	}

	if kind == summon.Git {
		if err := summon.CheckGitAddress(sourceURL); err != nil {
			return err
		}
		if s.SHA256 != "" {
			return fmt.Errorf("--sha256 pins single-file downloads; for git repositories use --rev <full-commit-hash>")
		}
		if summon.Unencrypted(sourceURL) && s.Rev == "" {
			return fmt.Errorf("%s is not encrypted: anyone on the network path could change what arrives.\nPin it with --rev <full-commit-hash>, or use an https:// or ssh address", sourceURL)
		}
		if err := s.Cloner.CheckGitAvailable(); err != nil {
			return err
		}

		if summon.IsSSHURL(sourceURL) {
			warnIfNoSSHKeys()
		}

		fmt.Printf("\nYou are about to clone a git repository:\n")
		fmt.Printf("  Source: %s\n", sourceURL)
		fmt.Printf("  Destination: %s (equipped; inventory/ once %d are equipped)\n", s.ConfigDir, equip.MaxEquipped)
		fmt.Printf("  Action: Find and copy the .profile.toml files at its top level\n\n")

		if s.Rev == "" {
			fmt.Printf("⚠️  Unverified summon: no --rev given. drako cannot confirm this deck\n")
			fmt.Printf("    is what its author published. To pin it:\n")
			fmt.Printf("    drako summon <url> --rev <full-40-hex-commit>\n\n")
		}

		if !s.UI.Confirm("Proceed with cloning?") {
			return fmt.Errorf("operation cancelled by user")
		}

		return s.summonFromGit(sourceURL)
	}

	// HTTP/HTTPS Download
	if s.Rev != "" {
		return fmt.Errorf("--rev pins git clones; for single-file downloads use --sha256 <hash>")
	}
	if summon.Unencrypted(sourceURL) && s.SHA256 == "" {
		return fmt.Errorf("%s is not encrypted: anyone on the network path could change what arrives.\nPin it with --sha256 <hash>, or use an https:// address", sourceURL)
	}
	filename := summon.ExtractFilenameFromURL(sourceURL)
	if filename == "" || !strings.HasSuffix(filename, ".profile.toml") {
		filename = "personal.profile.toml"
	}
	if err := summon.ValidateFilename(filename); err != nil {
		return err
	}

	dstPath, where, err := placeFor(s.ConfigDir, filename)
	if err != nil {
		return err
	}
	fmt.Printf("\nYou are about to download a profile:\n")
	fmt.Printf("  Source: %s\n", sourceURL)
	fmt.Printf("  Destination: %s\n", where)
	fmt.Printf("  Path: %s\n", dstPath)
	fmt.Println()

	if s.SHA256 == "" {
		fmt.Printf("⚠️  Unverified summon: no --sha256 given. drako cannot confirm this file\n")
		fmt.Printf("    is what its author published. To pin it:\n")
		fmt.Printf("    drako summon <url> --sha256 <hash>\n\n")
	}

	if !s.UI.Confirm("Proceed with download?") {
		return fmt.Errorf("operation cancelled by user")
	}

	return s.summonFromHTTP(sourceURL, dstPath)
}

// placeFor says where the summoned deck named file lands, as a path, and in
// words for the prompt: see equip.Place for the rule. It reads the disk each
// time, so decks summoned earlier in the same run count.
func placeFor(configDir, file string) (dstPath, where string, err error) {
	equipped, err := lair.List(configDir, equip.Equipped)
	if err != nil {
		return "", "", err
	}
	stashed, err := lair.List(configDir, equip.Inventory)
	if err != nil {
		return "", "", err
	}
	loc, name, replaces := equip.Place(equipped, stashed, file)
	return filepath.Join(lair.LocationDir(configDir, loc), name), destinationLabel(loc, name, replaces), nil
}

// destinationLabel says in words where a summoned deck lands: over the deck
// you already have, equipped or stashed, or as a new deck — equipped, or in
// the inventory once MaxEquipped are.
func destinationLabel(loc equip.Location, name string, replaces bool) string {
	switch {
	case replaces && loc == equip.Inventory:
		return "replaces " + name + " in the inventory"
	case replaces:
		return "replaces your equipped " + name
	case loc == equip.Inventory:
		return fmt.Sprintf("inventory (%d decks are equipped)", equip.MaxEquipped)
	}
	return "equipped"
}

// gitHeadFn and gitCheckoutFn are package vars so tests can stub them; the
// real ones shell out to git.
var (
	gitHeadFn     = summon.GitHead
	gitCheckoutFn = summon.GitCheckout
)

// warnIfNoSSHKeys says so when none of the usual SSH keys is set up.
func warnIfNoSSHKeys() {
	sshDir, missing := summon.MissingSSHKeys()
	if !missing {
		return
	}
	fmt.Printf("⚠️  Warning: No SSH keys found in %s\n", sshDir)
	fmt.Printf("   For private repositories, you may need to:\n")
	fmt.Printf("   1. Generate SSH keys: ssh-keygen -t ed25519 -C \"your_email@example.com\"\n")
	fmt.Printf("   2. Add the public key to your Git hosting service (GitHub/GitLab/etc.)\n")
	fmt.Printf("   3. Or use HTTPS URL instead: https://github.com/user/repo.git\n\n")
}

// validateProfile is summon.ValidateProfile, saying so when the deck is
// unusually large.
func validateProfile(f summon.File) (summon.File, error) {
	if size := len(f.Data); size > summon.ProfileWarnSize {
		fmt.Printf("⚠️  Warning: Profile is unusually large (%d KB). Validating...\n", size/1024)
	}
	return summon.ValidateProfile(f)
}

// installProfile hands the checked deck f to lair, which puts it where
// dstPath says: equipped (the config dir itself) or the inventory.
func installProfile(configDir string, f summon.File, dstPath string) error {
	loc := equip.Equipped
	if filepath.Dir(dstPath) != filepath.Clean(configDir) {
		loc = equip.Inventory
	}
	return lair.InstallProfile(configDir, loc, filepath.Base(dstPath), bytes.NewReader(f.Data))
}

// sameFile reports whether the deck at dstPath, in the config dir, and the
// checked file f are one file on disk: the deck is a link to it.
func sameFile(configDir string, f summon.File, dstPath string) bool {
	rel, err := filepath.Rel(configDir, dstPath)
	return err == nil && f.Info != nil && lair.SameFile(configDir, rel, f.Info)
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
func (s *Summoner) summonFromGit(repoURL string) error {
	q, err := summon.NewQuarantine()
	if err != nil {
		return err
	}
	defer q.Close()
	tempDir := q.Dir

	// Clone the repository using injected Cloner
	fmt.Printf("Cloning repository...\n")
	if err := s.Cloner.CloneRepo(repoURL, tempDir); err != nil {
		return err
	}

	// Check out and verify the pinned commit before any file is looked at. Git's
	// content addressing means one matching hash covers every file in the tree.
	if s.Rev != "" {
		if err := gitCheckoutFn(tempDir, s.Rev); err != nil {
			return err
		}
		head, err := gitHeadFn(tempDir)
		if err != nil {
			return err
		}
		if !strings.EqualFold(head, s.Rev) {
			return fmt.Errorf("commit MISMATCH\n  pinned %s\n  cloned %s\nnothing was summoned", strings.ToLower(s.Rev), head)
		}
		fmt.Printf("✓ Verified commit: HEAD = %.12s…\n", head)
	}
	return s.summonFromDir(tempDir, repoURL)
}

// summonFromFolder summons the decks in a folder on disk, for authors who
// haven't published them; it reads the folder in place.
func (s *Summoner) summonFromFolder(dir string) error {
	if s.SHA256 != "" || s.Rev != "" {
		return fmt.Errorf("--sha256 and --rev verify downloads; a folder on disk has nothing to pin")
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	// Inside the config dir, or around it, summon would copy decks onto themselves.
	inside, _ := summon.IsPathWithinBase(s.ConfigDir, root)
	around, _ := summon.IsPathWithinBase(root, s.ConfigDir)
	if inside || around {
		return fmt.Errorf("%s is or holds drako's config dir; summon from your deck's own folder", root)
	}

	fmt.Printf("\nYou are about to summon from a folder:\n")
	fmt.Printf("  Source: %s\n", root)
	fmt.Printf("  Destination: %s (equipped; inventory/ once %d are equipped)\n", s.ConfigDir, equip.MaxEquipped)
	return s.summonFromDir(root, root)
}

// summonFromDir offers every profile and spec at the top level of root, a
// clone or a local folder; source names root in messages.
func (s *Summoner) summonFromDir(root, source string) error {
	profileFiles, specFiles, err := summon.FindFiles(root)
	if err != nil {
		return fmt.Errorf("failed to search for files: %w", err)
	}
	if len(profileFiles) == 0 && len(specFiles) == 0 {
		return fmt.Errorf("no .profile.toml or .spec.toml files found in %s", source)
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
		outcome, err := s.summonRepoProfile(srcPath, root)
		if err != nil {
			return err
		}
		tally(outcome)
	}

	// 2. Process Spec Files
	if len(specFiles) > 0 {
		specsDir := lair.SpecsDir(s.ConfigDir)
		if err := lair.MakeSpecs(s.ConfigDir); err != nil {
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
		return fmt.Errorf("no valid items found in %s (%d skipped)", source, skipped)
	}

	log.Printf("Successfully summoned %d item(s) from %s (skipped: %d, cancelled: %d)", summoned, source, skipped, cancelled)
	return nil
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
func (s *Summoner) summonRepoProfile(srcPath, tempDir string) (summonOutcome, error) {
	dstName := filepath.Base(srcPath)

	// Validate filename
	if err := summon.ValidateFilename(dstName); err != nil {
		fmt.Printf("⚠️  Skipping %q: %v\n", dstName, err)
		return outcomeSkipped, nil
	}

	// Validate file content before copying
	fmt.Printf("\nValidating %s...\n", dstName)
	f, err := summon.Read(srcPath)
	if err == nil {
		f, err = validateProfile(f)
	}
	if err != nil {
		fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
		return outcomeSkipped, nil
	}
	assets := summon.ProfileAssets(f.Profile)

	dstPath, where, err := placeFor(s.ConfigDir, dstName)
	if err != nil {
		return outcomeFailed, err
	}

	size := len(f.Data)

	// Ask for confirmation (include assets plan if any)
	fmt.Printf("  File: %s\n", dstName)
	fmt.Printf("  Size: %d bytes (%.1f KB)\n", size, float64(size)/1024)
	fmt.Printf("  Destination: %s\n", where)
	profileName := strings.TrimSuffix(filepath.Base(dstPath), ".profile.toml")
	if len(assets) > 0 {
		printAssetsPlan(tempDir, filepath.Dir(srcPath), assets, profileName)
	}
	if s.Show {
		if err := printCommands(f); err != nil {
			fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
			return outcomeSkipped, nil
		}
	}

	if !s.UI.Confirm(fmt.Sprintf("Summon %s?", dstName)) {
		fmt.Printf("⊘ Cancelled: %s\n", dstName)
		return outcomeCancelled, nil
	}

	if sameFile(s.ConfigDir, f, dstPath) {
		// The deck is a link to this very file: it is already in place.
		fmt.Printf("✓ Already in place: %s\n", dstName)
	} else if err := installProfile(s.ConfigDir, f, dstPath); err != nil {
		return outcomeFailed, fmt.Errorf("failed to copy %s: %w", dstName, err)
	} else {
		fmt.Printf("✓ Summoned: %s\n", dstName)
	}

	// Handle assets (git-only feature)
	if len(assets) > 0 {
		aCopied, aSkipped, aMissing, aBytes := summon.CopyAssets(tempDir, filepath.Dir(srcPath), assets, summon.AssetSink{
			Install: func(rel string, r io.Reader, executable bool) error {
				return lair.InstallAsset(s.ConfigDir, profileName, rel, r, executable)
			},
			Large: func(name string, size int64, inFolder bool) {
				if inFolder {
					fmt.Printf("  ⚠️  Large asset file: %s (%.1f MB)\n", name, float64(size)/(1024*1024))
				} else {
					fmt.Printf("  ⚠️  Large asset: %s (%.1f MB)\n", name, float64(size)/(1024*1024))
				}
			},
		})
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
	plans := summon.PlanAssets(tempDir, srcDir, assets)
	totalPlannedBytes := int64(0)
	totalPlannedFiles := 0
	totalExecutable := 0
	missingPlanned := 0
	refusedPlanned := 0
	for _, p := range plans {
		status := "file"
		if p.IsDir {
			status = "dir"
		}
		dest := filepath.Join("~/.config/drako/assets", profileName, p.DestRel)
		switch {
		case p.Refused:
			fmt.Printf("    - %s [refused: outside the repository]\n", p.AssetRel)
			refusedPlanned++
		case p.Missing:
			fmt.Printf("    - %s (%s) -> %s [missing]\n", p.AssetRel, status, dest)
			missingPlanned++
		case p.IsDir:
			fmt.Printf("    - %s (%s, %d files%s, %.1f MB) -> %s\n", p.AssetRel, status, p.FileCount, executableNote(p), float64(p.Bytes)/(1024*1024), dest)
		default:
			fmt.Printf("    - %s (%s%s, %.1f MB) -> %s\n", p.AssetRel, status, executableNote(p), float64(p.Bytes)/(1024*1024), dest)
		}
		if !p.Refused && !p.Missing {
			totalPlannedBytes += p.Bytes
			totalPlannedFiles += p.FileCount
			totalExecutable += p.Executable
		}
	}
	fmt.Printf("  Assets summary: planned_files=%d, executable=%d, planned_bytes=%.1f MB, missing=%d, refused=%d\n",
		totalPlannedFiles, totalExecutable, float64(totalPlannedBytes)/(1024*1024), missingPlanned, refusedPlanned)
	fmt.Printf("  Note: No per-asset prompts. Missing and refused assets are skipped. Limits: %d files, total ≤ %d MB, per-file ≤ %d MB.\n",
		summon.AssetMaxFileCount, summon.AssetMaxTotalBytes/(1024*1024), summon.AssetMaxFileBytes/(1024*1024))
}

// executableNote says how many of p's files arrive executable, if any.
func executableNote(p summon.AssetPlan) string {
	switch {
	case p.Executable == 0:
		return ""
	case p.IsDir:
		return fmt.Sprintf(", %d executable", p.Executable)
	}
	return ", executable"
}

// printCommands lists every command the checked deck f holds, as written and
// for every platform, so the user reads them before confirming (--show).
func printCommands(f summon.File) error {
	cmds, err := f.Commands()
	if err != nil {
		return err
	}
	if shell := f.Profile.Shell; shell != nil {
		fmt.Printf("  Shell: %s\n", visible(*shell))
	}
	fmt.Printf("  Commands (%d), as written:\n", len(cmds))
	for _, c := range cmds {
		where := c.Cell
		if c.Item != "" {
			where += " › " + c.Item
		}
		if c.Platform != "" {
			where += " [" + c.Platform + "]"
		}
		fmt.Printf("    %s\n", visible(where))
		for _, line := range strings.Split(visible(c.Text), "\n") {
			fmt.Printf("      %s\n", line)
		}
	}
	return nil
}

// visible escapes what a terminal would not show as itself (control
// characters such as ESC or carriage return, direction overrides, odd
// spaces), so a deck cannot hide part of a command from the reader. Line
// breaks and tabs stay.
func visible(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || strconv.IsPrint(r) {
			b.WriteRune(r)
			continue
		}
		q := strconv.QuoteRune(r)
		b.WriteString(q[1 : len(q)-1])
	}
	return b.String()
}

// summonRepoSpec validates and copies one spec file on confirmation. Unlike
// profiles, a spec copy failure is reported and the summon continues.
func (s *Summoner) summonRepoSpec(srcPath, specsDir string) summonOutcome {
	dstName := filepath.Base(srcPath)

	if err := glyph.CheckName(dstName); err != nil {
		fmt.Printf("⚠️  Skipping %q: file name %v\n", dstName, err)
		return outcomeSkipped
	}
	f, err := summon.Read(srcPath)
	if err == nil {
		err = summon.ValidateSpec(f)
	}
	if err != nil {
		fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
		return outcomeSkipped
	}
	size := len(f.Data)

	fmt.Printf("  File: %s\n", dstName)
	fmt.Printf("  Size: %d bytes\n", size)
	fmt.Printf("  Destination: %s\n", specsDir)
	if lair.Exists(s.ConfigDir, filepath.Join(lair.SpecsDirName, dstName)) {
		fmt.Printf("  ⚠️  Warning: Will overwrite existing file\n")
	}

	if !s.UI.Confirm(fmt.Sprintf("Summon spec %s?", dstName)) {
		fmt.Printf("⊘ Cancelled: %s\n", dstName)
		return outcomeCancelled
	}

	if err := lair.InstallSpec(s.ConfigDir, dstName, bytes.NewReader(f.Data)); err != nil {
		fmt.Printf("Failed to copy %s: %v\n", dstName, err)
		return outcomeFailed
	}
	fmt.Printf("✓ Summoned spec: %s\n", dstName)
	return outcomeSummoned
}

// summonFromHTTP downloads a single profile file from an HTTP/HTTPS URL,
// validates it, and copies it to dstPath.
func (s *Summoner) summonFromHTTP(sourceURL, dstPath string) error {
	filename := filepath.Base(dstPath)

	// Download into a quarantine so validation happens before anything lands
	// in the config dir.
	q, err := summon.NewQuarantine()
	if err != nil {
		return err
	}
	defer q.Close()
	tempPath := filepath.Join(q.Dir, "download.toml")

	fmt.Printf("Downloading from %s...\n", sourceURL)
	if err := s.Downloader.DownloadFile(sourceURL, tempPath); err != nil {
		return err
	}

	f, err := summon.Read(tempPath)
	if err != nil {
		return err
	}

	// Verify the pin before the TOML parser ever touches the bytes: the
	// cheap check gates the risky one.
	if s.SHA256 != "" {
		fmt.Printf("Verifying sha256...\n")
		if err := summon.VerifySHA256(f.Data, s.SHA256); err != nil {
			return err
		}
		fmt.Printf("✓ sha256 verified\n")
	}

	fmt.Printf("Validating profile...\n")
	f, err = validateProfile(f)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if s.Show {
		if err := printCommands(f); err != nil {
			return err
		}
		if !s.UI.Confirm(fmt.Sprintf("Summon %s?", filename)) {
			return fmt.Errorf("operation cancelled by user")
		}
	}

	if err := installProfile(s.ConfigDir, f, dstPath); err != nil {
		return fmt.Errorf("failed to finalize file: %w", err)
	}

	fmt.Printf("✓ Summoned: %s\n", filename)
	log.Printf("Successfully summoned profile: %s from %s", filename, sourceURL)
	return nil
}

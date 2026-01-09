package cli

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/config"
)

// Asset copy limits
const (
	assetWarnSizeBytes = 1 * 1024 * 1024  // 1 MB warn
	assetMaxFileBytes  = 5 * 1024 * 1024  // 5 MB per file hard limit
	assetMaxTotalBytes = 50 * 1024 * 1024 // 50 MB total hard limit
	assetMaxFileCount  = 500              // safety cap
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

// RealUI implements UIInterface using stdin/stdout
type RealUI struct{}

func (ui *RealUI) Confirm(prompt string) bool {
	return ConfirmAction(prompt)
}

// ConfirmAction prompts the user to confirm an action
func ConfirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// SummonProfile Entry Point (Legacy Wrapper)
func SummonProfile(sourceURL, configDir string) error {
	summoner := NewSummoner(configDir)
	return summoner.Summon(sourceURL)
}

// Summon executes the summoning logic
func (s *Summoner) Summon(sourceURL string) error {
	inventoryDir := filepath.Join(s.ConfigDir, "inventory")
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

// isGitURL checks if the URL points to a git repository.
func isGitURL(urlStr string) bool {
	// SSH format: git@github.com:user/repo.git
	if strings.HasPrefix(urlStr, "git@") {
		return true
	}
	// Git protocol: git://github.com/user/repo.git
	if strings.HasPrefix(urlStr, "git://") {
		return true
	}
	// URLs ending with .git are repositories
	if strings.HasSuffix(urlStr, ".git") {
		return true
	}
	return false
}

// isSSHURL checks if the URL uses SSH protocol
func isSSHURL(urlStr string) bool {
	return strings.HasPrefix(urlStr, "git@") || strings.HasPrefix(urlStr, "ssh://")
}

// warnIfNoSSHKeys checks if SSH keys are configured and warns if not
func warnIfNoSSHKeys() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	sshDir := filepath.Join(home, ".ssh")

	// Check for common SSH key files
	keyFiles := []string{"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa"}
	hasKeys := false

	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)
		if _, err := os.Stat(keyPath); err == nil {
			hasKeys = true
			break
		}
	}

	if !hasKeys {
		fmt.Printf("⚠️  Warning: No SSH keys found in %s\n", sshDir)
		fmt.Printf("   For private repositories, you may need to:\n")
		fmt.Printf("   1. Generate SSH keys: ssh-keygen -t ed25519 -C \"your_email@example.com\"\n")
		fmt.Printf("   2. Add the public key to your Git hosting service (GitHub/GitLab/etc.)\n")
		fmt.Printf("   3. Or use HTTPS URL instead: https://github.com/user/repo.git\n\n")
	}
}

// GitCloner implements RepoCloner using exec.Command
type GitCloner struct{}

func (c *GitCloner) CheckGitAvailable() error {
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	return nil
}

func (c *GitCloner) CloneRepo(url, destDir string) error {
	fmt.Printf("Cloning repository...\n")
	// Use -- to prevent argument injection if repoURL starts with a hyphen
	cmd := exec.Command("git", "clone", "--", url, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	return nil
}

// summonFromGit clones a profile repository
func (s *Summoner) summonFromGit(repoURL, inventoryDir string) error {
	// Extract filename from URL or use default
	filename := extractFilenameFromURL(repoURL)
	if !strings.HasSuffix(filename, ".profile.toml") {
		filename = "personal.profile.toml"
	}

	// Create a temporary directory for cloning
	tempDir := filepath.Join(inventoryDir, ".summon-temp")
	defer os.RemoveAll(tempDir)

	// Clone the repository using injected Cloner
	if err := s.Cloner.CloneRepo(repoURL, tempDir); err != nil {
		return err
	}

	// Find .profile.toml files in the repo
	var profileFiles []string
	var specFiles []string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
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
	if err != nil {
		return fmt.Errorf("failed to search for files: %w", err)
	}

	if len(profileFiles) == 0 && len(specFiles) == 0 {
		return fmt.Errorf("no .profile.toml or .spec.toml files found in repository")
	}

	// Show what was found
	if len(profileFiles) > 0 {
		fmt.Printf("\nFound %d profile file(s) in repository:\n", len(profileFiles))
		for _, srcPath := range profileFiles {
			fmt.Printf("  - %s\n", filepath.Base(srcPath))
		}
	}
	if len(specFiles) > 0 {
		fmt.Printf("\nFound %d spec file(s) in repository:\n", len(specFiles))
		for _, srcPath := range specFiles {
			fmt.Printf("  - %s\n", filepath.Base(srcPath))
		}
	}
	fmt.Println()

	// Copy and validate all profile files to config directory
	summoned := 0
	skipped := 0
	cancelled := 0

	// 1. Process Profile Files
	for _, srcPath := range profileFiles {
		dstName := filepath.Base(srcPath)
		dstPath := filepath.Join(inventoryDir, dstName)

		// Validate filename
		if err := validateFilename(dstName); err != nil {
			fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
			skipped++
			continue
		}

		// Safety Check: Equipped Collision
		// Ensure this profile doesn't conflict with what is actively equipped in root
		if err := s.checkEquippedCollision(dstName); err != nil {
			fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
			skipped++
			continue
		}

		// Validate file content before copying
		fmt.Printf("\nValidating %s...\n", dstName)
		if err := validateProfileFile(srcPath); err != nil {
			fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
			skipped++
			continue
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

		// Check if destination exists
		overwriting := false
		if _, err := os.Stat(dstPath); err == nil {
			overwriting = true
		}

		// Ask for confirmation (include assets plan if any)
		fmt.Printf("  File: %s\n", dstName)
		fmt.Printf("  Size: %d bytes (%.1f KB)\n", size, float64(size)/1024)
		fmt.Printf("  Destination: %s\n", inventoryDir)
		if overwriting {
			fmt.Printf("  ⚠️  Warning: Will overwrite existing file\n")
		}
		if len(assets) > 0 {
			fmt.Printf("  Assets declared: %d\n", len(assets))
			// Determine profile name for asset destination
			profileName := strings.TrimSuffix(dstName, ".profile.toml")
			fmt.Printf("  Plan (destination under ~/.config/drako/assets/%s/):\n", profileName)
			plans := planAssetsList(tempDir, filepath.Dir(srcPath), assets, profileName)
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

		if !ConfirmAction(fmt.Sprintf("Summon %s?", dstName)) {
			fmt.Printf("⊘ Cancelled: %s\n", dstName)
			cancelled++
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", dstName, err)
		}
		fmt.Printf("✓ Summoned: %s\n", dstName)
		summoned++

		// Handle assets (git-only feature)
		if len(assets) > 0 {
			// Derive profile name from the destination filename (e.g. "my-profile.profile.toml" -> "my-profile")
			// We use dstName here because that's the final name in the inventory.
			profileName := strings.TrimSuffix(dstName, ".profile.toml")

			// We need to pass the profile name to copyAssetsList so it knows where to put them.
			aCopied, aSkipped, aMissing, aBytes := copyAssetsList(tempDir, filepath.Dir(srcPath), assets, profileName)
			fmt.Printf("  Assets: copied=%d, skipped=%d, missing=%d, total=%.1f MB\n",
				aCopied, aSkipped, aMissing, float64(aBytes)/(1024*1024))
			log.Printf("Assets for %s: copied=%d, skipped=%d, missing=%d, bytes=%d", dstName, aCopied, aSkipped, aMissing, aBytes)
		}
	}

	// 2. Process Spec Files
	if len(specFiles) > 0 {
		configDir, _ := config.GetConfigDir()
		specsDir := filepath.Join(configDir, "specs")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			fmt.Printf("⚠️  Failed to create specs directory: %v\n", err)
		} else {
			fmt.Printf("\nProcessing spec files...\n")
			for _, srcPath := range specFiles {
				dstName := filepath.Base(srcPath)
				dstPath := filepath.Join(specsDir, dstName)

				// Validate spec file content
				if err := validateSpecFile(srcPath); err != nil {
					fmt.Printf("⚠️  Skipping %s: %v\n", dstName, err)
					skipped++
					continue
				}

				info, _ := os.Stat(srcPath)
				size := info.Size()

				overwriting := false
				if _, err := os.Stat(dstPath); err == nil {
					overwriting = true
				}

				fmt.Printf("  File: %s\n", dstName)
				fmt.Printf("  Size: %d bytes\n", size)
				fmt.Printf("  Destination: %s\n", specsDir)
				if overwriting {
					fmt.Printf("  ⚠️  Warning: Will overwrite existing file\n")
				}

				if !ConfirmAction(fmt.Sprintf("Summon spec %s?", dstName)) {
					fmt.Printf("⊘ Cancelled: %s\n", dstName)
					cancelled++
					continue
				}

				if err := copyFile(srcPath, dstPath); err != nil {
					fmt.Printf("Failed to copy %s: %v\n", dstName, err)
					continue
				}
				fmt.Printf("✓ Summoned spec: %s\n", dstName)
				summoned++
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

// readAssetsFromProfile parses a profile file and returns declared assets (relative paths)
func readAssetsFromProfile(profilePath string) ([]string, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, err
	}
	var profile config.ProfileFile
	if _, err := toml.Decode(string(data), &profile); err != nil {
		return nil, err
	}
	if profile.Assets == nil {
		return nil, nil
	}
	var out []string
	for _, raw := range *profile.Assets {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		// Clean path to normalize separators and remove leading "./"
		s = filepath.Clean(s)
		s = strings.TrimPrefix(s, "./")
		out = append(out, s)
	}
	return out, nil
}

// copyAssetsList copies a list of assets (files or directories) from the cloned repo to configDir.
// - repoRoot: tempDir where repo was cloned
// - profileDir: directory of the profile file (assets are relative to this)
// - profileName: name of the profile (used for subfolder isolation)
// Returns counts of copied/skipped/missing and total bytes copied.
func copyAssetsList(repoRoot, profileDir string, assets []string, profileName string) (int, int, int, int64) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		log.Printf("assets: could not resolve config dir: %v", err)
		return 0, 0, len(assets), 0
	}
	var copied, skipped, missing int
	var totalBytes int64
	var fileCount int

	for _, rel := range assets {
		// Resolve asset path relative to profileDir
		cleanRel, safe := cleanAssetRel(rel)
		if !safe {
			log.Printf("assets: skipping unsafe relative path %s", rel)
			skipped++
			continue
		}
		src := filepath.Join(profileDir, cleanRel)
		// Ensure src is within repoRoot
		ok, err := isPathWithinBase(repoRoot, src)
		if err != nil || !ok {
			log.Printf("assets: skipping unsafe path %s (err=%v)", rel, err)
			skipped++
			continue
		}
		// Destination is the assets/ directory + profile name + original relative path
		// e.g. ~/.config/drako/assets/my-profile/script.sh
		dst := filepath.Join(configDir, "assets", profileName, cleanRel)

		info, statErr := os.Stat(src)
		if statErr != nil {
			log.Printf("assets: missing %s", rel)
			missing++
			continue
		}
		if info.IsDir() {
			c, s, m, _ := copyDirWithLimits(src, dst, &fileCount, &totalBytes)
			copied += c
			skipped += s
			missing += m
			continue
		}
		// Single file copy with limits
		if allow, _ := checkAssetFileAllowed(info.Size(), fileCount, totalBytes); !allow {
			log.Printf("assets: skipping (limits) %s", rel)
			skipped++
			continue
		}
		if info.Size() > assetWarnSizeBytes {
			fmt.Printf("  ⚠️  Large asset: %s (%.1f MB)\n", rel, float64(info.Size())/(1024*1024))
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			log.Printf("assets: mkdir failed for %s: %v", dst, err)
			skipped++
			continue
		}
		if err := copyFile(src, dst); err != nil {
			log.Printf("assets: copy failed %s -> %s: %v", src, dst, err)
			skipped++
			continue
		}
		copied++
		fileCount++
		totalBytes += info.Size()
	}
	return copied, skipped, missing, totalBytes
}

// assetPlanItem describes what will be copied for an asset
type assetPlanItem struct {
	AssetRel  string
	DestRel   string
	IsDir     bool
	FileCount int
	Bytes     int64
	Missing   bool
}

// planAssetsList enumerates assets to present a copy plan before confirmation
func planAssetsList(repoRoot, profileDir string, assets []string, profileName string) []assetPlanItem {
	configDir, _ := config.GetConfigDir()
	var plans []assetPlanItem
	for _, rel := range assets {
		cleanRel, safe := cleanAssetRel(rel)
		if !safe {
			plans = append(plans, assetPlanItem{AssetRel: rel, DestRel: cleanRel, Missing: true})
			continue
		}
		src := filepath.Join(profileDir, cleanRel)
		ok, err := isPathWithinBase(repoRoot, src)
		if err != nil || !ok {
			plans = append(plans, assetPlanItem{AssetRel: rel, DestRel: cleanRel, Missing: true})
			continue
		}
		info, statErr := os.Stat(src)
		if statErr != nil {
			plans = append(plans, assetPlanItem{AssetRel: rel, DestRel: cleanRel, Missing: true})
			continue
		}
		if info.IsDir() {
			// Walk dir to count files/bytes
			var files int
			var bytes int64
			filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				fi, e := d.Info()
				if e != nil {
					return nil
				}
				files++
				bytes += fi.Size()
				return nil
			})
			plans = append(plans, assetPlanItem{
				AssetRel:  rel,
				DestRel:   cleanRel,
				IsDir:     true,
				FileCount: files,
				Bytes:     bytes,
				Missing:   false,
			})
		} else {
			plans = append(plans, assetPlanItem{
				AssetRel:  rel,
				DestRel:   cleanRel,
				IsDir:     false,
				FileCount: 1,
				Bytes:     info.Size(),
				Missing:   false,
			})
		}
		_ = configDir // only used to hint destination in UI printing (done above)
	}
	return plans
}

// cleanAssetRel normalizes an asset relative path and ensures it is safe (no abs, no parent escapes)
func cleanAssetRel(rel string) (string, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	if filepath.IsAbs(rel) {
		return "", false
	}
	clean := filepath.Clean(rel)
	clean = strings.TrimPrefix(clean, "./")
	// Reject parent traversal
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", false
	}
	return clean, true
}

// copyDirWithLimits recursively copies files from srcDir to dstDir with size/count limits
func copyDirWithLimits(srcDir, dstDir string, fileCount *int, totalBytes *int64) (int, int, int, int64) {
	copied := 0
	skipped := 0
	missing := 0
	var copiedBytes int64
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, serr := d.Info()
		if serr != nil {
			return serr
		}
		if allow, _ := checkAssetFileAllowed(info.Size(), *fileCount, *totalBytes); !allow {
			skipped++
			return nil
		}
		rel, rerr := filepath.Rel(srcDir, path)
		if rerr != nil {
			skipped++
			return nil
		}
		dst := filepath.Join(dstDir, rel)
		if info.Size() > assetWarnSizeBytes {
			fmt.Printf("  ⚠️  Large asset file: %s (%.1f MB)\n", rel, float64(info.Size())/(1024*1024))
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			skipped++
			return nil
		}
		if err := copyFile(path, dst); err != nil {
			skipped++
			return nil
		}
		*fileCount++
		*totalBytes += info.Size()
		copiedBytes += info.Size()
		copied++
		return nil
	})
	if err != nil {
		log.Printf("assets: copy dir error %s: %v", srcDir, err)
	}
	return copied, skipped, missing, copiedBytes
}

// checkAssetFileAllowed enforces per-file and aggregate limits
func checkAssetFileAllowed(size int64, fileCount int, totalBytes int64) (bool, int64) {
	if size > assetMaxFileBytes {
		return false, totalBytes
	}
	if fileCount+1 > assetMaxFileCount {
		return false, totalBytes
	}
	if totalBytes+size > assetMaxTotalBytes {
		return false, totalBytes
	}
	return true, totalBytes + size
}

// isPathWithinBase checks if target is within base after resolving symlinks
func isPathWithinBase(base, target string) (bool, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false, err
	}

	// Use EvalSymlinks to resolve the true final path.
	// Example: If the repo contains a file 'script.sh' which is actually a symlink
	// pointing to '../../../../etc/passwd', EvalSymlinks reveals that true path.
	// We then check if that resolved path is still inside the repo folder.
	// If it points outside, we reject it to prevent stealing system files.
	targetResolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		// If it doesn't exist, EvalSymlinks fails. We can fallback to Abs + Clean if we are creating it,
		// but for existing source files in the repo, it MUST exist.
		// If checking destination, it might not exist yet.
		// However, this function is primarily used to check if a SOURCE file inside the repo
		// is actually safe to copy (i.e. it doesn't point outside the repo).
		return false, err
	}

	targetAbs, err := filepath.Abs(targetResolved)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false, err
	}
	// Must not start with ".."
	return !strings.HasPrefix(rel, ".."), nil
}

// HTTPDownloader implements FileDownloader using http package
type HTTPDownloader struct{}

// DownloadFile downloads a file from URL to destPath
func (d *HTTPDownloader) DownloadFile(sourceURL, dstPath string) error {
	fmt.Printf("Downloading from %s...\n", sourceURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "drako-summon/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	if resp.ContentLength > profileMaxSize {
		return fmt.Errorf("file too large (%d bytes, max %d bytes)", resp.ContentLength, profileMaxSize)
	}

	// Create temporary file first
	tempPath := dstPath + ".tmp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	written, err := io.CopyN(tempFile, resp.Body, profileMaxSize+1)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if written > profileMaxSize {
		return fmt.Errorf("file too large (>%d bytes). This is not a valid profile", profileMaxSize)
	}
	return nil
}

// summonFromHTTP downloads a profile file from an HTTP/HTTPS URL
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

	// Use the injected downloader
	// We download to a temp location handled by the downloader or here?
	// The RealDownloader above downloads to dstPath.tmp. Wait.
	// The current impl does tmp file handling inside.
	// Let's rely on the downloader to do the heavy lifting of network IO.
	// But validation happens AFTER download? Yes.

	// To keep it simple, we let the downloader download to the final path? No, validation.
	// Let's download to a temp path ourselves.

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

// Helper to copy file (since os.Rename might fail across partitions)
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// extractFilenameFromURL extracts a filename from a URL
func extractFilenameFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// Get the last segment of the path
	path := u.Path
	if path == "" || path == "/" {
		return ""
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Get the last component
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		// Remove query parameters if present
		if idx := strings.Index(filename, "?"); idx != -1 {
			filename = filename[:idx]
		}
		return filename
	}

	return ""
}

// Profile size limits
const (
	profileWarnSize = 500 * 1024      // 500KB - warn if larger
	profileMaxSize  = 2 * 1024 * 1024 // 2MB - reject if larger
)

// validateProfileFile checks if a file is a valid drako profile
func validateProfileFile(path string) error {
	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	size := info.Size()
	if size > profileMaxSize {
		return fmt.Errorf("file too large (%d bytes, max %d bytes). This is not a valid profile", size, profileMaxSize)
	}

	if size > profileWarnSize {
		fmt.Printf("⚠️  Warning: Profile is unusually large (%d KB). Validating...\n", size/1024)
	}

	// Read and parse as TOML
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Try to parse as ProfileFile (what drako expects)
	var profile config.ProfileFile
	if _, err := toml.Decode(string(data), &profile); err != nil {
		return fmt.Errorf("invalid TOML format: %w", err)
	}

	// Check if it has at least one profile-related field
	if ok, missing := config.ValidateProfileFile(profile); !ok {
		return fmt.Errorf("file contains no profile settings (missing %s)", strings.Join(missing, ", "))
	}

	return nil
}

// validateFilename checks if the filename is a valid profile name
func validateFilename(filename string) error {
	if !strings.HasSuffix(filename, ".profile.toml") {
		return fmt.Errorf("filename must end with .profile.toml (got: %s)", filename)
	}
	return nil
}

// validateSpecFile checks if a file is a valid drako spec
func validateSpecFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > profileMaxSize {
		return fmt.Errorf("file too large")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Basic TOML validation
	var temp map[string]interface{}
	if _, err := toml.Decode(string(data), &temp); err != nil {
		return fmt.Errorf("invalid TOML: %w", err)
	}

	// Spec files should expect a 'profiles' key (list of strings)
	if profiles, ok := temp["profiles"]; ok {
		if _, isList := profiles.([]interface{}); !isList {
			return fmt.Errorf("missing or invalid 'profiles' list")
		}
	} else {
		return fmt.Errorf("missing 'profiles' key")
	}

	return nil
}

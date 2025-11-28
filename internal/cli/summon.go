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

// SummonProfile downloads a profile from a URL and saves it to the inventory directory.
// Supports:
//   - HTTP/HTTPS URLs (raw file downloads) - user provides file URL
//   - Git repository URLs (clones whole repo) - user provides repo URL
func SummonProfile(sourceURL, configDir string) error {
	// Inventory directory is where summoned profiles go
	inventoryDir := filepath.Join(configDir, "inventory")
	if err := os.MkdirAll(inventoryDir, 0o755); err != nil {
		return fmt.Errorf("failed to create inventory directory: %w", err)
	}

	// Check if it's a git repository (user wants whole repo)
	if isGitURL(sourceURL) {
		// Check if git is available
		if err := checkGitAvailable(); err != nil {
			return err
		}

		// Warn if SSH URL but no SSH keys
		if isSSHURL(sourceURL) {
			warnIfNoSSHKeys()
		}

		// Confirm before cloning
		fmt.Printf("\nYou are about to clone a git repository:\n")
		fmt.Printf("  Source: %s\n", sourceURL)
		fmt.Printf("  Destination: %s\n", inventoryDir)
		fmt.Printf("  Action: Find and copy all .profile.toml files\n\n")

		if !ConfirmAction("Proceed with cloning?") {
			return fmt.Errorf("operation cancelled by user")
		}

		return summonFromGit(sourceURL, inventoryDir)
	}

	// Otherwise, treat as HTTP/HTTPS file download (user wants single file)
	// Extract filename for confirmation
	filename := extractFilenameFromURL(sourceURL)
	if filename == "" || !strings.HasSuffix(filename, ".profile.toml") {
		filename = "personal.profile.toml"
	}

	fmt.Printf("\nYou are about to download a profile:\n")
	fmt.Printf("  Source: %s\n", sourceURL)
	fmt.Printf("  Destination: %s/%s\n", inventoryDir, filename)

	// Check if file exists
	dstPath := filepath.Join(inventoryDir, filename)
	if _, err := os.Stat(dstPath); err == nil {
		fmt.Printf("  ⚠️  Warning: %s already exists and will be overwritten\n", filename)
	}
	fmt.Println()

	if !ConfirmAction("Proceed with download?") {
		return fmt.Errorf("operation cancelled by user")
	}

	return summonFromHTTP(sourceURL, inventoryDir)
}

// isGitURL checks if the URL points to a git repository.
// User is smart: if they want a repo, they provide a repo URL.
// If they want a file, they provide a file URL.
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

// checkGitAvailable verifies that git is installed and accessible
func checkGitAvailable() error {
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git is not installed or not in PATH. Please install git to summon from repositories.\n\nInstall git:\n  - Linux: sudo apt install git (Debian/Ubuntu) or sudo dnf install git (RHEL/Fedora)\n  - macOS: xcode-select --install\n  - Windows: https://git-scm.com/download/win")
	}
	return nil
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

// summonFromGit clones a profile repository
func summonFromGit(repoURL, inventoryDir string) error {
	// Extract filename from URL or use default
	filename := extractFilenameFromURL(repoURL)
	if !strings.HasSuffix(filename, ".profile.toml") {
		filename = "personal.profile.toml"
	}

	// Create a temporary directory for cloning
	tempDir := filepath.Join(inventoryDir, ".summon-temp")
	defer os.RemoveAll(tempDir)

	// Clone the repository
	fmt.Printf("Cloning repository...\n")
	// Use -- to prevent argument injection if repoURL starts with a hyphen
	cmd := exec.Command("git", "clone", "--", repoURL, tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Find .profile.toml files in the repo
	var profileFiles []string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".profile.toml") {
			profileFiles = append(profileFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to search for profile files: %w", err)
	}

	if len(profileFiles) == 0 {
		return fmt.Errorf("no .profile.toml files found in repository")
	}

	// Show what was found
	fmt.Printf("\nFound %d profile file(s) in repository:\n", len(profileFiles))
	for _, srcPath := range profileFiles {
		fmt.Printf("  - %s\n", filepath.Base(srcPath))
	}
	fmt.Println()

	// Copy and validate all profile files to config directory
	summoned := 0
	skipped := 0
	cancelled := 0

	for _, srcPath := range profileFiles {
		dstName := filepath.Base(srcPath)
		dstPath := filepath.Join(inventoryDir, dstName)

		// Validate filename
		if err := validateFilename(dstName); err != nil {
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
			// However, copyAssetsList signature is fixed for now.
			// Wait, I can just change copyAssetsList signature since it is internal.
			aCopied, aSkipped, aMissing, aBytes := copyAssetsList(tempDir, filepath.Dir(srcPath), assets, profileName)
			fmt.Printf("  Assets: copied=%d, skipped=%d, missing=%d, total=%.1f MB\n",
				aCopied, aSkipped, aMissing, float64(aBytes)/(1024*1024))
			log.Printf("Assets for %s: copied=%d, skipped=%d, missing=%d, bytes=%d", dstName, aCopied, aSkipped, aMissing, aBytes)
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
			return fmt.Errorf("no profiles summoned (all cancelled by user)")
		}
		return fmt.Errorf("no valid profile files found in repository (%d skipped)", skipped)
	}

	log.Printf("Successfully summoned %d profile(s) from repository: %s (skipped: %d, cancelled: %d)", summoned, repoURL, skipped, cancelled)
	return nil
}

// readAssetsFromProfile parses a profile file and returns declared assets (relative paths)
func readAssetsFromProfile(profilePath string) ([]string, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, err
	}
	var overlay config.ProfileOverlay
	if _, err := toml.Decode(string(data), &overlay); err != nil {
		return nil, err
	}
	if overlay.Assets == nil {
		return nil, nil
	}
	var out []string
	for _, raw := range *overlay.Assets {
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

// summonFromHTTP downloads a profile file from an HTTP/HTTPS URL
func summonFromHTTP(sourceURL, inventoryDir string) error {
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

	fmt.Printf("Downloading from %s...\n", sourceURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent header
	req.Header.Set("User-Agent", "drako-summon/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Check Content-Length if provided
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
		os.Remove(tempPath) // Clean up temp file if it still exists
	}()

	// Copy response body to temp file with size limit
	written, err := io.CopyN(tempFile, resp.Body, profileMaxSize+1)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if written > profileMaxSize {
		return fmt.Errorf("file too large (>%d bytes). This is not a valid profile", profileMaxSize)
	}

	tempFile.Close()

	// Validate the downloaded file
	fmt.Printf("Validating profile...\n")
	if err := validateProfileFile(tempPath); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Move temp file to final destination
	if err := os.Rename(tempPath, dstPath); err != nil {
		return fmt.Errorf("failed to finalize file: %w", err)
	}

	fmt.Printf("✓ Summoned: %s\n", filename)
	log.Printf("Successfully summoned profile: %s from %s", filename, sourceURL)
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

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
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

	// Try to parse as profileOverlay (what drako expects)
	var overlay config.ProfileOverlay
	if _, err := toml.Decode(string(data), &overlay); err != nil {
		return fmt.Errorf("invalid TOML format: %w", err)
	}

	// Check if it has at least one profile-related field
	if config.OverlayIsEmpty(overlay) {
		return fmt.Errorf("file contains no profile settings (missing x, y, commands, theme, etc.)")
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

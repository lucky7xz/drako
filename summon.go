package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// confirmAction prompts the user to confirm an action
func confirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// summonProfile downloads a profile from a URL and saves it to the inventory directory.
// Supports:
//   - HTTP/HTTPS URLs (raw file downloads) - user provides file URL
//   - Git repository URLs (clones whole repo) - user provides repo URL
func summonProfile(sourceURL, configDir string) error {
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
		
		if !confirmAction("Proceed with cloning?") {
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
	
	if !confirmAction("Proceed with download?") {
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
	cmd := exec.Command("git", "clone", repoURL, tempDir)
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

		// Get file info for size display
		info, _ := os.Stat(srcPath)
		size := info.Size()
		
		// Check if destination exists
		overwriting := false
		if _, err := os.Stat(dstPath); err == nil {
			overwriting = true
		}

		// Ask for confirmation
		fmt.Printf("  File: %s\n", dstName)
		fmt.Printf("  Size: %d bytes (%.1f KB)\n", size, float64(size)/1024)
		fmt.Printf("  Destination: %s\n", inventoryDir)
		if overwriting {
			fmt.Printf("  ⚠️  Warning: Will overwrite existing file\n")
		}
		
		if !confirmAction(fmt.Sprintf("Summon %s?", dstName)) {
			fmt.Printf("⊘ Cancelled: %s\n", dstName)
			cancelled++
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", dstName, err)
		}
		fmt.Printf("✓ Summoned: %s\n", dstName)
		summoned++
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
	profileWarnSize = 500 * 1024  // 500KB - warn if larger
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
	var overlay profileOverlay
	if _, err := toml.Decode(string(data), &overlay); err != nil {
		return fmt.Errorf("invalid TOML format: %w", err)
	}

	// Check if it has at least one profile-related field
	if overlayIsEmpty(overlay) {
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


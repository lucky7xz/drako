package cli

// Transports for summon: how bytes get from a git repo or an HTTP URL onto
// this machine. No validation or policy lives here — see summon_validate.go
// and summon.go for those.

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

// gitHeadFn resolves a checkout's HEAD commit. A package var so tests can
// stub it; the real implementation shells out to git.
var gitHeadFn = gitHead

func gitHead(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("could not resolve the cloned HEAD commit: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
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

// HTTPDownloader implements FileDownloader using http package
type HTTPDownloader struct{}

// DownloadFile downloads a file from URL to dstPath. On failure dstPath is
// removed, so a partial download never survives. The caller stages dstPath
// somewhere temporary and validates before anything reaches the inventory.
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

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Copy limit+1 so an oversized body is detected rather than truncated.
	written, err := io.CopyN(out, resp.Body, profileMaxSize+1)
	out.Close()
	if err != nil && err != io.EOF {
		os.Remove(dstPath)
		return fmt.Errorf("failed to write file: %w", err)
	}
	if written > profileMaxSize {
		os.Remove(dstPath)
		return fmt.Errorf("file too large (>%d bytes). This is not a valid profile", profileMaxSize)
	}
	return nil
}

// copyFile copies src to dst (os.Rename might fail across partitions)
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

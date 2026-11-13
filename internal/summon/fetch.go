package summon

// Transports for summon: how bytes get from a git repo or an HTTP URL onto
// this machine. No validation or policy lives here — see summon_validate.go
// and summon.go for those.

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucky7xz/drako/internal/proc"
)

// IsGitURL checks if the URL points to a git repository.
func IsGitURL(urlStr string) bool {
	// SSH format: git@github.com:user/repo.git
	if strings.HasPrefix(urlStr, "git@") {
		return true
	}
	// Git and SSH protocols: git://host/user/repo, ssh://git@host/user/repo
	if strings.HasPrefix(urlStr, "git://") || strings.HasPrefix(urlStr, "ssh://") {
		return true
	}
	// URLs ending with .git are repositories
	if strings.HasSuffix(urlStr, ".git") {
		return true
	}
	return false
}

// Unencrypted reports whether src travels in the clear: an http:// or git://
// address. Anyone on the network path can change what arrives, so such a
// summon must be pinned.
func Unencrypted(src string) bool {
	u, err := url.Parse(src)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return scheme == "http" || scheme == "git"
}

// CheckGitAddress refuses a git address drako does not clone. Accepted:
// https://, ssh://, user@host:path, a local path or file://, and http:// or
// git:// (pinned, see Unencrypted). "x::" and other schemes would make git
// run a helper program (ext:: runs any command).
func CheckGitAddress(addr string) error {
	refused := fmt.Errorf("%q: drako clones only https://, ssh://, user@host:path, local paths, and pinned http:// or git:// addresses", addr)
	if strings.Contains(addr, "::") {
		return refused
	}
	if scheme, _, ok := strings.Cut(addr, "://"); ok {
		switch strings.ToLower(scheme) {
		case "https", "ssh", "file", "http", "git":
			return nil
		}
		return refused
	}
	return nil
}

// IsSSHURL checks if the URL uses SSH protocol
func IsSSHURL(urlStr string) bool {
	return strings.HasPrefix(urlStr, "git@") || strings.HasPrefix(urlStr, "ssh://")
}

// MissingSSHKeys reports the user's SSH folder and whether none of the usual
// key files is in it. With no home directory to look in, nothing is missing.
func MissingSSHKeys() (sshDir string, missing bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}

	sshDir = filepath.Join(home, ".ssh")

	// Check for common SSH key files
	for _, keyFile := range []string{"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa"} {
		if _, err := os.Stat(filepath.Join(sshDir, keyFile)); err == nil {
			return sshDir, false
		}
	}
	return sshDir, true
}

// GitHead resolves a checkout's HEAD commit.
func GitHead(dir string) (string, error) {
	out, err := proc.Git.Command("-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("could not resolve the cloned HEAD commit: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GitCheckout checks out a commit in a clone.
func GitCheckout(dir, rev string) error {
	out, err := proc.Git.Command("-C", dir, "-c", "advice.detachedHead=false", "checkout", "-q", "--detach", rev).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pinned commit %s is not in the repository: %w\n%s", rev, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// GitCloner implements RepoCloner by running git
type GitCloner struct{}

func (c *GitCloner) CheckGitAvailable() error {
	cmd := proc.Git.Command("--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	return nil
}

func (c *GitCloner) CloneRepo(url, destDir string) error {
	// Use -- to prevent argument injection if repoURL starts with a hyphen
	cmd := proc.Git.Command("clone", "--", url, destDir)
	cmd.Env = append(os.Environ(), "GIT_ALLOW_PROTOCOL="+gitProtocols(url))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	return nil
}

// gitProtocols is the only protocols git may use, whatever the user's git
// config says: the same rule as CheckGitAddress, enforced by git, so it
// also bounds redirects. Plain http and git only for a pinned address.
func gitProtocols(addr string) string {
	if Unencrypted(addr) {
		return "https:ssh:file:http:git"
	}
	return "https:ssh:file"
}

// HTTPDownloader implements FileDownloader using http package
type HTTPDownloader struct{}

// DownloadFile downloads a file from URL to dstPath. On failure dstPath is
// removed, so a partial download never survives. The caller stages dstPath
// somewhere temporary and validates before anything reaches the inventory.
func (d *HTTPDownloader) DownloadFile(sourceURL, dstPath string) error {
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: keepHTTPS,
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

	if resp.ContentLength > ProfileMaxSize {
		return fmt.Errorf("file too large (%d bytes, max %d bytes)", resp.ContentLength, ProfileMaxSize)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Copy limit+1 so an oversized body is detected rather than truncated.
	written, err := io.CopyN(out, resp.Body, ProfileMaxSize+1)
	out.Close()
	if err != nil && err != io.EOF {
		os.Remove(dstPath)
		return fmt.Errorf("failed to write file: %w", err)
	}
	if written > ProfileMaxSize {
		os.Remove(dstPath)
		return fmt.Errorf("file too large (>%d bytes). This is not a valid profile", ProfileMaxSize)
	}
	return nil
}

// keepHTTPS is the downloader's redirect rule: Go's usual limit of 10, and a
// download that started on HTTPS may not be sent to an unencrypted address.
func keepHTTPS(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if strings.EqualFold(via[0].URL.Scheme, "https") && !strings.EqualFold(req.URL.Scheme, "https") {
		return fmt.Errorf("refused a redirect from https to an unencrypted address (%s)", req.URL.Redacted())
	}
	return nil
}

// ExtractFilenameFromURL extracts a filename from a URL
func ExtractFilenameFromURL(urlStr string) string {
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

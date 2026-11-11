package summon

// Validation of summoned files: everything that arrives via summon is
// untrusted input and must pass these checks before it lands in inventory.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/profile"
)

// IsHexString reports whether s is exactly n hexadecimal characters.
func IsHexString(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// VerifyFileSHA256 hashes path and compares it (case-insensitively) against
// the expected hex digest. A mismatch is an error that names both hashes.
func VerifyFileSHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file for verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("could not hash file: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("sha256 MISMATCH\n  expected %s\n  got      %s\nnothing was written to the inventory", strings.ToLower(expected), got)
	}
	return nil
}

// Profile size limits
const (
	ProfileWarnSize = 500 * 1024      // 500KB - warn if larger
	ProfileMaxSize  = 2 * 1024 * 1024 // 2MB - reject if larger
)

// ValidateProfileFile checks that a file is a valid drako profile and returns
// it decoded, with the file's size: one over ProfileWarnSize is unusual, and
// the caller may want to say so.
func ValidateProfileFile(path string) (pf profile.File, size int64, err error) {
	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		return pf, 0, fmt.Errorf("failed to stat file: %w", err)
	}

	size = info.Size()
	if size > ProfileMaxSize {
		return pf, size, fmt.Errorf("file too large (%d bytes, max %d bytes). This is not a valid profile", size, ProfileMaxSize)
	}

	// Read and parse as TOML
	data, err := os.ReadFile(path)
	if err != nil {
		return pf, size, fmt.Errorf("failed to read file: %w", err)
	}

	pf, problems, err := profile.Decode(data, nil) // validating only: no platform needed
	if err != nil {
		return pf, size, fmt.Errorf("invalid TOML format: %w", err)
	}
	if len(problems) > 0 {
		return pf, size, fmt.Errorf("invalid profile: %s", strings.Join(problems, "; "))
	}
	return pf, size, nil
}

// ValidateFilename checks if the filename is a valid profile name
func ValidateFilename(filename string) error {
	// A plain name: separators of any OS would let it land outside the inventory.
	if strings.ContainsAny(filename, `/\`) || !filepath.IsLocal(filename) {
		return fmt.Errorf("filename must be a plain file name (got: %s)", filename)
	}
	if !strings.HasSuffix(filename, ".profile.toml") {
		return fmt.Errorf("filename must end with .profile.toml (got: %s)", filename)
	}
	// The stem names the deck's assets dir: "..." would put them in the config root.
	if stem := strings.TrimSuffix(filename, ".profile.toml"); stem == "" || stem == "." || stem == ".." {
		return fmt.Errorf("filename needs a deck name before .profile.toml (got: %s)", filename)
	}
	return nil
}

// ValidateSpecFile checks if a file is a valid drako spec
func ValidateSpecFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > ProfileMaxSize {
		return fmt.Errorf("file too large")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Basic TOML validation
	var temp map[string]any
	if _, err := toml.Decode(string(data), &temp); err != nil {
		return fmt.Errorf("invalid TOML: %w", err)
	}

	// Spec files should expect a 'profiles' key (list of strings)
	if profiles, ok := temp["profiles"]; ok {
		if _, isList := profiles.([]any); !isList {
			return fmt.Errorf("missing or invalid 'profiles' list")
		}
	} else {
		return fmt.Errorf("missing 'profiles' key")
	}

	return nil
}

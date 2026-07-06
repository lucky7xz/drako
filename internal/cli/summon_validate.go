package cli

// Validation of summoned files: everything that arrives via summon is
// untrusted input and must pass these checks before it lands in inventory.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/config"
)

// isHexString reports whether s is exactly n hexadecimal characters.
func isHexString(s string, n int) bool {
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

// verifyFileSHA256 hashes path and compares it (case-insensitively) against
// the expected hex digest. A mismatch is an error that names both hashes.
func verifyFileSHA256(path, expected string) error {
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
	if ok, problems := config.ValidateProfileFile(profile, data); !ok {
		return fmt.Errorf("invalid profile: %s", strings.Join(problems, "; "))
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

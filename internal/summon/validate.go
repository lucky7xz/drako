package summon

// Validation of summoned files: everything that arrives via summon is
// untrusted input and must pass these checks before it lands in inventory.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/glyph"
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

// VerifySHA256 compares data's hash, case-insensitively, with the expected
// hex digest.
func VerifySHA256(data []byte, expected string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
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

// ValidateProfile checks that f is a valid drako deck and returns it decoded.
func ValidateProfile(f File) (File, error) {
	pf, problems, err := profile.Decode(f.Data, nil) // validating only: no platform needed
	if err != nil {
		return f, fmt.Errorf("invalid TOML format: %w", err)
	}
	if len(problems) > 0 {
		return f, fmt.Errorf("invalid profile: %s", strings.Join(problems, "; "))
	}
	f.Profile = pf
	return f, nil
}

// ValidateFilename checks if the filename is a valid profile name
func ValidateFilename(filename string) error {
	// Drawn on screen and typed by the user, so exactly what drako draws.
	if err := glyph.CheckName(filename); err != nil {
		return fmt.Errorf("file name %v", err)
	}
	// A plain name: separators of any OS would let it land outside the inventory.
	if strings.ContainsAny(filename, `/\`) || !filepath.IsLocal(filename) {
		return fmt.Errorf("filename must be a plain file name (got: %s)", filename)
	}
	if !strings.HasSuffix(filename, ".profile.toml") {
		return fmt.Errorf("filename must end with .profile.toml (got: %s)", filename)
	}
	// The stem names the deck's assets dir: "..." would put them in the config root.
	if stem := strings.TrimSuffix(filename, ".profile.toml"); stem == "" || stem == "." || stem == ".." {
		return fmt.Errorf("filename needs a name before .profile.toml (got: %s)", filename)
	}
	return nil
}

// ValidateSpec checks that f is a valid drako spec.
func ValidateSpec(f File) error {
	var temp map[string]any
	if _, err := toml.Decode(string(f.Data), &temp); err != nil {
		return fmt.Errorf("invalid TOML: %w", err)
	}
	// Spec files should expect a 'profiles' key (list of strings)
	profiles, ok := temp["profiles"]
	if !ok {
		return fmt.Errorf("missing 'profiles' key")
	}
	if _, isList := profiles.([]any); !isList {
		return fmt.Errorf("missing or invalid 'profiles' list")
	}
	return nil
}

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateSpecFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string // empty means valid
	}{
		{"valid spec", "profiles = [\"work\", \"alpha\"]\n", ""},
		{"empty profiles list is valid", "profiles = []\n", ""},
		{"missing profiles key", "x = 1\n", "missing 'profiles' key"},
		{"profiles not a list", "profiles = \"work\"\n", "missing or invalid 'profiles' list"},
		{"garbage toml", "not { toml", "invalid TOML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, "x.spec.toml", tt.content)
			err := validateSpecFile(path)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateSpecFile_MissingFile(t *testing.T) {
	if err := validateSpecFile(filepath.Join(t.TempDir(), "ghost.spec.toml")); err == nil {
		t.Fatal("missing file must be an error")
	}
}

func TestValidateSpecFile_TooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.spec.toml")
	if err := os.WriteFile(path, bytes.Repeat([]byte("# pad\n"), (profileMaxSize/6)+2), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateSpecFile(path)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("oversized spec must be rejected, got %v", err)
	}
}

func TestValidateFilename(t *testing.T) {
	if err := validateFilename("networking.profile.toml"); err != nil {
		t.Errorf("valid name rejected: %v", err)
	}
	for _, bad := range []string{"networking.toml", "profile.toml.bak", "networking"} {
		if err := validateFilename(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestValidateProfileFile(t *testing.T) {
	valid := "x = 1\ny = 1\n[[commands]]\nname = \"noop\"\ncommand = \"true\"\nrow = 0\ncol = \"a\"\n"
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"valid profile", valid, ""},
		{"garbage toml", "not { toml", "invalid TOML format"},
		{"no commands", "x = 1\ny = 1\n", "invalid profile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, "x.profile.toml", tt.content)
			err := validateProfileFile(path)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateProfileFile_TooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.profile.toml")
	if err := os.WriteFile(path, bytes.Repeat([]byte("# pad\n"), (profileMaxSize/6)+2), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateProfileFile(path)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("oversized profile must be rejected, got %v", err)
	}
}

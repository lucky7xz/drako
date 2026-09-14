package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// decodeForTest gives tests a real toml.MetaData instead of a zero-value one.
func decodeForTest(t *testing.T, raw string) (ProfileFile, toml.MetaData) {
	t.Helper()
	var pf ProfileFile
	meta, err := toml.Decode(raw, &pf)
	if err != nil {
		t.Fatalf("test fixture failed to decode: %v", err)
	}
	return pf, meta
}

func TestValidateProfileFile_Valid(t *testing.T) {
	raw := "x = 3\ny = 3\n\n[[commands]]\nname = \"a\"\n"
	pf, meta := decodeForTest(t, raw)
	if ok, problems := ValidateProfileFile(pf, []byte(raw), meta); !ok {
		t.Fatalf("valid profile rejected: %v", problems)
	}
}

func TestValidateProfileFile_OutOfRangeReportsLine(t *testing.T) {
	raw := "x = 12\ny = 3\n\n[[commands]]\nname = \"a\"\n"
	pf, meta := decodeForTest(t, raw)

	ok, problems := ValidateProfileFile(pf, []byte(raw), meta)
	if ok {
		t.Fatal("x = 12 should be rejected")
	}
	msg := strings.Join(problems, "; ")
	if !strings.Contains(msg, "x = 12") || !strings.Contains(msg, "must be 1-9") {
		t.Errorf("message should explain the x range, got %q", msg)
	}
	if !strings.Contains(msg, "line 1") {
		t.Errorf("message should point to the line, got %q", msg)
	}
}

func TestValidateProfileFile_NoCommands(t *testing.T) {
	raw := "x = 3\ny = 3\n"
	pf, meta := decodeForTest(t, raw)
	ok, problems := ValidateProfileFile(pf, []byte(raw), meta)
	if ok {
		t.Fatal("a profile with no commands should be rejected")
	}
	if !strings.Contains(strings.Join(problems, "; "), "command") {
		t.Errorf("expected a command problem, got %v", problems)
	}
}

func TestValidateProfileFile_MissingKeyReportsMissing(t *testing.T) {
	// y is absent entirely, not just out of range — the message should say
	// so instead of reporting the Go zero-value as if it were written.
	raw := "x = 3\n\n[[commands]]\nname = \"a\"\n"
	pf, meta := decodeForTest(t, raw)

	ok, problems := ValidateProfileFile(pf, []byte(raw), meta)
	if ok {
		t.Fatal("a profile missing y should be rejected")
	}
	msg := strings.Join(problems, "; ")
	if !strings.Contains(msg, "y is missing") {
		t.Errorf("expected a 'y is missing' problem, got %q", msg)
	}
	if strings.Contains(msg, "y = 0") {
		t.Errorf("should not fabricate 'y = 0' for an absent key, got %q", msg)
	}
}

func TestValidateProfileFile_UnrecognizedKeyReported(t *testing.T) {
	// A typo'd key (y -> ssy) leaves y looking "missing" AND leaves ssy
	// sitting in the file unused — both should be named.
	raw := "x = 3\nssy = 3\n\n[[commands]]\nname = \"a\"\n"
	pf, meta := decodeForTest(t, raw)

	ok, problems := ValidateProfileFile(pf, []byte(raw), meta)
	if ok {
		t.Fatal("a profile with an unrecognized key should be rejected")
	}
	msg := strings.Join(problems, "; ")
	if !strings.Contains(msg, "unrecognized key(s): ssy") {
		t.Errorf("expected ssy to be named as unrecognized, got %q", msg)
	}
	if !strings.Contains(msg, "y is missing") {
		t.Errorf("expected y to still be reported missing, got %q", msg)
	}
}

func TestFindKeyLine(t *testing.T) {
	raw := []byte("# comment\nx = 3\ny=4\n[[commands]]\n")
	cases := map[string]int{"x": 2, "y": 3, "z": 0}
	for key, want := range cases {
		if got := findKeyLine(raw, key); got != want {
			t.Errorf("findKeyLine(%q) = %d, want %d", key, got, want)
		}
	}
}

func TestCheckProfileFile(t *testing.T) {
	dir := t.TempDir()

	good := filepath.Join(dir, "g.profile.toml")
	os.WriteFile(good, []byte("x=1\ny=1\n[[commands]]\nname=\"a\"\ncol=\"A\"\nrow=0\n"), 0o644)
	if err := CheckProfileFile(good); err != nil {
		t.Errorf("valid file rejected: %v", err)
	}

	bad := filepath.Join(dir, "b.profile.toml")
	os.WriteFile(bad, []byte("not [ toml"), 0o644)
	if err := CheckProfileFile(bad); err == nil {
		t.Error("invalid TOML accepted")
	}

	empty := filepath.Join(dir, "e.profile.toml")
	os.WriteFile(empty, []byte(""), 0o644)
	if err := CheckProfileFile(empty); err == nil {
		t.Error("empty profile accepted")
	}

	if err := CheckProfileFile(filepath.Join(dir, "missing.profile.toml")); err == nil {
		t.Error("missing file accepted")
	}
}

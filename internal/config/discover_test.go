package config

import (
	"strings"
	"testing"
)

func TestValidateProfileFile_Valid(t *testing.T) {
	pf := ProfileFile{X: 3, Y: 3, Commands: []Command{{Name: "a"}}}
	if ok, problems := ValidateProfileFile(pf, nil); !ok {
		t.Fatalf("valid profile rejected: %v", problems)
	}
}

func TestValidateProfileFile_OutOfRangeReportsLine(t *testing.T) {
	raw := []byte("x = 12\ny = 3\n\n[[commands]]\nname = \"a\"\n")
	pf := ProfileFile{X: 12, Y: 3, Commands: []Command{{Name: "a"}}}

	ok, problems := ValidateProfileFile(pf, raw)
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
	pf := ProfileFile{X: 3, Y: 3}
	ok, problems := ValidateProfileFile(pf, nil)
	if ok {
		t.Fatal("a profile with no commands should be rejected")
	}
	if !strings.Contains(strings.Join(problems, "; "), "command") {
		t.Errorf("expected a command problem, got %v", problems)
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

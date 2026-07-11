package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func falsePtr() *bool { b := false; return &b }

func explainBundle() config.ConfigBundle {
	return config.ConfigBundle{
		ActiveIndex: 0,
		Profiles: []config.ProfileInfo{
			{Name: "core", Profile: config.ProfileFile{X: 2, Y: 2, Commands: []config.Command{
				{Name: "Update", Command: "sudo apt update", Col: "a", Row: 0, Description: "refresh packages"},
				{Name: "Maint", Col: "a", Row: 1, Description: "housekeeping folder", Items: []config.CommandItem{
					{Name: "Clean", Command: "apt clean", Description: "clear caches"},
					{Name: "Scan", Command: "clamscan -r", AutoCloseExecution: falsePtr()},
				}},
				{Name: "GhostCmd", Col: "b", Row: 0, Description: "no variant here"},
			}}},
			{Name: "work", Profile: config.ProfileFile{X: 1, Y: 1, Commands: []config.Command{
				{Name: "Deploy", Command: "make deploy", Col: "a", Row: 0, AutoCloseExecution: falsePtr()},
			}}},
		},
	}
}

func explainOut(t *testing.T, ref string) (string, error) {
	t.Helper()
	var sb strings.Builder
	err := explainCell(&sb, explainBundle(), ref)
	return sb.String(), err
}

func TestExplainActiveProfileDefault(t *testing.T) {
	out, err := explainOut(t, "A0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"core", "A0", "Update", "sudo apt update", "refresh packages", "true (default)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainQualifiedProfile(t *testing.T) {
	out, err := explainOut(t, "work:a0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"work", "Deploy", "make deploy"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "false") || strings.Contains(out, "(default)") {
		t.Errorf("explicit auto_close=false must print without the default marker:\n%s", out)
	}
}

func TestExplainDropdownItem(t *testing.T) {
	out, err := explainOut(t, "A1.2")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"A1.2", "Scan", "clamscan -r", "false"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainFolderListsItems(t *testing.T) {
	out, err := explainOut(t, "A1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Maint", "housekeeping folder", "items:", "A1.1", "Clean", "apt clean", "A1.2", "Scan"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainFolderItemSummaryIsOneLine(t *testing.T) {
	b := explainBundle()
	b.Profiles[0].Profile.Commands[1].Items[0].Command = "# check first\nif true; then\n  echo hi\nfi"

	var sb strings.Builder
	if err := explainCell(&sb, b, "A1"); err != nil {
		t.Fatal(err)
	}
	out := sb.String()
	if strings.Contains(out, "if true") {
		t.Errorf("items list must summarize multi-line commands to their first line:\n%s", out)
	}
	if !strings.Contains(out, "# check first") {
		t.Errorf("first line of the command should still show:\n%s", out)
	}
}

func TestExplainPlatformEmptyCommand(t *testing.T) {
	out, err := explainOut(t, "B0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "(none for this platform)") {
		t.Errorf("empty command must be explicit:\n%s", out)
	}
}

func TestExplainErrors(t *testing.T) {
	for ref, wantErr := range map[string]string{
		"nosuch:A0": "core",    // unknown profile error lists the equipped names
		"C5":        "no cell", // outside the deck
		"A1.9":      "item",    // folder exists, item index doesn't
		"A0.1":      "item",    // plain command has no items
	} {
		_, err := explainOut(t, ref)
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("explainCell(%q) err = %v, want mention of %q", ref, err, wantErr)
		}
	}
}

func TestHandleExplainCommand_ExitCodes(t *testing.T) {
	if got := HandleExplainCommand([]string{"drako", "explain"}); got != 1 {
		t.Errorf("no refs should exit 1, got %d", got)
	}

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("DRAKO_PROFILE", "")
	dir := filepath.Join(tmp, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("theme = \"default\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := "x = 1\ny = 1\n[[commands]]\nname = \"noop\"\ncommand = \"true\"\nrow = 0\ncol = \"a\"\n"
	if err := os.WriteFile(filepath.Join(dir, "solo.profile.toml"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := HandleExplainCommand([]string{"drako", "explain", "A0"}); got != 0 {
		t.Errorf("valid ref should exit 0, got %d", got)
	}
	if got := HandleExplainCommand([]string{"drako", "explain", "A0", "Z9"}); got != 1 {
		t.Errorf("partial failure should exit 1, got %d", got)
	}
}

func TestParseCellRef(t *testing.T) {
	tests := []struct {
		ref         string
		wantProfile string
		wantAddr    string
		wantItem    int // 0 = no item
		wantErr     bool
	}{
		{"A0", "", "A0", 0, false},
		{"b12", "", "B12", 0, false},
		{"work:A1.2", "work", "A1", 2, false},
		{":A0", "", "A0", 0, false},
		{"Work:B2", "Work", "B2", 0, false},
		{"a1.10", "", "A1", 10, false},
		{"A", "", "", 0, true},
		{"1A", "", "", 0, true},
		{"A1.0", "", "", 0, true},
		{"A1.2.3", "", "", 0, true},
		{"work:", "", "", 0, true},
		{"", "", "", 0, true},
		{"AA1", "", "", 0, true},
		{"A1.x", "", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			profile, addr, item, err := parseCellRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseCellRef(%q) should fail, got (%q,%q,%d)", tt.ref, profile, addr, item)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCellRef(%q) = %v", tt.ref, err)
			}
			if profile != tt.wantProfile || addr != tt.wantAddr || item != tt.wantItem {
				t.Errorf("parseCellRef(%q) = (%q,%q,%d), want (%q,%q,%d)",
					tt.ref, profile, addr, item, tt.wantProfile, tt.wantAddr, tt.wantItem)
			}
		})
	}
}

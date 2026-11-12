package profile

import (
	"strings"
	"testing"
)

// A string that steers the terminal makes the deck invalid, wherever it sits:
// labels, commands of any platform, items, top-level keys, keys drako ignores.
func TestDecodeRefusesSteeringText(t *testing.T) {
	for place, deck := range map[string]string{
		`commands › "\x1b[2JHi" › name`:          "[[commands]]\nname = \"\\u001b[2JHi\"\ncommand = \"echo\"",
		`commands › "A" › description`:           "[[commands]]\nname = \"A\"\ndescription = \"x\\u202ey\"\ncommand = \"echo\"",
		`commands › "A" › command › macos`:       "[[commands]]\nname = \"A\"\ncommand = { linux_arch = \"ok\", macos = \"rm\\u001b[1A\" }",
		`commands › "A" › items › "B" › command`: "[[commands]]\nname = \"A\"\n[[commands.items]]\nname = \"B\"\ncommand = \"echo \\r\"",
		`header_art`: "header_art = \"\\u001b]52;c;eA==\\u0007\"\n[[commands]]\nname = \"A\"\ncommand = \"echo\"",
		`notes`:      "notes = \"\\u001b\"\n[[commands]]\nname = \"A\"\ncommand = \"echo\"",
	} {
		_, problems, err := Decode([]byte("x = 1\ny = 1\n"+deck+"\n"), nil)
		if err != nil {
			t.Fatalf("%s: %v", place, err)
		}
		found := false
		for _, p := range problems {
			if strings.HasPrefix(p, place+" contains U+") {
				found = true
			}
		}
		if !found {
			t.Errorf("want a problem at %s, got %q", place, problems)
		}
	}
}

// Labels are kept in the form drako draws; commands run as written.
func TestDecodeCleansLabelsNotCommands(t *testing.T) {
	deck := "x = 1\ny = 1\nheader_art = \"👩\u200d🏫\"\n[[commands]]\nname = \"👩\u200d🏫 Grown-ups\"\ndescription = \"⚠\ufe0f\"\ncommand = \"echo ⚠\ufe0f\"\n"
	pf, problems, err := Decode([]byte(deck), nil)
	if err != nil || len(problems) > 0 {
		t.Fatalf("err = %v, problems = %q", err, problems)
	}
	c := pf.Commands[0]
	if c.Name != "👩🏫 Grown-ups" || c.Description != "⚠" || *pf.HeaderArt != "👩🏫" {
		t.Errorf("labels not cleaned: name %q, description %q, art %q", c.Name, c.Description, *pf.HeaderArt)
	}
	if c.Command != "echo ⚠\ufe0f" {
		t.Errorf("command changed: %q", c.Command)
	}
}

// check warns about a label drawn without some of its characters, never
// about a command.
func TestLintWarnsAboutJoinedLabels(t *testing.T) {
	deck := "x = 1\ny = 1\n[[commands]]\nname = \"👩\u200d🏫 Grown-ups\"\ncol = \"a\"\nrow = 0\ncommand = \"echo ⚠\ufe0f\"\n"
	var warned []string
	for _, f := range Lint([]byte(deck)) {
		if f.Level == "error" {
			t.Errorf("unexpected error: %s", f.Msg)
		}
		if strings.Contains(f.Msg, "different widths") {
			warned = append(warned, f.Msg)
		}
	}
	if len(warned) != 1 || !strings.Contains(warned[0], `name holds`) || !strings.Contains(warned[0], `"👩🏫 Grown-ups"`) {
		t.Errorf("want one warning about the name, got %q", warned)
	}
}

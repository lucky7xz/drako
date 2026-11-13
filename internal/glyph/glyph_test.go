package glyph

import (
	"strings"
	"testing"
)

// Escapes only: this file must not itself hold the characters it is about.

func TestCheck(t *testing.T) {
	for _, s := range []string{"", "Health Check", "\U0001fa7a Health", "\u5578\u9f99\u5fd7", "\u250f\u2501\u252b \u2718", "two\nlines\tand a tab", "\u26a0\ufe0f \U0001f469\u200d\U0001f3eb"} {
		if err := Check(s); err != nil {
			t.Errorf("Check(%q) = %v, want nil", s, err)
		}
	}
	for s, want := range map[string]string{
		"ls\x1b[2J":       "U+001B",
		"over\rwrite":     "U+000D",
		"del\x7f":         "U+007F",
		"c1\u009b":        "U+009B",
		"rm \u202e-rf":    "U+202E",
		"isolate\u2066x":  "U+2066",
		"left mark\u200e": "U+200E",
	} {
		err := Check(s)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Check(%q) = %v, want an error naming %s", s, err, want)
		}
	}
}

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"Health Check":                         "Health Check",
		"\U0001f469\u200d\U0001f3eb Grown-ups": "\U0001f469\U0001f3eb Grown-ups",
		"\u26a0\ufe0f careful":                 "\u26a0 careful",
		"zero\u200bwidth":                      "zerowidth",
		"soft\u00adhyphen":                     "softhyphen",
		"ls\x1b[2J":                            "ls\ufffd[2J",
		"rm \u202e-rf":                         "rm \ufffd-rf",
		"two\nlines\tkept":                     "two\nlines\tkept",
		"\u5578 \u250f\u2501 \u2718 \U0001f1eb\U0001f1f7 \u22ee": "\u5578 \u250f\u2501 \u2718 \U0001f1eb\U0001f1f7 \u22ee",
	} {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// What Clean gives back, Check accepts: drawing never reintroduces a
// character that steers the terminal.
func TestCleanPassesCheck(t *testing.T) {
	for _, s := range []string{"\x1b]52;c;ZWNobw==\a", "\u202e\u2066\x00\x1b", "a\u200d\ufe0fb"} {
		if err := Check(Clean(s)); err != nil {
			t.Errorf("Check(Clean(%q)) = %v", s, err)
		}
	}
}

func TestCheckName(t *testing.T) {
	for _, ok := range []string{"work.profile.toml", "\U0001f409 dragon.profile.toml", "\u5578\u9f99.profile.toml"} {
		if err := CheckName(ok); err != nil {
			t.Errorf("CheckName(%q) = %v", ok, err)
		}
	}
	for name, want := range map[string]string{
		"ops\u26a0\ufe0f.profile.toml":                "U+FE0F, which drako does not draw as written",
		"team\U0001f469\u200d\U0001f4bb.profile.toml": "U+200D, which drako does not draw as written",
		"x\x1b]52;c;eA==.profile.toml":                "U+001B, a character that steers the terminal",
	} {
		if err := CheckName(name); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("CheckName(%q) = %v, want %q", name, err, want)
		}
	}
}

// A label is one line: line breaks and tabs are drawn as spaces (U-12, P-01).
func TestLabel(t *testing.T) {
	for in, want := range map[string]string{
		"plain":      "plain",
		"two\nlines": "two lines",
		"tab\there":  "tab here",
		"a‍b\nc":     "ab c",
	} {
		if got := Label(in); got != want {
			t.Errorf("Label(%q) = %q, want %q", in, got, want)
		}
		if strings.ContainsAny(Label(in), "\n\t") {
			t.Errorf("Label(%q) still holds a line break or tab", in)
		}
	}
}

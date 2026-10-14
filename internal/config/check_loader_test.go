package config

import (
	"os"
	"path/filepath"
	"testing"
)

// drako check is the CI gate: any deck the loader would show as broken must
// fail it, never pass as "ok".
func TestCheckRejectsWhatTheLoaderRejects(t *testing.T) {
	cell := "[[commands]]\nname = \"a\"\ncommand = \"true\"\nrow = 0\ncol = \"a\"\n"
	decks := map[string]string{
		"missing y":      "x = 1\n" + cell,
		"x out of range": "x = 12\ny = 1\n" + cell,
		"typo'd key":     "x = 1\ny = 1\nssy = 3\n" + cell,
		"x as a string":  "x = \"3\"\ny = 1\n" + cell,
	}
	for name, deck := range decks {
		path := filepath.Join(t.TempDir(), "d.profile.toml")
		if err := os.WriteFile(path, []byte(deck), 0o644); err != nil {
			t.Fatal(err)
		}
		if CheckProfileFile(path) == nil {
			t.Fatalf("%s: setup expects the loader to reject this deck", name)
		}
		hasError := false
		for _, f := range CheckProfile([]byte(deck)) {
			hasError = hasError || f.Level == "error"
		}
		if !hasError {
			t.Errorf("%s: drako check passes a deck the loader rejects", name)
		}
	}
}

package proc

import (
	"slices"
	"testing"
)

func TestResolveEditor(t *testing.T) {
	t.Run("VISUAL wins over EDITOR", func(t *testing.T) {
		t.Setenv("VISUAL", "visual-ed")
		t.Setenv("EDITOR", "other-ed")
		ed, err := resolveEditor()
		if err != nil || ed[0] != "visual-ed" {
			t.Errorf("got %v (%v), want visual-ed", ed, err)
		}
	})
	t.Run("EDITOR with arguments splits", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "code -w")
		ed, err := resolveEditor()
		if err != nil || len(ed) != 2 || ed[0] != "code" || ed[1] != "-w" {
			t.Errorf("got %v (%v), want [code -w]", ed, err)
		}
	})
	t.Run("falls back to a common editor", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "")
		ed, err := resolveEditor()
		// The machine may or may not have nano/vim; either a fallback or
		// the set-$EDITOR error is acceptable — never an empty success.
		if err == nil && len(ed) == 0 {
			t.Error("empty editor with nil error")
		}
	})
}

func TestEditorOpensThePath(t *testing.T) {
	t.Setenv("VISUAL", "code -w")
	cmd, err := Editor("/tmp/x.profile.toml")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"code", "-w", "/tmp/x.profile.toml"}; !slices.Equal(cmd.Args, want) {
		t.Errorf("args=%v, want %v", cmd.Args, want)
	}
}

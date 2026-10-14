package config

import (
	"io/fs"
	"strings"
	"testing"
)

func TestDangerDecodes(t *testing.T) {
	p := decodeProfile(t, `
x = 2
y = 2
[[commands]]
name = "safe"
command = "true"
col = "a"
row = 0
[[commands]]
name = "nuke"
command = "true"
col = "a"
row = 1
danger = true
[[commands]]
name = "folder"
col = "b"
row = 0
items = [
    { name = "calm", command = "true" },
    { name = "wipe", command = "true", danger = true },
]
`)
	if p.Commands[0].Danger {
		t.Error("absent danger must decode as false")
	}
	if !p.Commands[1].Danger {
		t.Error("cell danger = true not decoded")
	}
	if p.Commands[2].Danger {
		t.Error("folder without danger decoded as dangerous")
	}
	if p.Commands[2].Items[0].Danger || !p.Commands[2].Items[1].Danger {
		t.Errorf("item danger = %v, %v; want false, true",
			p.Commands[2].Items[0].Danger, p.Commands[2].Items[1].Danger)
	}
}

func TestDangerSurvivesOverlay(t *testing.T) {
	p := ProfileFile{X: 1, Y: 1, Commands: []Command{
		{Name: "f", Col: "a", Danger: true, Items: []CommandItem{{Name: "i", Danger: true}}},
	}}
	cfg := ApplyProfileOverlay(Config{}, p)
	if !cfg.Commands[0].Danger || !cfg.Commands[0].Items[0].Danger {
		t.Error("overlay dropped the danger flag")
	}
}

func TestCheckDangerMustBeBool(t *testing.T) {
	src := `
x = 1
y = 2
[[commands]]
name = "typo"
command = "true"
col = "a"
row = 0
danger = "yes"
[[commands]]
name = "folder"
col = "a"
row = 1
items = [ { name = "item", command = "true", danger = 1 } ]
`
	var errs []string
	for _, f := range CheckProfile([]byte(src)) {
		if f.Level == "error" && strings.Contains(f.Msg, "danger must be true or false") {
			errs = append(errs, f.Msg)
		}
	}
	if len(errs) != 2 {
		t.Fatalf("want 2 danger findings (cell, item), got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], `"typo"`) || !strings.Contains(errs[1], `"item"`) {
		t.Errorf("findings don't name their cells: %v", errs)
	}
}

func TestCheckDangerBoolIsClean(t *testing.T) {
	src := `
x = 1
y = 1
[[commands]]
name = "ok"
command = "true"
col = "a"
row = 0
danger = false
`
	for _, f := range CheckProfile([]byte(src)) {
		if strings.Contains(f.Msg, "danger") {
			t.Errorf("boolean danger flagged: %s", f.Msg)
		}
	}
}

// 🚨 is the visual marker and danger is the rule; in the shipped decks the
// two must agree, so the emoji never promises a guard that isn't there.
func TestBootstrapDangerMatchesMarker(t *testing.T) {
	pinPlatform(t, "linux_generic")
	var decks []string
	fs.WalkDir(bootstrapFS, "bootstrap", func(path string, d fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".profile.toml") {
			decks = append(decks, path)
		}
		return nil
	})
	if len(decks) < 2 {
		t.Fatalf("found only %d bootstrap decks: %v", len(decks), decks)
	}

	marked := 0
	for _, path := range decks {
		src, err := bootstrapFS.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p := decodeProfile(t, string(src))
		check := func(name string, danger bool) {
			if strings.Contains(name, "🚨") != danger {
				t.Errorf("%s: %q has 🚨 = %v but danger = %v", path, name, !danger, danger)
			}
			if danger {
				marked++
			}
		}
		for _, c := range p.Commands {
			check(c.Name, c.Danger)
			for _, it := range c.Items {
				check(it.Name, it.Danger)
			}
		}
	}
	if marked == 0 {
		t.Error("no danger cells found; the walk is probably broken")
	}
}

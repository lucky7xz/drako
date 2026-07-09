package cli

import (
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func lsBundle() config.ConfigBundle {
	return config.ConfigBundle{
		ActiveIndex: 1,
		Profiles: []config.ProfileInfo{
			{
				Name: "core",
				Profile: config.ProfileFile{X: 2, Y: 2, Commands: []config.Command{
					{Name: "🧹 Maintenance", Col: "B", Row: 1, Description: "line one\nline two", Items: []config.CommandItem{
						{Name: "Clean", Description: "clears caches", Command: "apt clean"},
					}},
					{Name: "⬆️ Update", Col: "A", Row: 0, Description: strings.Repeat("x", 100)},
				}},
			},
			{
				Name:    "t103",
				Profile: config.ProfileFile{X: 1, Y: 1, Commands: []config.Command{{Name: "Solo", Col: "A", Row: 0, Description: "d"}}},
			},
		},
		Broken: []config.ProfileParseError{{Name: "busted", Err: "invalid TOML: something"}},
	}
}

func TestRenderLs(t *testing.T) {
	var sb strings.Builder
	renderLs(&sb, lsBundle())
	out := sb.String()

	for _, want := range []string{
		"core — 2x2",           // first profile, not active
		"t103 (active) — 1x1",  // active marker on the right profile
		"A0",                   // address of the sorted-first command
		"B1.1",                 // dropdown item sub-address
		"Clean",                // item name present
		"line one",             // first description line kept
		"⚠️  busted is broken", // broken footer
		"invalid TOML",         // with its error
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}

	if strings.Contains(out, "line two") {
		t.Error("descriptions must be truncated to their first line")
	}
	if strings.Contains(out, strings.Repeat("x", 100)) {
		t.Error("long descriptions must be width-truncated")
	}
	// Deck order: A0 (Update) must precede B1 (Maintenance) despite input order.
	if strings.Index(out, "⬆️ Update") > strings.Index(out, "🧹 Maintenance") {
		t.Error("commands not in deck (column,row) order")
	}
}

func TestRenderLsEmpty(t *testing.T) {
	var sb strings.Builder
	renderLs(&sb, config.ConfigBundle{})
	if !strings.Contains(sb.String(), "No equipped profiles") {
		t.Errorf("empty bundle output = %q", sb.String())
	}
}

func TestHandleLsCommand_UsageError(t *testing.T) {
	if got := HandleLsCommand([]string{"drako", "ls", "extra"}); got != 1 {
		t.Errorf("HandleLsCommand with extra arg = %d, want 1", got)
	}
}

package cli

import (
	"github.com/charmbracelet/x/ansi"
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
	renderLs(&sb, lsBundle(), 200)
	out := sb.String()

	for _, want := range []string{
		"core — 2x2",           // first profile, not active
		"t103 (active) — 1x1",  // active marker on the right profile
		"A0",                   // address of the sorted-first command
		"B1.1",                 // dropdown item sub-address
		"│   Clean",            // item name present, indented under its parent
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
	renderLs(&sb, config.ConfigBundle{}, 200)
	if !strings.Contains(sb.String(), "No equipped profiles") {
		t.Errorf("empty bundle output = %q", sb.String())
	}
}

// wideBundle exercises the width fitting: long emoji names, a 100-cell
// description, and a broken-profile footer with a long error.
func wideBundle() config.ConfigBundle {
	return config.ConfigBundle{
		Profiles: []config.ProfileInfo{{
			Name: "core",
			Profile: config.ProfileFile{X: 2, Y: 2, Commands: []config.Command{
				{Name: "⬆️ System Update With A Very Long Cell Name Indeed", Col: "A", Row: 0,
					Description: strings.Repeat("very long description ", 10)},
				{Name: "🧹 Maintenance", Col: "A", Row: 1,
					Description: strings.Repeat("more words here ", 10)},
			}},
		}},
		Broken: []config.ProfileParseError{{Name: "busted", Err: strings.Repeat("broken toml ", 20)}},
	}
}

func maxLineWidth(out string) int {
	widest := 0
	for _, line := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(line); w > widest {
			widest = w
		}
	}
	return widest
}

func TestRenderLsFitsTerminalWidth(t *testing.T) {
	var sb strings.Builder
	renderLs(&sb, wideBundle(), 80)
	out := sb.String()

	if w := maxLineWidth(out); w > 80 {
		t.Errorf("line width %d exceeds 80-col terminal\n---\n%s", w, out)
	}
	if !strings.Contains(out, "DESCRIPTION") {
		t.Error("80 cols has room for a (shrunk) description column")
	}
}

func TestRenderLsNarrowDropsDescriptions(t *testing.T) {
	var sb strings.Builder
	renderLs(&sb, wideBundle(), 45)
	out := sb.String()

	if w := maxLineWidth(out); w > 45 {
		t.Errorf("line width %d exceeds 45-col terminal\n---\n%s", w, out)
	}
	if strings.Contains(out, "DESCRIPTION") {
		t.Error("too narrow for descriptions: the column should be dropped")
	}
	if !strings.Contains(out, "ADDR") {
		t.Error("address/name table should survive in narrow terminals")
	}
}

func TestRenderLsWideKeepsDescriptionCap(t *testing.T) {
	var sb strings.Builder
	renderLs(&sb, wideBundle(), 500)
	out := sb.String()

	// Even with a huge terminal the description column stays readable
	// (capped at lsDescriptionWidth), so lines stay well under the width.
	if w := maxLineWidth(out); w > 120 {
		t.Errorf("wide terminal should still cap the table at readable width, got %d", w)
	}
}

func TestHandleLsCommand_UsageError(t *testing.T) {
	if got := HandleLsCommand([]string{"drako", "ls", "extra"}); got != 1 {
		t.Errorf("HandleLsCommand with extra arg = %d, want 1", got)
	}
}

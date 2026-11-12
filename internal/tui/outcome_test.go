package tui

import (
	"slices"
	"testing"

	"github.com/lucky7xz/drako/internal/launch"
)

// The host acts on Outcome alone, so it must say one thing: quitting outranks
// a batch, a batch outranks a single run, and nothing asked means resume.
func TestOutcome(t *testing.T) {
	m := batchTestModel() // "one" and "two" run echo; "folder" has no command of its own
	if got := m.Outcome(); got.Action != Resume {
		t.Errorf("nothing asked: action %v, want Resume", got.Action)
	}

	m.selected = launch.Selection{Cell: "one"}
	got := m.Outcome()
	if got.Action != Run || len(got.Plans) != 1 || got.Plans[0].Name != "one" || !got.Plans[0].Runnable() {
		t.Errorf("single run: %+v, want Run with the plan for one", got)
	}

	m.selectedBatch = []launch.Selection{{Cell: "two"}, {Cell: "folder"}}
	m.selectedTabs = []int{2}
	m.Config.BatchForceTmux = true
	got = m.Outcome()
	if got.Action != Batch || len(got.Plans) != 2 || !slices.Equal(got.Tabs, []int{2}) || !got.ForceTmux {
		t.Fatalf("batch: %+v, want Batch with a plan per mark, the tabs and the tmux setting", got)
	}
	if !got.Plans[0].Runnable() || got.Plans[1].Runnable() {
		t.Errorf("batch plans: %+v, want two runnable and folder not", got.Plans)
	}

	m.quitting, m.exitCode = true, 3
	if got := m.Outcome(); got.Action != Quit || got.ExitCode != 3 || got.Plans != nil {
		t.Errorf("quitting: %+v, want Quit with exit code 3 and nothing else", got)
	}
}

package settings

import "testing"

func TestMix(t *testing.T) {
	if got, want := mix("#ff2e63", "#0d0221", lockedAccentWeight), "#79153e"; got != want {
		t.Errorf("mix(#ff2e63, #0d0221) = %s, want %s", got, want)
	}
	if mix("not-a-color", "#000000", 0.5) != "not-a-color" {
		t.Error("a malformed first hex should be returned unchanged")
	}
	if mix("#ff2e63", "#zzzzzz", 0.5) != "#ff2e63" {
		t.Error("a malformed second hex should return the first unchanged")
	}
}

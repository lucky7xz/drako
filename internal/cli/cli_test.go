package cli

import "testing"

func TestHandlerExitCodes(t *testing.T) {
	if got := HandleSummonCommand([]string{"drako", "summon"}); got != 1 {
		t.Errorf("summon without url = %d, want 1", got)
	}
	if got := HandleOpenCLI([]string{"drako", "open"}); got != 1 {
		t.Errorf("open without path = %d, want 1", got)
	}
}

func TestHandleCLI(t *testing.T) {
	clean := writeCheckFile(t, t.TempDir(), "good.profile.toml", cleanProfile)

	tests := []struct {
		name        string
		args        []string
		wantHandled bool
		wantCode    int
	}{
		{"no args proceeds to TUI", []string{"drako"}, false, 0},
		{"unknown command errors", []string{"drako", "nonsense"}, true, 1},
		{"version succeeds", []string{"drako", "version"}, true, 0},
		{"help succeeds", []string{"drako", "help"}, true, 0},
		{"check routes and succeeds", []string{"drako", "check", clean}, true, 0},
		{"explain without refs errors", []string{"drako", "explain"}, true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled, code := HandleCLI(tt.args)
			if handled != tt.wantHandled || code != tt.wantCode {
				t.Errorf("HandleCLI(%v) = (%v, %d), want (%v, %d)",
					tt.args, handled, code, tt.wantHandled, tt.wantCode)
			}
		})
	}
}

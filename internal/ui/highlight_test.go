package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// forceTrueColor pins lipgloss to truecolor for deterministic SGR output and
// restores the previous profile afterward — the profile is process-global, so
// leaking it would corrupt other tests' plain-text assertions.
func forceTrueColor(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// sgr is the truecolor SGR body lipgloss emits for a foreground hex, e.g.
// "#ff79c6" -> "38;2;255;121;198". Used to assert a token got its color.
func sgr(hex string) string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("x")
	return s[2:strings.IndexByte(s, 'm')] // between "\x1b[" and "m"
}

func TestHighlightShell_WidthPreserved(t *testing.T) {
	forceTrueColor(t)
	bg := lipgloss.Color("#282a36")
	cases := []string{
		"# a comment",
		`A='(^ *|[";|&(] *)'`,
		`BK="$CFG/sudo-backup-$(date +%Y%m%d-%H%M%S)"`,
		`echo "🚨 Sudo Switch — $CFG"`,
		"if [ \"$total\" -eq 0 ]; then",
		"",
		"    total=42",
	}
	for _, ln := range cases {
		got := lipgloss.Width(highlightShell(ln, bg))
		want := lipgloss.Width(ln) + 2 // matches valueStyle's Padding(0,1)
		if got != want {
			t.Errorf("width(%q) = %d, want %d", ln, got, want)
		}
	}
}

func TestHighlightShell_RunesPreserved(t *testing.T) {
	forceTrueColor(t)
	bg := lipgloss.Color("#282a36")
	cases := []string{
		"# comment",
		`echo "🚨 $CFG" ${BK} $(date) 'lit'`,
		"for f in a b c; do done",
	}
	for _, ln := range cases {
		stripped := ansi.Strip(highlightShell(ln, bg))
		if want := " " + ln + " "; stripped != want {
			t.Errorf("stripped = %q, want %q", stripped, want)
		}
	}
}

func TestHighlightShell_ColorsApplied(t *testing.T) {
	forceTrueColor(t)
	bg := lipgloss.Color("#282a36")

	tests := []struct {
		name string
		line string
		hex  string
	}{
		{"comment", "# hello world", string(hlComment)},
		{"keyword", "if true; then", string(hlKeyword)},
		{"variable", "echo $CFG", string(hlVariable)},
		{"cmd-subst", "x=$(date)", string(hlVariable)},
		{"string", "A='literal text'", string(hlString)},
		{"number", "n=42 rest", string(hlNumber)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := highlightShell(tc.line, bg)
			if !strings.Contains(out, sgr(tc.hex)) {
				t.Errorf("%q: expected %s color (%s) in output", tc.line, tc.name, tc.hex)
			}
		})
	}
}

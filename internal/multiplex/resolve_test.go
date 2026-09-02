package multiplex

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// onPath builds an Env whose LookPath finds exactly the named binaries.
func onPath(tmux, herdr string, installed ...string) Env {
	return Env{Tmux: tmux, Herdr: herdr, LookPath: func(name string) (string, error) {
		if slices.Contains(installed, name) {
			return "/usr/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}}
}

func TestResolve(t *testing.T) {
	const in = "/tmp/tmux-1000/default,123,0" // a plausible $TMUX

	cases := []struct {
		name  string
		force bool
		env   Env
		want  string // backend Name(), or "" when batch is unavailable
	}{
		// The default follows the multiplexer drako is already inside, so a
		// batch never nests one in the other.
		{"inside herdr", false, onPath("", "1", "tmux", "herdr"), "herdr"},
		{"inside tmux", false, onPath(in, "", "tmux", "herdr"), "tmux"},
		{"inside both", false, onPath(in, "1", "tmux", "herdr"), "herdr"},

		// Outside either, tmux is the only one that can start a session from
		// nothing, so it is the only thing reachable.
		{"outside, both installed", false, onPath("", "", "tmux", "herdr"), "tmux"},
		{"outside, only tmux", false, onPath("", "", "tmux"), "tmux"},
		{"outside, only herdr", false, onPath("", "", "herdr"), ""},
		{"nothing installed", false, onPath("", ""), ""},

		// The one override: tmux from inside herdr.
		{"forced from inside herdr", true, onPath("", "1", "tmux", "herdr"), "tmux"},
		{"forced with tmux missing", true, onPath("", "1", "herdr"), ""},
		{"forced changes nothing outside herdr", true, onPath("", "", "tmux"), "tmux"},

		// HERDR_ENV is 1 inside herdr; anything else is not inside.
		{"herdr env set to something else", false, onPath("", "0", "tmux", "herdr"), "tmux"},
	}

	for _, c := range cases {
		b, err := Resolve(c.force, c.env)
		switch {
		case c.want == "":
			if err == nil {
				t.Errorf("%s: got %s, want batch unavailable", c.name, b.Name())
			} else if !strings.HasPrefix(err.Error(), "Batch") {
				t.Errorf("%s: %q does not read as a status line", c.name, err)
			}
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case b.Name() != c.want:
			t.Errorf("%s: got %s, want %s", c.name, b.Name(), c.want)
		}
	}
}

// The tmux backend has to know whether it is joining drako's session or
// building a new one, and Resolve is the one place that reads $TMUX.
func TestResolve_TellsTmuxWhetherItIsNested(t *testing.T) {
	for _, c := range []struct {
		tmux string
		want bool
	}{{"/tmp/tmux-1000/default,123,0", true}, {"", false}} {
		b, err := Resolve(true, onPath(c.tmux, "", "tmux"))
		if err != nil {
			t.Fatal(err)
		}
		tmux, ok := b.(*Tmux)
		if !ok {
			t.Fatalf("got %T, want *Tmux", b)
		}
		if tmux.Inside != c.want {
			t.Errorf("$TMUX=%q gave Inside=%v, want %v", c.tmux, tmux.Inside, c.want)
		}
	}
}

// herdr cannot create a session from outside itself, so the refusal has to say
// that rather than "not installed" — the binary is right there.
func TestResolve_SaysWhyHerdrIsUnreachableFromOutside(t *testing.T) {
	_, err := Resolve(false, onPath("", "", "herdr"))
	if err == nil {
		t.Fatal("herdr outside herdr must be refused")
	}
	if !strings.Contains(err.Error(), "inside") {
		t.Errorf("%q should say drako must run inside herdr", err)
	}
}

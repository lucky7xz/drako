package config

import (
	"strings"
	"testing"
)

func findingsContain(fs []Finding, level, substr string) bool {
	for _, f := range fs {
		if f.Level == level && strings.Contains(f.Msg, substr) {
			return true
		}
	}
	return false
}

func TestCheckProfileCleanDeck(t *testing.T) {
	fs := CheckProfile([]byte(`
x = 2
y = 2
[[commands]]
name = "a"
command = "echo a"
col = "A"
row = 0
[[commands]]
name = "folder"
col = "B"
row = 1
items = [
  { name = "one", command = { linux_debian = "apt x", macos = "brew x" } },
]
`))
	if len(fs) != 0 {
		t.Errorf("clean deck flagged: %+v", fs)
	}
}

// The real-world cases this linter exists for.
func TestCheckProfileCatchesRealBugs(t *testing.T) {
	t.Run("unterminated string (the crowdsec case)", func(t *testing.T) {
		fs := CheckProfile([]byte("x = 1\ny = 1\n[[commands]]\nname = \"broken\n"))
		if !findingsContain(fs, "error", "parse error at line") {
			t.Errorf("missing parse finding: %+v", fs)
		}
	})

	t.Run("typo'd variant key (the acos case)", func(t *testing.T) {
		fs := CheckProfile([]byte(`
x = 1
y = 1
[[commands]]
name = "path"
command = { linux_generic = "ok", acos = "broken silently" }
col = "A"
row = 0
`))
		if !findingsContain(fs, "error", `unknown variant key "acos"`) {
			t.Errorf("missing variant finding: %+v", fs)
		}
	})

	t.Run("duplicate names", func(t *testing.T) {
		fs := CheckProfile([]byte(`
x = 1
y = 2
[[commands]]
name = "twin"
command = "echo 1"
col = "A"
row = 0
[[commands]]
name = "twin"
command = "echo 2"
col = "A"
row = 1
`))
		if !findingsContain(fs, "error", "duplicate command name") {
			t.Errorf("missing duplicate-name finding: %+v", fs)
		}
	})

	t.Run("cell collision", func(t *testing.T) {
		fs := CheckProfile([]byte(`
x = 1
y = 1
[[commands]]
name = "first"
command = "echo 1"
col = "A"
row = 0
[[commands]]
name = "second"
command = "echo 2"
col = "a"
row = 0
`))
		if !findingsContain(fs, "error", "share cell A0") {
			t.Errorf("missing collision finding: %+v", fs)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		fs := CheckProfile([]byte(`
x = 1
y = 1
[[commands]]
name = "far"
command = "echo"
col = "C"
row = 5
`))
		if !findingsContain(fs, "error", "column") || !findingsContain(fs, "error", "row") {
			t.Errorf("missing bounds findings: %+v", fs)
		}
	})
}

func TestCheckProfileShapes(t *testing.T) {
	t.Run("neither command nor items", func(t *testing.T) {
		fs := CheckProfile([]byte("x=1\ny=1\n[[commands]]\nname = \"empty\"\ncol = \"A\"\nrow = 0\n"))
		if !findingsContain(fs, "error", "neither a command nor items") {
			t.Errorf("missing finding: %+v", fs)
		}
	})

	t.Run("more than 9 items warns", func(t *testing.T) {
		items := make([]string, 10)
		for i := range items {
			items[i] = `{ name = "i", command = "x" },`
		}
		fs := CheckProfile([]byte("x=1\ny=1\n[[commands]]\nname = \"big\"\ncol = \"A\"\nrow = 0\nitems = [" + strings.Join(items, "\n") + "]\n"))
		if !findingsContain(fs, "warning", "dropdowns support at most 9") {
			t.Errorf("missing item-count warning: %+v", fs)
		}
	})

	t.Run("z column resolves against grid", func(t *testing.T) {
		fs := CheckProfile([]byte(`
x = 3
y = 1
[[commands]]
name = "last col"
command = "echo"
col = "z"
row = 0
`))
		if len(fs) != 0 {
			t.Errorf("z column wrongly flagged: %+v", fs)
		}
	})

	t.Run("items via [[commands.items]] headers", func(t *testing.T) {
		fs := CheckProfile([]byte(`
x = 1
y = 1
[[commands]]
name = "folder"
col = "A"
row = 0

[[commands.items]]
name = "one"
command = { windos = "typo" }
`))
		if !findingsContain(fs, "error", `unknown variant key "windos"`) {
			t.Errorf("header-syntax items not linted: %+v", fs)
		}
	})

	t.Run("no commands at all", func(t *testing.T) {
		fs := CheckProfile([]byte("x = 1\ny = 1\n"))
		if !findingsContain(fs, "error", "no [[commands]]") {
			t.Errorf("missing finding: %+v", fs)
		}
	})

	t.Run("uppercase X Y respected", func(t *testing.T) {
		fs := CheckProfile([]byte(`
X = 1
Y = 1
[[commands]]
name = "far"
command = "echo"
col = "B"
row = 0
`))
		if !findingsContain(fs, "error", "grid is only 1 wide") {
			t.Errorf("uppercase grid size ignored: %+v", fs)
		}
	})
}

package config

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func pinPlatform(t *testing.T, target string) {
	t.Helper()
	orig := runtimeTargetFn
	runtimeTargetFn = func() string { return target }
	t.Cleanup(func() { runtimeTargetFn = orig })
}

func decodeProfile(t *testing.T, src string) ProfileFile {
	t.Helper()
	var p ProfileFile
	if _, err := toml.Decode(src, &p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return p
}

func TestCommandVariants(t *testing.T) {
	src := `
x = 1
y = 3
[[commands]]
name = "plain"
command = "echo same everywhere"
col = "A"
row = 0
[[commands]]
name = "varied"
command = { linux_arch = "pacman -Syu", linux_debian = "apt upgrade", macos = "brew upgrade" }
col = "A"
row = 1
[[commands]]
name = "generic-fallback"
command = { linux_generic = "generic linux", windows = "win" }
col = "A"
row = 2
`
	t.Run("arch resolves its variant", func(t *testing.T) {
		pinPlatform(t, "linux_arch")
		p := decodeProfile(t, src)
		if p.Commands[0].Command != "echo same everywhere" {
			t.Errorf("plain string mangled: %q", p.Commands[0].Command)
		}
		if p.Commands[1].Command != "pacman -Syu" {
			t.Errorf("variant = %q, want pacman", p.Commands[1].Command)
		}
		if p.Commands[2].Command != "generic linux" {
			t.Errorf("generic fallback = %q", p.Commands[2].Command)
		}
	})

	t.Run("macos resolves its variant, no linux fallback", func(t *testing.T) {
		pinPlatform(t, "macos")
		p := decodeProfile(t, src)
		if p.Commands[1].Command != "brew upgrade" {
			t.Errorf("variant = %q, want brew", p.Commands[1].Command)
		}
		// macos must NOT fall through to linux_generic
		if p.Commands[2].Command != "" {
			t.Errorf("macos wrongly matched linux_generic: %q", p.Commands[2].Command)
		}
		if !strings.Contains(p.Commands[2].Description, "no command variant") ||
			!strings.Contains(p.Commands[2].Description, "linux_generic, windows") {
			t.Errorf("missing/incomplete note: %q", p.Commands[2].Description)
		}
	})
}

// A command-folder cell: every item resolves independently.
func TestItemVariants(t *testing.T) {
	pinPlatform(t, "linux_debian")
	p := decodeProfile(t, `
x = 1
y = 1
[[commands]]
name = "folder"
description = "keeps its own description"
col = "A"
row = 0
items = [
  { name = "install", description = "d1", command = { linux_debian = "apt install x", linux_arch = "pacman -S x" } },
  { name = "plain", command = "echo hi" },
  { name = "elsewhere", command = { macos = "brew x" }, auto_close_execution = false },
]
`)
	items := p.Commands[0].Items
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	if items[0].Command != "apt install x" {
		t.Errorf("item variant = %q", items[0].Command)
	}
	if items[1].Command != "echo hi" {
		t.Errorf("plain item = %q", items[1].Command)
	}
	if items[2].Command != "" || !strings.Contains(items[2].Description, "macos") {
		t.Errorf("unresolved item: cmd=%q desc=%q", items[2].Command, items[2].Description)
	}
	if items[2].AutoCloseExecution == nil || *items[2].AutoCloseExecution {
		t.Error("item flags lost in manual decode")
	}
	if p.Commands[0].Description != "keeps its own description" {
		t.Errorf("parent description touched: %q", p.Commands[0].Description)
	}
}

// Items written as [[commands.items]] headers arrive as []map[string]any
// instead of []any — both syntaxes must decode.
func TestItemsHeaderSyntax(t *testing.T) {
	pinPlatform(t, "linux_arch")
	p := decodeProfile(t, `
x = 1
y = 1
[[commands]]
name = "folder"
col = "A"
row = 0

[[commands.items]]
name = "one"
command = { linux_arch = "pacman -Q", macos = "brew list" }

[[commands.items]]
name = "two"
command = "echo hi"
`)
	items := p.Commands[0].Items
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Command != "pacman -Q" || items[1].Command != "echo hi" {
		t.Errorf("header-syntax items misdecoded: %+v", items)
	}
}

func TestVariantErrors(t *testing.T) {
	pinPlatform(t, "linux_arch")
	var p ProfileFile
	if _, err := toml.Decode(`
x = 1
y = 1
[[commands]]
name = "bad"
command = { linux_arch = 42 }
col = "A"
row = 0
`, &p); err == nil || !strings.Contains(err.Error(), "must be a string") {
		t.Errorf("non-string variant accepted: %v", err)
	}

	if _, err := toml.Decode(`
x = 1
y = 1
[[commands]]
name = "worse"
command = 42
col = "A"
row = 0
`, &p); err == nil {
		t.Error("numeric command accepted")
	}
}

// The manual decode must keep every field the struct carries.
func TestManualDecodeFieldParity(t *testing.T) {
	pinPlatform(t, "linux_arch")
	p := decodeProfile(t, `
x = 2
y = 2
[[commands]]
name = "full"
description = "desc"
command = "run"
col = "B"
row = 1
auto_close_execution = false
debug_execution = true
`)
	c := p.Commands[0]
	if c.Name != "full" || c.Description != "desc" || c.Command != "run" ||
		c.Col != "B" || c.Row != 1 ||
		c.AutoCloseExecution == nil || *c.AutoCloseExecution ||
		c.DebugExecution == nil || !*c.DebugExecution {
		t.Errorf("field parity broken: %+v", c)
	}
}

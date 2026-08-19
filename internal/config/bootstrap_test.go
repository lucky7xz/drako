package config

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// The embedded core deck must fully resolve on every platform drako ships
// variants for: each cell either runs something or opens a folder whose items
// all run something. A "[no command variant …]" note in any description means
// a platform was forgotten in bootstrap/core.profile.toml.
func TestEmbeddedCoreDeckResolvesEverywhere(t *testing.T) {
	deck, err := bootstrapFS.ReadFile("bootstrap/core.profile.toml")
	if err != nil {
		t.Fatalf("embedded core deck missing: %v", err)
	}

	// Deliberately empty on some platforms — not a coverage gap.
	knownEmpty := map[string]bool{}

	targets := []string{
		"linux_debian", "linux_arch", "linux_fedora", "linux_void",
		"linux_immutable", "linux_generic", "macos", "windows",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			pinPlatform(t, target)
			var p ProfileFile
			if _, err := toml.Decode(string(deck), &p); err != nil {
				t.Fatalf("core deck does not parse: %v", err)
			}
			if len(p.Commands) == 0 {
				t.Fatal("core deck decoded to zero commands")
			}
			for _, c := range p.Commands {
				if strings.Contains(c.Description, "no command variant") {
					t.Errorf("cell %q unresolved on %s", c.Name, target)
				}
				if len(c.Items) == 0 && strings.TrimSpace(c.Command) == "" {
					t.Errorf("cell %q has neither command nor items on %s", c.Name, target)
				}
				for _, it := range c.Items {
					if strings.Contains(it.Description, "no command variant") {
						t.Errorf("item %q/%q unresolved on %s", c.Name, it.Name, target)
					}
					if strings.TrimSpace(it.Command) == "" && !knownEmpty[target+"/"+strings.TrimSpace(it.Name)] {
						t.Errorf("item %q/%q empty on %s", c.Name, it.Name, target)
					}
				}
			}
		})
	}
}

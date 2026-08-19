package config

import (
	"strings"
	"testing"
)

// The immutable key is an override, not a replacement: cells that install
// something get brew, everything else still answers from the base distro.
func TestImmutableOverridesOnlyWhatItDefines(t *testing.T) {
	pinPlatformChain(t, ImmutableKey, "linux_arch")

	p := decodeProfile(t, `
x = 1
y = 1
[[commands]]
name = "Packages"
col = "a"
row = 0
command = { linux_immutable = "brew install x", linux_arch = "pacman -S x" }
[[commands]]
name = "Not packages"
col = "b"
row = 0
command = { linux_arch = "arch-only", linux_generic = "generic" }
[[commands]]
name = "Generic only"
col = "c"
row = 0
command = { linux_generic = "generic" }
`)

	for i, want := range []string{"brew install x", "arch-only", "generic"} {
		if got := p.Commands[i].Command; got != want {
			t.Errorf("%s: got %q, want %q", p.Commands[i].Name, got, want)
		}
	}
}

// A mutable Arch box must never pick up an immutable variant.
func TestNonImmutableIgnoresImmutableVariant(t *testing.T) {
	pinPlatform(t, "linux_arch")

	p := decodeProfile(t, `
x = 1
y = 1
[[commands]]
name = "Packages"
col = "a"
row = 0
command = { linux_immutable = "brew install x", linux_arch = "pacman -S x" }
`)

	if got := p.Commands[0].Command; got != "pacman -S x" {
		t.Errorf("got %q, want the arch variant", got)
	}
}

// On an image-managed host the distro's package manager is not how software
// gets installed, so no resolved command may reach for one — including via
// the linux_generic probe, which would find the base distro's manager.
func TestCoreDeckNeverUsesASystemPackageManagerWhenImmutable(t *testing.T) {
	deck, err := bootstrapFS.ReadFile("bootstrap/core.profile.toml")
	if err != nil {
		t.Fatalf("embedded core deck missing: %v", err)
	}

	// A Steam Deck: immutable, Arch underneath.
	pinPlatformChain(t, ImmutableKey, "linux_arch")
	p := decodeProfile(t, string(deck))

	managers := []string{"apt", "dnf", "pacman", "zypper", "xbps", "rpm-ostree"}
	brewCells := 0

	check := func(name, cmd string) {
		if !strings.Contains(cmd, "brew") {
			return // not a package cell on this platform
		}
		brewCells++
		for _, pm := range managers {
			if strings.Contains(cmd, pm) {
				t.Errorf("%s: reaches for %s on an immutable host:\n%s", name, pm, cmd)
			}
		}
	}

	for _, c := range p.Commands {
		check(c.Name, c.Command)
		for _, item := range c.Items {
			check(c.Name+" > "+item.Name, item.Command)
		}
	}

	// Guard the guard: without this the test would pass on a deck that had
	// lost its brew variants entirely.
	if brewCells < 4 {
		t.Errorf("found %d brew-backed cells, want at least 4", brewCells)
	}
}

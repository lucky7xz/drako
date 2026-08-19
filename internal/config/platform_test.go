package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// pinMarkers swaps the immutability probes for paths a test controls.
func pinMarkers(t *testing.T, markers ...string) {
	t.Helper()
	orig := immutableMarkers
	immutableMarkers = markers
	t.Cleanup(func() { immutableMarkers = orig })
}

// Immutability is probed from the filesystem rather than matched against a
// distro list, so one marker covers a whole family: /run/ostree-booted is
// every Fedora Atomic image, not just the one that prompted this.
func TestIsImmutableDetectsEachMarker(t *testing.T) {
	for _, name := range []string{"ostree-booted", "steamos-readonly"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, name)
			if err := os.WriteFile(marker, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			// Present alongside one that does not exist: any marker is enough.
			pinMarkers(t, filepath.Join(dir, "absent"), marker)

			if !isImmutable() {
				t.Error("isImmutable() = false with the marker present")
			}
		})
	}
}

func TestIsImmutableFalseWithoutMarkers(t *testing.T) {
	pinMarkers(t, filepath.Join(t.TempDir(), "absent"))

	if isImmutable() {
		t.Error("isImmutable() = true with no marker present")
	}
}

// The immutable key is an override, not a replacement: the base distro stays
// in the chain so cells that aren't installing anything still resolve.
func TestDetectRuntimeTargetsChain(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("chain shape is linux-specific")
	}

	pinMarkers(t, filepath.Join(t.TempDir(), "absent"))
	plain := detectRuntimeTargets()
	if len(plain) != 1 {
		t.Fatalf("plain host targets = %v, want just the base distro", plain)
	}
	base := plain[0]

	dir := t.TempDir()
	marker := filepath.Join(dir, "ostree-booted")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	pinMarkers(t, marker)

	got := detectRuntimeTargets()
	want := []string{ImmutableKey, base}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("immutable host targets = %v, want %v", got, want)
	}
}

// Termux is no longer special-cased: it lands on a Debian key or the generic
// fallback, where sudo is the only thing wrong and the Sudo Switch strips it.
func TestTermuxIsNotSpecialCased(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")

	if got := detectLinuxDistro(); got == "linux_termux" {
		t.Errorf("detectLinuxDistro() = %q, want the base distro or linux_generic", got)
	}
}

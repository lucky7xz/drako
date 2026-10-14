package config

import (
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"
)

// Platform detection for per-platform command variants: which variant key
// (linux_debian, macos, windows, …) describes the machine drako is running on.

// DistroKeywords maps a runtime target key to a list of identifying strings found in /etc/os-release.
// To add a new distro, simply append its keywords to the appropriate list or create a new entry.
var DistroKeywords = map[string][]string{
	"linux_arch":   {"arch", "manjaro", "endeavouros", "cachy", "garuda", "omarchy"},
	"linux_fedora": {"fedora", "rhel", "centos", "nobara"},
	"linux_debian": {"debian", "ubuntu", "pop", "mint", "kali", "mx", "zorin", "elementary", "neon"},
	"linux_suse":   {"suse", "opensuse", "sles"},
	"linux_void":   {"void"},
	"linux_nixos":  {"nixos"},
}

// ImmutableKey is tried ahead of the base distro on image-managed systems.
const ImmutableKey = "linux_immutable"

// immutableMarkers are the files whose presence means the root filesystem is
// image-managed. Capability probes rather than a distro list: the ostree
// marker covers the whole Fedora Atomic family (Silverblue, Kinoite, Bazzite,
// Bluefin, Aurora), not just the one distro that prompted this.
var immutableMarkers = []string{
	"/run/ostree-booted",        // ostree / rpm-ostree systems
	"/usr/bin/steamos-readonly", // SteamOS: A/B read-only root, no ostree
}

// isImmutable reports whether the distro's package manager is not the way
// software gets installed here.
func isImmutable() bool {
	for _, marker := range immutableMarkers {
		if _, err := os.Stat(marker); err == nil {
			return true
		}
	}
	return false
}

// detectRuntimeTargets lists the variant keys to try, most specific first.
// An immutable host prepends linux_immutable: brew replaces its package
// manager, but it is still a Fedora or Arch box underneath, so the base key
// stays in the chain for every cell that isn't installing something.
func detectRuntimeTargets() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"windows"}
	case "darwin":
		return []string{"macos"}
	case "linux":
		base := detectLinuxDistro()
		if isImmutable() {
			return []string{ImmutableKey, base}
		}
		return []string{base}
	default:
		return []string{"linux_generic"}
	}
}

// detectLinuxDistro identifies the base distro. Termux (Android) is not
// special-cased: it lands on a Debian key or the generic fallback, and the
// only thing wrong there is sudo, which the Sudo Switch strips.
func detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux_generic"
	}
	return distroFromOSRelease(string(data))
}

// distroFromOSRelease matches ID, then each ID_LIKE entry, by prefix in a fixed
// family order: a URL or name that holds another family's keyword never decides.
func distroFromOSRelease(data string) string {
	fields := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			fields[k] = strings.ToLower(strings.Trim(v, `"'`))
		}
	}
	families := slices.Sorted(maps.Keys(DistroKeywords))
	for _, id := range append([]string{fields["ID"]}, strings.Fields(fields["ID_LIKE"])...) {
		for _, key := range families {
			for _, kw := range DistroKeywords[key] {
				if id != "" && strings.HasPrefix(id, kw) {
					return key
				}
			}
		}
	}
	return "linux_generic"
}

package config

import (
	"os"
	"runtime"
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
}

func detectRuntimeTarget() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macos"
	case "linux":
		return detectLinuxDistro()
	default:
		return "linux_generic"
	}
}

func detectLinuxDistro() string {
	// Termux (Android) ships no /etc/os-release and no sudo — $PREFIX belongs
	// to the app's own uid, so package commands need no privilege at all.
	// Checked first: a proot distro inside Termux sets neither var and keeps
	// matching its real distro below, which is what it wants.
	if os.Getenv("TERMUX_VERSION") != "" || strings.Contains(os.Getenv("PREFIX"), "com.termux") {
		return "linux_termux"
	}

	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux_generic"
	}
	content := strings.ToLower(string(data))

	// Check against our map
	for key, keywords := range DistroKeywords {
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				return key
			}
		}
	}

	// No specific distro matched
	return "linux_generic"
}

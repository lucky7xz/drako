package config

import "testing"

// Termux is detected from the environment, not /etc/os-release: Android has no
// such file, and the linux_termux variants drop the sudo the generic fallback
// would otherwise reach for.
func TestDetectLinuxDistroTermux(t *testing.T) {
	cases := map[string]struct{ termuxVersion, prefix string }{
		"TERMUX_VERSION set": {termuxVersion: "0.118.0"},
		"termux PREFIX":      {prefix: "/data/data/com.termux/files/usr"},
		"both set":           {termuxVersion: "0.118.0", prefix: "/data/data/com.termux/files/usr"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TERMUX_VERSION", tc.termuxVersion)
			t.Setenv("PREFIX", tc.prefix)
			if got := detectLinuxDistro(); got != "linux_termux" {
				t.Errorf("detectLinuxDistro() = %q, want linux_termux", got)
			}
		})
	}
}

// A plain Linux box must never resolve to Termux, whatever its distro is —
// including one that sets an unrelated PREFIX (a common autotools habit).
func TestDetectLinuxDistroNonTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("PREFIX", "/usr/local")
	if got := detectLinuxDistro(); got == "linux_termux" {
		t.Error("detectLinuxDistro() = linux_termux without any Termux marker")
	}
}

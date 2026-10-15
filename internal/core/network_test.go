package core

import "testing"

func TestIsVirtualInterface(t *testing.T) {
	for name, want := range map[string]bool{
		"lo": true, "Loopback Pseudo-Interface 1": true, "docker0": true, "virbr0": true,
		"wg0": true, "utun3": true, "tailscale0": true,
		"eth0": false, "wlp0s20f3": false, "en0": false, "Wi-Fi": false,
	} {
		if got := isVirtualInterface(name); got != want {
			t.Errorf("isVirtualInterface(%q) = %v, want %v", name, got, want)
		}
	}
}

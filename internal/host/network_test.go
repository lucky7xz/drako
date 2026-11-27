package host

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

// A unit takes over only above the one below: exactly 1 KB/s still reads in bytes.
func TestFormatTraffic(t *testing.T) {
	for bps, want := range map[float64]string{
		0:               "0 B/s",
		1024:            "1024 B/s",
		1536:            "1.50 KB/s",
		1024 * 1024:     "1024.00 KB/s",
		5 * 1024 * 1024: "5.00 MB/s",
	} {
		if got := FormatTraffic(bps); got != want {
			t.Errorf("FormatTraffic(%v) = %q, want %q", bps, got, want)
		}
	}
}

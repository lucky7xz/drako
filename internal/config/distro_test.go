package config

import "testing"

// The distro comes from ID, then each ID_LIKE entry, by prefix: another family's
// keyword in a URL or name never decides, and every call agrees.
func TestDistroFromOSRelease(t *testing.T) {
	cases := map[string]string{
		"ID=pop\nID_LIKE=\"ubuntu debian\"\nHOME_URL=\"https://pop.system76.com\"\n": "linux_debian",
		"ID=fedora\nHOME_URL=\"https://archive.fedoraproject.org/\"\n":               "linux_fedora",
		"ID=cachyos\nID_LIKE=arch\n":                                                 "linux_arch",
		"ID=\"opensuse-tumbleweed\"\nID_LIKE=\"opensuse suse\"\n":                    "linux_suse",
		"ID=bazzite\nID_LIKE=\"fedora\"\nVARIANT_ID=bazzite-deck\n":                  "linux_fedora",
		"ID=plan9\nNAME=\"popular archive\"\n":                                       "linux_generic",
	}
	for content, want := range cases {
		for range 200 { // map order must not matter
			if got := distroFromOSRelease(content); got != want {
				t.Errorf("os-release %q: got %s, want %s", content, got, want)
				break
			}
		}
	}
}

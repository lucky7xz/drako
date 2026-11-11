package config

import (
	"testing"

	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/profile"
)

// pinned is the platform chain decodeProfile resolves for; nil means the
// machine the tests run on.
var pinned []string

func pinPlatform(t *testing.T, target string) {
	t.Helper()
	pinPlatformChain(t, target)
}

// pinPlatformChain pins the whole resolution chain, most specific first —
// what an immutable host produces (linux_immutable, then its base distro).
func pinPlatformChain(t *testing.T, targets ...string) {
	t.Helper()
	orig := pinned
	pinned = targets
	t.Cleanup(func() { pinned = orig })
}

// decodeFor decodes src for the pinned platform; what validation would say
// about it is not this helper's business.
func decodeFor(src string) (profile.File, error) {
	targets := pinned
	if targets == nil {
		targets = host.Targets()
	}
	pf, _, err := profile.Decode([]byte(src), targets)
	return pf, err
}

func decodeProfile(t *testing.T, src string) profile.File {
	t.Helper()
	p, err := decodeFor(src)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return p
}

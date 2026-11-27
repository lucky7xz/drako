package version

import (
	"strings"
	"testing"
)

// A stamped release reports its tag; any other build reports a dev version.
func TestString(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	version = "v1.2.3"
	if got := String(); got != "v1.2.3" {
		t.Errorf("stamped: got %q", got)
	}

	version = ""
	if got := String(); !strings.HasPrefix(got, "dev") {
		t.Errorf("unstamped test build: got %q, want dev…", got)
	}
}

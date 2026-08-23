package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// symlinkedRepo builds a repo directory plus a symlink pointing at it, standing
// in for a home directory that lives behind a link (/home -> /var/home on
// Fedora atomic). Returns the link, which is how the caller names the repo.
func symlinkedRepo(t *testing.T) (root, link string) {
	t.Helper()
	root = t.TempDir()
	real := filepath.Join(root, "realrepo")
	if err := os.MkdirAll(filepath.Join(real, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return root, link
}

// An asset that really is inside the repo must be accepted even when the repo
// is named through a symlink. Resolving only the target left every asset
// looking like an escape, so summon reported them all [missing].
func TestIsPathWithinBase_BaseBehindSymlink(t *testing.T) {
	_, link := symlinkedRepo(t)
	asset := filepath.Join(link, "sub", "serve.sh")
	if err := os.WriteFile(asset, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := isPathWithinBase(link, asset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("asset inside the repo was rejected because the repo path contains a symlink")
	}
}

// The escape check itself must survive that: an asset symlinked out of the repo
// stays rejected, symlinked base or not.
func TestIsPathWithinBase_RejectsEscapeFromSymlinkedBase(t *testing.T) {
	root, link := symlinkedRepo(t)
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("s3cret"), 0o600); err != nil {
		t.Fatal(err)
	}
	escaping := filepath.Join(link, "evil.sh")
	if err := os.Symlink(secret, escaping); err != nil {
		t.Fatal(err)
	}

	ok, _ := isPathWithinBase(link, escaping)
	if ok {
		t.Error("a symlink pointing outside the repo was accepted")
	}
}

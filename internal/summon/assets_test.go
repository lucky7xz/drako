package summon

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
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

	ok, err := IsPathWithinBase(link, asset)
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

	ok, _ := IsPathWithinBase(link, escaping)
	if ok {
		t.Error("a symlink pointing outside the repo was accepted")
	}
}

// The plan and the copy say the same about every declared asset: a path that
// leaves the repository is refused, one that is not there is missing, and
// neither is ever opened.
func TestAssetsRefusedAndMissingAreToldApart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating links needs privileges on Windows")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "outside.txt"), "outside")
	write(filepath.Join(repo, "note.txt"), "note")
	write(filepath.Join(repo, "scripts", "run.sh"), "run")
	if err := os.Symlink(filepath.Join(root, "outside.txt"), filepath.Join(repo, "link-out.txt")); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(filepath.Join(repo, "scripts", "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	declared := []string{"scripts", "note.txt", "../outside.txt", "link-out.txt", "nope.txt"}
	want := map[string]string{
		"scripts": "copy", "note.txt": "copy",
		"../outside.txt": "refused", "link-out.txt": "refused", "nope.txt": "missing",
	}
	for _, p := range PlanAssets(repo, repo, declared) {
		got := "copy"
		switch {
		case p.Refused:
			got = "refused"
		case p.Missing:
			got = "missing"
		}
		if got != want[p.AssetRel] {
			t.Errorf("plan: %s is %s, want %s", p.AssetRel, got, want[p.AssetRel])
		}
		// The plan names what arrives executable: the script, not the note.
		if wantExec := map[string]int{"scripts": 1}[p.AssetRel]; p.Executable != wantExec {
			t.Errorf("plan: %s has %d executable, want %d", p.AssetRel, p.Executable, wantExec)
		}
	}
	var installed []string
	copied, skipped, missing, _ := CopyAssets(repo, repo, declared, AssetSink{
		Install: func(rel string, r io.Reader, executable bool) error {
			installed = append(installed, filepath.ToSlash(rel))
			// The execute bit travels with the file: the script has it, the note does not.
			if wantExec := strings.HasSuffix(rel, ".sh"); executable != wantExec {
				t.Errorf("%s: executable=%v, want %v", rel, executable, wantExec)
			}
			return nil
		},
		Large: func(string, int64, bool) {},
	})
	if copied != 2 || skipped != 2 || missing != 1 {
		t.Errorf("copied=%d skipped=%d missing=%d, want 2, 2 (the two refused), 1", copied, skipped, missing)
	}
	slices.Sort(installed)
	if !slices.Equal(installed, []string{"note.txt", "scripts/run.sh"}) {
		t.Errorf("installed %v, want only the two files inside the repository", installed)
	}
}

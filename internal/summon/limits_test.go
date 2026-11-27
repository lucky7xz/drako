package summon

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// Each limit refuses on its own, and a file exactly at a limit still fits.
func TestCheckAssetFileAllowed(t *testing.T) {
	cases := []struct {
		name  string
		size  int64
		count int
		total int64
		want  bool
	}{
		{"small", 10, 0, 0, true},
		{"at the file limit", AssetMaxFileBytes, 0, 0, true},
		{"over the file limit", AssetMaxFileBytes + 1, 0, 0, false},
		{"last file that fits the count", 1, AssetMaxFileCount - 1, 0, true},
		{"one file too many", 1, AssetMaxFileCount, 0, false},
		{"fills the total exactly", 10, 0, AssetMaxTotalBytes - 10, true},
		{"over the total", 11, 0, AssetMaxTotalBytes - 10, false},
	}
	for _, c := range cases {
		if got := checkAssetFileAllowed(c.size, c.count, c.total); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// Oversized files and links are skipped, inside a declared folder or declared
// on their own; a large file is announced before it is handed over.
func TestCopyAssetsLimits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating links needs privileges on Windows")
	}
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	sized := func(rel string, size int64) {
		t.Helper()
		f, err := os.Create(filepath.Join(repo, rel))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := f.Truncate(size); err != nil { // sparse: no real disk use
			t.Fatal(err)
		}
	}
	sized("assets/small.txt", 3)
	sized("assets/large.bin", assetWarnSizeBytes+1)
	sized("assets/huge.bin", AssetMaxFileBytes+1)
	sized("huge.bin", AssetMaxFileBytes+1)
	if err := os.Symlink("small.txt", filepath.Join(repo, "assets", "link.txt")); err != nil {
		t.Fatal(err)
	}

	var installed, large []string
	sink := AssetSink{
		Install: func(rel string, r io.Reader, executable bool) error {
			installed = append(installed, rel)
			return nil
		},
		Large: func(name string, size int64, inFolder bool) {
			if !inFolder {
				t.Errorf("%s: inFolder false for a file in a declared folder", name)
			}
			large = append(large, name)
		},
	}
	copied, skipped, missing, bytes := CopyAssets(repo, repo, []string{"assets", "huge.bin"}, sink)

	slices.Sort(installed)
	if want := []string{filepath.Join("assets", "large.bin"), filepath.Join("assets", "small.txt")}; !slices.Equal(installed, want) {
		t.Errorf("installed %v, want %v", installed, want)
	}
	if !slices.Equal(large, []string{"large.bin"}) {
		t.Errorf("announced as large %v, want [large.bin]", large)
	}
	if copied != 2 || skipped != 3 || missing != 0 {
		t.Errorf("copied/skipped/missing = %d/%d/%d, want 2/3/0", copied, skipped, missing)
	}
	if want := int64(assetWarnSizeBytes + 1 + 3); bytes != want {
		t.Errorf("bytes %d, want %d", bytes, want)
	}
}

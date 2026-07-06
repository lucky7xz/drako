package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSummonArgs(t *testing.T) {
	goodSha := strings.Repeat("a", 64)
	goodRev := strings.Repeat("b", 40)

	tests := []struct {
		name    string
		args    []string
		url     string
		sha     string
		rev     string
		wantErr string
	}{
		{name: "url only", args: []string{"https://x/p.profile.toml"}, url: "https://x/p.profile.toml"},
		{name: "sha after url", args: []string{"https://x/p", "--sha256", goodSha}, url: "https://x/p", sha: goodSha},
		{name: "sha before url", args: []string{"--sha256", goodSha, "https://x/p"}, url: "https://x/p", sha: goodSha},
		{name: "equals form", args: []string{"https://x/p", "--sha256=" + goodSha}, url: "https://x/p", sha: goodSha},
		{name: "rev for git", args: []string{"git@h:r.git", "--rev", goodRev}, url: "git@h:r.git", rev: goodRev},
		{name: "no url", args: []string{"--sha256", goodSha}, wantErr: "no source URL"},
		{name: "missing value", args: []string{"https://x/p", "--sha256"}, wantErr: "requires a value"},
		{name: "unknown flag", args: []string{"https://x/p", "--md5", "x"}, wantErr: "unknown flag"},
		{name: "two urls", args: []string{"https://x/a", "https://x/b"}, wantErr: "more than one"},
		{name: "truncated sha rejected", args: []string{"https://x/p", "--sha256", "abc123"}, wantErr: "64-character"},
		{name: "branch name rejected as rev", args: []string{"g@h:r.git", "--rev", "main"}, wantErr: "40-character"},
		{name: "non-hex rev rejected", args: []string{"g@h:r.git", "--rev", strings.Repeat("z", 40)}, wantErr: "40-character"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, sha, rev, err := parseSummonArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tt.url || sha != tt.sha || rev != tt.rev {
				t.Errorf("got (%q,%q,%q), want (%q,%q,%q)", url, sha, rev, tt.url, tt.sha, tt.rev)
			}
		})
	}
}

func TestVerifyFileSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	payload := []byte("deck bytes")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	good := hex.EncodeToString(sum[:])

	if err := verifyFileSHA256(path, good); err != nil {
		t.Errorf("correct hash rejected: %v", err)
	}
	if err := verifyFileSHA256(path, strings.ToUpper(good)); err != nil {
		t.Errorf("uppercase hash must match (case-insensitive): %v", err)
	}
	err := verifyFileSHA256(path, strings.Repeat("0", 64))
	if err == nil || !strings.Contains(err.Error(), "MISMATCH") {
		t.Errorf("wrong hash: err = %v, want MISMATCH", err)
	}
}

// End to end through Summon with the mock downloader: a correct pin admits
// the file to the inventory, a wrong pin keeps the inventory empty.
func TestSummonHTTPPin(t *testing.T) {
	payload := "x=3\ny=3\n[[commands]]\nname=\"T\"\ncommand=\"echo t\"\n"
	sum := sha256.Sum256([]byte(payload))
	goodPin := hex.EncodeToString(sum[:])

	newSummoner := func(dir, pin string) *Summoner {
		return &Summoner{
			ConfigDir: dir,
			Downloader: &MockDownloader{DownloadFunc: func(url, dst string) error {
				return os.WriteFile(dst, []byte(payload), 0o644)
			}},
			Cloner: &MockCloner{},
			UI:     &MockUI{},
			SHA256: pin,
		}
	}

	t.Run("matching pin summons", func(t *testing.T) {
		dir := t.TempDir()
		if err := newSummoner(dir, goodPin).Summon("https://x/team.profile.toml"); err != nil {
			t.Fatalf("Summon: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "inventory", "team.profile.toml")); err != nil {
			t.Errorf("expected file in inventory: %v", err)
		}
	})

	t.Run("wrong pin refuses before validation", func(t *testing.T) {
		dir := t.TempDir()
		err := newSummoner(dir, strings.Repeat("0", 64)).Summon("https://x/team.profile.toml")
		if err == nil || !strings.Contains(err.Error(), "MISMATCH") {
			t.Fatalf("err = %v, want MISMATCH", err)
		}
		entries, _ := os.ReadDir(filepath.Join(dir, "inventory"))
		if len(entries) != 0 {
			t.Errorf("inventory must stay empty on mismatch, has %d entries", len(entries))
		}
	})
}

// mkdirCloner fakes a clone by creating an (empty) checkout directory, so
// the post-verification file walk has something real to visit.
type mkdirCloner struct{}

func (mkdirCloner) CloneRepo(url, destDir string) error { return os.MkdirAll(destDir, 0o755) }
func (mkdirCloner) CheckGitAvailable() error            { return nil }

// Git pin: verified against the cloned HEAD before any file is examined.
func TestSummonGitRevPin(t *testing.T) {
	pinned := strings.Repeat("3", 40)
	origGitHead := gitHeadFn
	defer func() { gitHeadFn = origGitHead }()

	newSummoner := func(dir string) *Summoner {
		return &Summoner{ConfigDir: dir, Downloader: &MockDownloader{}, Cloner: mkdirCloner{}, UI: &MockUI{}, Rev: pinned}
	}

	t.Run("HEAD mismatch aborts before file discovery", func(t *testing.T) {
		gitHeadFn = func(dir string) (string, error) { return strings.Repeat("4", 40), nil }
		err := newSummoner(t.TempDir()).Summon("git@host:repo.git")
		if err == nil || !strings.Contains(err.Error(), "MISMATCH") {
			t.Fatalf("err = %v, want commit MISMATCH", err)
		}
	})

	t.Run("HEAD match proceeds to file discovery", func(t *testing.T) {
		gitHeadFn = func(dir string) (string, error) { return pinned, nil }
		err := newSummoner(t.TempDir()).Summon("git@host:repo.git")
		// The mock clone creates no files, so passing verification lands on
		// the empty-repo error — proof the pin gate was cleared.
		if err == nil || !strings.Contains(err.Error(), "no .profile.toml") {
			t.Fatalf("err = %v, want the empty-repo error after verification", err)
		}
	})
}

// Cross-flag misuse is refused with guidance.
func TestSummonPinFlagMismatch(t *testing.T) {
	s := &Summoner{ConfigDir: t.TempDir(), Downloader: &MockDownloader{}, Cloner: &MockCloner{}, UI: &MockUI{}}

	s.SHA256 = strings.Repeat("a", 64)
	if err := s.Summon("git@host:repo.git"); err == nil || !strings.Contains(err.Error(), "--rev") {
		t.Errorf("sha256 on git URL: err = %v, want guidance towards --rev", err)
	}

	s.SHA256, s.Rev = "", strings.Repeat("b", 40)
	if err := s.Summon("https://x/p.profile.toml"); err == nil || !strings.Contains(err.Error(), "--sha256") {
		t.Errorf("rev on http URL: err = %v, want guidance towards --sha256", err)
	}
}

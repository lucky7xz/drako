package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/summon"
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

// End to end through Summon with the mock downloader: a correct pin admits
// the deck, a wrong pin keeps it out of both equipped and the inventory.
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
		if _, err := os.Stat(filepath.Join(dir, "team.profile.toml")); err != nil {
			t.Errorf("expected the deck equipped: %v", err)
		}
	})

	t.Run("wrong pin refuses before validation", func(t *testing.T) {
		dir := t.TempDir()
		err := newSummoner(dir, strings.Repeat("0", 64)).Summon("https://x/team.profile.toml")
		if err == nil || !strings.Contains(err.Error(), "MISMATCH") {
			t.Fatalf("err = %v, want MISMATCH", err)
		}
		for _, d := range []string{dir, filepath.Join(dir, "inventory")} {
			if _, err := os.Stat(filepath.Join(d, "team.profile.toml")); err == nil {
				t.Errorf("the deck landed in %s despite the mismatch", d)
			}
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
	origGitHead, origCheckout := gitHeadFn, gitCheckoutFn
	defer func() { gitHeadFn, gitCheckoutFn = origGitHead, origCheckout }()
	gitCheckoutFn = func(dir, rev string) error { return nil }

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

// A pin names a commit, not a branch tip: it must still summon after the
// author pushes again.
func TestSummonGitRevChecksOutThePinnedCommit(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.git")
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", src, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	git("init", "-q")
	deck := filepath.Join(src, "deck.profile.toml")
	os.WriteFile(deck, []byte(validDeck), 0o644)
	git("add", ".")
	git("commit", "-qm", "v1")
	pinned := git("rev-parse", "HEAD")
	os.WriteFile(deck, []byte(validDeck+"# v2\n"), 0o644)
	git("commit", "-qam", "v2")

	s := &Summoner{ConfigDir: t.TempDir(), Downloader: &MockDownloader{}, Cloner: &summon.GitCloner{}, UI: &yesToAll{}, Rev: pinned}
	// A clone a Ctrl-C left behind in the old fixed spot must not get in the way.
	os.MkdirAll(filepath.Join(s.ConfigDir, "inventory", ".summon-temp"), 0o755)
	os.WriteFile(filepath.Join(s.ConfigDir, "inventory", ".summon-temp", "leftover"), nil, 0o644)
	if err := s.Summon(src); err != nil {
		t.Fatalf("Summon pinned to the older commit: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(s.ConfigDir, "deck.profile.toml"))
	if err != nil {
		t.Fatalf("pinned deck not summoned: %v", err)
	}
	if strings.Contains(string(got), "v2") {
		t.Error("summoned the branch tip instead of the pinned commit")
	}
}

// An unencrypted address (http://, git://) is refused unless pinned; a pinned
// one goes ahead, and the pin then decides.
func TestSummonUnencryptedNeedsPin(t *testing.T) {
	payload := "x=3\ny=3\n[[commands]]\nname=\"T\"\ncommand=\"echo t\"\n"
	sum := sha256.Sum256([]byte(payload))
	pin := hex.EncodeToString(sum[:])
	newSummoner := func() *Summoner {
		return &Summoner{
			ConfigDir: t.TempDir(),
			Downloader: &MockDownloader{DownloadFunc: func(url, dst string) error {
				return os.WriteFile(dst, []byte(payload), 0o644)
			}},
			Cloner: &MockCloner{},
			UI:     &MockUI{},
		}
	}

	for _, src := range []string{"http://x/p.profile.toml", "HTTP://x/p.profile.toml"} {
		if err := newSummoner().Summon(src); err == nil || !strings.Contains(err.Error(), "not encrypted") || !strings.Contains(err.Error(), "--sha256") {
			t.Errorf("unpinned %s: err = %v, want refusal pointing to --sha256", src, err)
		}
	}
	for _, src := range []string{"git://host/repo", "http://host/repo.git"} {
		if err := newSummoner().Summon(src); err == nil || !strings.Contains(err.Error(), "not encrypted") || !strings.Contains(err.Error(), "--rev") {
			t.Errorf("unpinned %s: err = %v, want refusal pointing to --rev", src, err)
		}
	}

	s := newSummoner()
	s.SHA256 = pin
	if err := s.Summon("http://x/p.profile.toml"); err != nil {
		t.Errorf("pinned http summon: %v", err)
	}
}

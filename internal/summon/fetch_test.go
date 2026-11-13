package summon

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lucky7xz/drako/internal/proc"
)

// The real downloader must leave the payload at dstPath. It used to stage to
// a .tmp file and delete it without ever writing dstPath, so real HTTP summon
// could never succeed while the injected test fake passed — this pins the
// real one.
func TestHTTPDownloaderWritesDestination(t *testing.T) {
	const payload = "x = 2\ny = 2\n[[commands]]\nname = \"a\"\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	dstPath := filepath.Join(t.TempDir(), "a.profile.toml")
	d := &HTTPDownloader{}
	if err := d.DownloadFile(srv.URL, dstPath); err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("destination file missing: %v", err)
	}
	if string(got) != payload {
		t.Errorf("content = %q, want %q", got, payload)
	}
}

// On failure dstPath must not survive — that's the downloader's contract.
func TestHTTPDownloaderRejectsOversized(t *testing.T) {
	big := strings.Repeat("a", ProfileMaxSize+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(big))
	}))
	defer srv.Close()

	dstPath := filepath.Join(t.TempDir(), "big.profile.toml")
	if err := (&HTTPDownloader{}).DownloadFile(srv.URL, dstPath); err == nil {
		t.Fatal("expected an error for an oversized download")
	}
	if _, err := os.Stat(dstPath); !os.IsNotExist(err) {
		t.Error("destination must not exist after a rejected download")
	}
}

func TestUnencrypted(t *testing.T) {
	for src, want := range map[string]bool{
		"http://x/p.profile.toml":  true,
		"HTTP://x/p.profile.toml":  true,
		"git://host/repo":          true,
		"http://host/repo.git":     true,
		"https://x/p.profile.toml": false,
		"https://host/repo.git":    false,
		"ssh://git@host/repo":      false,
		"git@host:user/repo.git":   false,
		"./decks":                  false,
	} {
		if got := Unencrypted(src); got != want {
			t.Errorf("Unencrypted(%q) = %v, want %v", src, got, want)
		}
	}
}

// A download that starts on HTTPS stays on HTTPS; one that starts on HTTP
// (only allowed when pinned) may go anywhere.
func TestKeepHTTPS(t *testing.T) {
	req := func(u string) *http.Request {
		r, err := http.NewRequest("GET", u, nil)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if err := keepHTTPS(req("http://evil/p"), []*http.Request{req("https://x/p")}); err == nil {
		t.Error("https -> http redirect was allowed")
	}
	if err := keepHTTPS(req("http://y/p"), []*http.Request{req("https://x/p"), req("https://z/p")}); err == nil {
		t.Error("https -> https -> http redirect was allowed")
	}
	if err := keepHTTPS(req("https://y/p"), []*http.Request{req("https://x/p")}); err != nil {
		t.Errorf("https -> https redirect refused: %v", err)
	}
	if err := keepHTTPS(req("http://y/p"), []*http.Request{req("http://x/p")}); err != nil {
		t.Errorf("http -> http redirect refused: %v", err)
	}
	if err := keepHTTPS(req("https://y/p"), make([]*http.Request, 10)); err == nil {
		t.Error("an 11th redirect was allowed")
	}
}

func TestGitProtocols(t *testing.T) {
	for addr, want := range map[string]string{
		"https://host/r.git":   "https:ssh:file",
		"git@host:r.git":       "https:ssh:file",
		"ssh://host/r.git":     "https:ssh:file",
		"ext::sh -c x% y .git": "https:ssh:file",
		"http://host/r.git":    "https:ssh:file:http:git",
		"git://host/r":         "https:ssh:file:http:git",
	} {
		if got := gitProtocols(addr); got != want {
			t.Errorf("gitProtocols(%q) = %q, want %q", addr, got, want)
		}
	}
}

// isolatedGit gives git a fresh home holding only config.
func isolatedGit(t *testing.T, config string) {
	t.Helper()
	if !proc.Git.Found() {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// ext:: stays refused even when the user's git config allows it.
func TestCloneRefusesExt(t *testing.T) {
	isolatedGit(t, "[protocol \"ext\"]\n\tallow = always\n")
	dir := t.TempDir()
	addr := func(marker string) string { return "ext::sh -c touch% " + filepath.Join(dir, marker) }

	// Control: plain git runs it.
	proc.Git.Command("clone", "--", addr("control"), filepath.Join(dir, "c")).Run()
	if _, err := os.Stat(filepath.Join(dir, "control")); err != nil {
		t.Skip("git does not run ext:: here; nothing to prove")
	}

	if err := (&GitCloner{}).CloneRepo(addr("ran"), filepath.Join(dir, "d")); err == nil {
		t.Error("clone succeeded")
	}
	if _, err := os.Stat(filepath.Join(dir, "ran")); err == nil {
		t.Error("ext:: ran a command")
	}
}

// An https clone is not redirected to plain http.
func TestCloneStaysOnHTTPS(t *testing.T) {
	isolatedGit(t, "")
	t.Setenv("GIT_SSL_NO_VERIFY", "1")
	var hits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.NotFound(w, r)
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+r.URL.RequestURI(), http.StatusFound)
	}))
	defer secure.Close()
	dir := t.TempDir()

	// Control: plain git follows it.
	proc.Git.Command("clone", "--", secure.URL+"/r.git", filepath.Join(dir, "c")).Run()
	if hits.Load() == 0 {
		t.Skip("git does not follow the redirect here; nothing to prove")
	}
	hits.Store(0)

	if err := (&GitCloner{}).CloneRepo(secure.URL+"/r.git", filepath.Join(dir, "d")); err == nil {
		t.Error("clone succeeded")
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("plain http got %d request(s)", n)
	}
}

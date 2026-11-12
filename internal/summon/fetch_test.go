package summon

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

package cli

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
	big := strings.Repeat("a", profileMaxSize+1)
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

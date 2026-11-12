package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const showDeck = `x = 2
y = 2
shell = "bash"

[[commands]]
name = "Hello"
command = "echo hi"

[[commands]]
name = "Update"
command = { linux_arch = "sudo pacman -Syu", macos = "brew upgrade" }

[[commands]]
name = "Tools"
[[commands.items]]
name = "Wipe"
command = "rm -rf ~/tmp\u001b[2K\rls"
`

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	f()
	os.Stdout = orig
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

// --show lists every command as written, each platform included, with the
// characters that could hide text escaped; then the deck is confirmed.
func TestSummonShowListsCommandsBeforeConfirming(t *testing.T) {
	ui := &MockUI{}
	var listedBeforeAsking bool
	var out string
	s := &Summoner{
		ConfigDir: t.TempDir(),
		Downloader: &MockDownloader{DownloadFunc: func(url, dst string) error {
			return os.WriteFile(dst, []byte(showDeck), 0o644)
		}},
		Cloner: &MockCloner{},
		UI:     ui,
		Show:   true,
	}
	var prompts []string
	ui.ConfirmFunc = func(p string) bool {
		prompts = append(prompts, p)
		if strings.HasPrefix(p, "Summon ") {
			// Confirming the deck itself: the listing must already be out.
			listedBeforeAsking = true
			return false
		}
		return true
	}
	var err error
	out = captureStdout(t, func() { err = s.Summon("https://x/show.profile.toml") })
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err = %v, want cancelled at the second prompt", err)
	}
	if !listedBeforeAsking || len(prompts) != 2 {
		t.Fatalf("prompts = %q, want a download prompt and a deck prompt", prompts)
	}
	for _, want := range []string{
		"Shell: bash",
		"Commands (4), as written:",
		"    Hello\n      echo hi\n",
		"    Update [linux_arch]\n      sudo pacman -Syu\n",
		"    Update [macos]\n      brew upgrade\n",
		`    Tools › Wipe` + "\n" + `      rm -rf ~/tmp\x1b[2K\rls`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(s.ConfigDir, "show.profile.toml")); err == nil {
		t.Error("the deck was installed although it was declined")
	}
}

// Without --show nothing changes: one prompt, no listing.
func TestSummonWithoutShowAsksOnce(t *testing.T) {
	ui := &yesToAll{}
	s := &Summoner{
		ConfigDir: t.TempDir(),
		Downloader: &MockDownloader{DownloadFunc: func(url, dst string) error {
			return os.WriteFile(dst, []byte(showDeck), 0o644)
		}},
		Cloner: &MockCloner{},
		UI:     ui,
	}
	var err error
	out := captureStdout(t, func() { err = s.Summon("https://x/show.profile.toml") })
	if err != nil {
		t.Fatal(err)
	}
	if len(ui.prompts) != 1 || strings.Contains(out, "as written") {
		t.Errorf("prompts = %q, listing shown = %v", ui.prompts, strings.Contains(out, "as written"))
	}
}

func TestTakeFlag(t *testing.T) {
	rest, found := takeFlag([]string{"--show", "https://x/p", "--sha256", "a"}, "--show")
	if !found || strings.Join(rest, " ") != "https://x/p --sha256 a" {
		t.Errorf("got %q, %v", rest, found)
	}
	if _, found := takeFlag([]string{"https://x/p"}, "--show"); found {
		t.Error("found a flag that is not there")
	}
}

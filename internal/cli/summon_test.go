package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockUI implements UIInterface for testing
type MockUI struct {
	ConfirmFunc func(prompt string) bool
}

func (m *MockUI) Confirm(prompt string) bool {
	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(prompt)
	}
	return true // Default yes
}

// MockDownloader implements FileDownloader for testing
type MockDownloader struct {
	DownloadFunc func(url, destPath string) error
}

func (m *MockDownloader) DownloadFile(url, destPath string) error {
	if m.DownloadFunc != nil {
		return m.DownloadFunc(url, destPath)
	}
	// Simulate creating a valid profile file with required fields
	content := `
x=3
y=3
[[commands]]
name="Test"
command="echo test"
	`
	return os.WriteFile(destPath, []byte(content), 0644)
}

// MockCloner implements RepoCloner for testing
type MockCloner struct{}

func (m *MockCloner) CloneRepo(url, destDir string) error {
	return nil
}
func (m *MockCloner) CheckGitAvailable() error {
	return nil
}

// An equipped deck is replaced where it is; no second copy lands in the inventory.
func TestSummonHTTPReplacesEquippedDeck(t *testing.T) {
	tmpDir := t.TempDir()
	equippedPath := filepath.Join(tmpDir, "git.profile.toml")
	if err := os.WriteFile(equippedPath, []byte("theme='y'"), 0644); err != nil {
		t.Fatal(err)
	}

	summoner := &Summoner{
		ConfigDir:  tmpDir,
		Downloader: &MockDownloader{}, // writes default content
		Cloner:     &MockCloner{},
		UI:         &MockUI{},
	}
	if err := summoner.Summon("https://example.com/git.profile.toml"); err != nil {
		t.Fatalf("Summon failed: %v", err)
	}

	if content, _ := os.ReadFile(equippedPath); string(content) == "theme='y'" {
		t.Error("the equipped deck was not replaced")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "inventory", "git.profile.toml")); err == nil {
		t.Error("a second copy landed in the inventory")
	}
}

func TestSummon_InventoryOverwrite_Confirm(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drako_summon_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Inventory dir
	invDir := filepath.Join(tmpDir, "inventory")
	os.MkdirAll(invDir, 0755)

	// Case: Profile exists in INVENTORY
	existingPath := filepath.Join(invDir, "new.profile.toml")
	os.WriteFile(existingPath, []byte("old_content=true"), 0644)

	summoner := &Summoner{
		ConfigDir:  tmpDir,
		Downloader: &MockDownloader{}, // writes default content
		Cloner:     &MockCloner{},
		UI: &MockUI{
			ConfirmFunc: func(p string) bool { return true }, // Confirm overwrite
		},
	}

	// Summon
	err = summoner.Summon("https://example.com/new.profile.toml")
	if err != nil {
		t.Fatalf("Summon failed: %v", err)
	}

	// Verify content changed (MockDownloader writes defaults)
	content, _ := os.ReadFile(existingPath)
	if string(content) == "old_content=true" {
		t.Error("File was not overwritten")
	}
}

func TestSummon_InventoryOverwrite_Deny(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drako_summon_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	invDir := filepath.Join(tmpDir, "inventory")
	os.MkdirAll(invDir, 0755)

	existingPath := filepath.Join(invDir, "deny.profile.toml")
	os.WriteFile(existingPath, []byte("original"), 0644)

	summoner := &Summoner{
		ConfigDir:  tmpDir,
		Downloader: &MockDownloader{},
		Cloner:     &MockCloner{},
		UI: &MockUI{
			ConfirmFunc: func(p string) bool { return false }, // Deny overwrite
		},
	}

	err = summoner.Summon("https://example.com/deny.profile.toml")
	if err == nil {
		t.Error("Expected error 'operation cancelled', got nil")
	}

	// Verify content NOT changed
	content, _ := os.ReadFile(existingPath)
	if string(content) != "original" {
		t.Error("File WAS overwritten despite denial")
	}
}

func TestSummonGitReplacesEquippedDeck(t *testing.T) {
	s, _ := gitSummoner(t, func(dir string) error {
		return os.WriteFile(filepath.Join(dir, "repo.profile.toml"), []byte(validDeck+"# v2\n"), 0o644)
	})
	equippedPath := filepath.Join(s.ConfigDir, "repo.profile.toml")
	if err := os.WriteFile(equippedPath, []byte(validDeck), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := s.Summon("git@github.com:user/repo.git"); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if content, _ := os.ReadFile(equippedPath); !strings.Contains(string(content), "# v2") {
		t.Error("the equipped deck was not replaced")
	}
	if _, err := os.Stat(filepath.Join(s.ConfigDir, "inventory", "repo.profile.toml")); err == nil {
		t.Error("a second copy landed in the inventory")
	}
}

const validDeck = "x = 1\ny = 1\n[[commands]]\nname = \"T\"\ncommand = \"true\"\ncol = \"a\"\nrow = 0\n"

// repoCloner fakes a clone by running build against the checkout directory.
type repoCloner struct{ build func(dir string) error }

func (c repoCloner) CloneRepo(url, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return c.build(destDir)
}
func (repoCloner) CheckGitAvailable() error { return nil }

// yesToAll answers every prompt with yes and records it.
type yesToAll struct{ prompts []string }

func (u *yesToAll) Confirm(prompt string) bool {
	u.prompts = append(u.prompts, prompt)
	return true
}

func gitSummoner(t *testing.T, build func(dir string) error) (*Summoner, *yesToAll) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // anything written here went to the wrong config dir
	ui := &yesToAll{}
	return &Summoner{ConfigDir: t.TempDir(), Downloader: &MockDownloader{}, Cloner: repoCloner{build}, UI: ui}, ui
}

func TestSummonGitSkipsSymlinkedProfiles(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.profile.toml")
	if err := os.WriteFile(outside, []byte(validDeck), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := gitSummoner(t, func(dir string) error {
		if err := os.WriteFile(filepath.Join(dir, "real.profile.toml"), []byte(validDeck), 0o644); err != nil {
			return err
		}
		return os.Symlink(outside, filepath.Join(dir, "evil.profile.toml"))
	})
	if err := s.Summon("git@host:repo.git"); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if got := placed(s, "real.profile.toml"); got != "equipped" {
		t.Errorf("regular profile: placed %q, want equipped", got)
	}
	if got := placed(s, "evil.profile.toml"); got != "" {
		t.Errorf("a symlinked profile was followed out of the repo: placed %q", got)
	}
}

func TestSummonGitAssetsSkipSymlinks(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(secret, []byte("PRIVATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, _ := gitSummoner(t, func(dir string) error {
		deck := "assets = [\"data\"]\n" + validDeck
		if err := os.WriteFile(filepath.Join(dir, "deck.profile.toml"), []byte(deck), 0o644); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "data", "ok.txt"), []byte("ok"), 0o644); err != nil {
			return err
		}
		return os.Symlink(secret, filepath.Join(dir, "data", "leak"))
	})
	if err := s.Summon("git@host:repo.git"); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	assets := filepath.Join(s.ConfigDir, "assets", "deck", "data")
	if _, err := os.Stat(filepath.Join(assets, "ok.txt")); err != nil {
		t.Errorf("regular asset not copied into the summoner's config dir: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(assets, "leak")); err == nil {
		t.Error("a symlinked asset was followed out of the repo")
	}
}

func TestSummonGitUsesSummonerConfigDirAndUI(t *testing.T) {
	s, ui := gitSummoner(t, func(dir string) error {
		if err := os.WriteFile(filepath.Join(dir, "deck.profile.toml"), []byte(validDeck), 0o644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "team.spec.toml"), []byte("profiles = [\"deck\"]\n"), 0o644)
	})
	if err := s.Summon("git@host:repo.git"); err != nil {
		t.Fatalf("Summon: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.ConfigDir, "specs", "team.spec.toml")); err != nil {
		t.Errorf("spec not in the summoner's config dir: %v", err)
	}
	if got := strings.Join(ui.prompts, " | "); !strings.Contains(got, "deck.profile.toml") || !strings.Contains(got, "team.spec.toml") {
		t.Errorf("per-file prompts bypassed the summoner's UI; it saw: %s", got)
	}
}

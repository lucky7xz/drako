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

func TestSummon_EquippedCollision(t *testing.T) {
	// Setup temp config dir
	tmpDir, err := os.MkdirTemp("", "drako_summon_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Case: Profile exists in ROOT (Equipped)
	// We want to summon "git.profile.toml".
	// Create "git.profile.toml" in ROOT.
	equippedPath := filepath.Join(tmpDir, "git.profile.toml")
	if err := os.WriteFile(equippedPath, []byte("theme='y'"), 0644); err != nil {
		t.Fatal(err)
	}

	summoner := &Summoner{
		ConfigDir:  tmpDir,
		Downloader: &MockDownloader{},
		Cloner:     &MockCloner{},
		UI:         &MockUI{},
	}

	// Attempt to summon same name
	err = summoner.Summon("https://example.com/git.profile.toml")

	if err == nil {
		t.Error("Expected error due to Equipped Collision, got nil")
	} else {
		// Check error message contains specific safety violation text
		expected := "safety violation: 'git.profile.toml' is currently EQUIPPED"
		if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
			t.Errorf("Expected collision error, got: %v", err)
		}
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

// Upgraded MockCloner
type MockClonerFunc struct {
	CloneFunc func(url, destDir string) error
}

func (m *MockClonerFunc) CheckGitAvailable() error { return nil }
func (m *MockClonerFunc) CloneRepo(url, destDir string) error {
	if m.CloneFunc != nil {
		return m.CloneFunc(url, destDir)
	}
	return nil
}

func TestSummon_Git_EquippedCollision_Real(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drako_summon_git_test_real")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	equippedPath := filepath.Join(tmpDir, "repo.profile.toml")
	os.WriteFile(equippedPath, []byte("x=1\ny=1\n[[commands]]\nname='E'\ncommand='e'"), 0644)

	summoner := &Summoner{
		ConfigDir:  tmpDir,
		Downloader: &MockDownloader{},
		Cloner: &MockClonerFunc{
			CloneFunc: func(url, destDir string) error {
				// Simulate git clone by writing the file into destDir
				// Must create dir first since real git clone would do it
				if err := os.MkdirAll(destDir, 0755); err != nil {
					return err
				}
				f := filepath.Join(destDir, "repo.profile.toml")
				return os.WriteFile(f, []byte("x=2\ny=2\n[[commands]]\nname='N'\ncommand='n'"), 0644)
			},
		},
		UI: &MockUI{},
	}

	// Use a git URL to trigger summonFromGit
	err = summoner.Summon("git@github.com:user/repo.git")

	// Since we strictly skip collisions in the loop, if the *only* file was a collision,
	// Summon should return an error saying "no valid items found" or similar.
	// It should NOT succeed silently.
	if err == nil {
		t.Error("FAIL: Git summon succeeded despite Equipped Collision")
	} else {
		// Expect "no valid items found" or similar error generated by summonFromGit when everything is skipped
		// Also implicitly confirms the safety check worked.
		expected := "no valid items found"
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Expected 'no valid items' error (indicating skip), got: %v", err)
		}
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
	inv := filepath.Join(s.ConfigDir, "inventory")
	if _, err := os.Stat(filepath.Join(inv, "real.profile.toml")); err != nil {
		t.Errorf("regular profile not summoned: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(inv, "evil.profile.toml")); err == nil {
		t.Error("a symlinked profile was followed out of the repo")
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

func TestSummonPathsRejectBackslashTraversal(t *testing.T) {
	if _, ok := cleanAssetRel(`..\..\x.profile.toml`); ok {
		t.Error(`cleanAssetRel accepted ..\..\x.profile.toml`)
	}
	if err := validateFilename(`..\..\work.profile.toml`); err == nil {
		t.Error(`validateFilename accepted ..\..\work.profile.toml`)
	}
	if !isGitURL("ssh://git@host/user/repo") {
		t.Error("ssh:// URL not treated as git")
	}
}

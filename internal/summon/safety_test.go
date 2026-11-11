package summon

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummonPathsRejectBackslashTraversal(t *testing.T) {
	if _, ok := cleanAssetRel(`..\..\x.profile.toml`); ok {
		t.Error(`cleanAssetRel accepted ..\..\x.profile.toml`)
	}
	if err := ValidateFilename(`..\..\work.profile.toml`); err == nil {
		t.Error(`ValidateFilename accepted ..\..\work.profile.toml`)
	}
	if !IsGitURL("ssh://git@host/user/repo") {
		t.Error("ssh:// URL not treated as git")
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

	if err := VerifyFileSHA256(path, good); err != nil {
		t.Errorf("correct hash rejected: %v", err)
	}
	if err := VerifyFileSHA256(path, strings.ToUpper(good)); err != nil {
		t.Errorf("uppercase hash must match (case-insensitive): %v", err)
	}
	err := VerifyFileSHA256(path, strings.Repeat("0", 64))
	if err == nil || !strings.Contains(err.Error(), "MISMATCH") {
		t.Errorf("wrong hash: err = %v, want MISMATCH", err)
	}
}

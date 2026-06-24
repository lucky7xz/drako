package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MoveToTrash moves file from the equipped (config) dir into the trash dir,
// timestamping the destination so it never overwrites an existing trashed file.
// file is validated to resolve inside configDir (no path traversal).
func MoveToTrash(configDir, file string) error {
	src := filepath.Clean(filepath.Join(configDir, file))

	absConfig, err := filepath.Abs(configDir)
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}
	if !strings.HasPrefix(absSrc, absConfig+string(os.PathSeparator)) && absSrc != absConfig {
		return fmt.Errorf("path traversal detected: %s", file)
	}
	if _, err := os.Stat(src); err != nil {
		return err
	}

	trashDir := Trash.Dir(configDir)
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return err
	}
	stamp := time.Now().Format("20060102-150405")
	dst := filepath.Join(trashDir, fmt.Sprintf("%s.%s", filepath.Base(src), stamp))
	return os.Rename(src, dst)
}

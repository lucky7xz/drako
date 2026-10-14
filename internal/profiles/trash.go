package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MoveToTrash moves file (validated to stay inside configDir) into the trash dir
// under a timestamped name, numbered when taken, so nothing trashed is replaced.
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
	base := filepath.Join(trashDir, filepath.Base(src)+"."+time.Now().Format("20060102-150405"))
	dst := base
	for n := 2; ; n++ {
		_, err := os.Lstat(dst)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return err
		}
		dst = fmt.Sprintf("%s-%d", base, n)
	}
	return os.Rename(src, dst)
}

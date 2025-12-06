package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucky7xz/drako/internal/native"
)

// OpenPath expands the path and opens it using native.Open.
func OpenPath(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("empty path")
	}

	// Expand home directory if present
	if strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			target = filepath.Join(home, target[2:])
		}
	}

	return native.Open(target)
}

// HandleOpenCommand executes the "open" operation from an internal command string.
// It parses the command string (expected "drako open <path>"), resolves the path,
// and invokes native.Open.
func HandleOpenCommand(command string) {
	// Format: "drako open <path>"
	parts := strings.Fields(command)
	if len(parts) < 3 {
		fmt.Println("Usage: drako open <path>")
		return
	}

	// Reconstruct the path in case it had spaces
	prefix := "drako open "
	if !strings.HasPrefix(command, prefix) {
		return
	}
	target := strings.TrimSpace(strings.TrimPrefix(command, prefix))

	if err := OpenPath(target); err != nil {
		fmt.Printf("Error opening '%s': %v\n", target, err)
	}
}

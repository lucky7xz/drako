package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OpenPath expands the path and opens it using the OS default application.
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
	} else if strings.HasPrefix(target, "~\\") && runtime.GOOS == "windows" {
		home, err := os.UserHomeDir()
		if err == nil {
			target = filepath.Join(home, target[2:])
		}
	}

	return openNative(target)
}

// openNative opens the specified URL, file, or directory using the OS default application.
func openNative(target string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "windows":
		// Windows 'start' via cmd /c is reliable
		cmd = exec.Command("cmd", "/c", "start", "", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open '%s': %w", target, err)
	}
	return nil
}

// HandleOpenCommand executes the "open" operation from an internal command string.
func HandleOpenCommand(command string) {
	// Format: "drako open <path>"
	// We handle spaces by taking everything after "drako open "
	prefix := "drako open "
	if !strings.HasPrefix(command, prefix) {
		return
	}
	target := strings.TrimSpace(strings.TrimPrefix(command, prefix))

	if err := OpenPath(target); err != nil {
		fmt.Printf("Error opening '%s': %v\n", target, err)
	}
}

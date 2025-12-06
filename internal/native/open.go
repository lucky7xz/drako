package native

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open opens the specified URL, file, or directory using the OS default application.
func Open(target string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "windows":
		// Windows 'start' is a shell built-in, so we need to run it via cmd /c
		// We use rundll32 for URLs sometimes, but 'start' is more general for files/dirs too.
		// However, 'start' has weird quoting rules.
		// A reliable way for URLs and files on Windows from Go:
		cmd = exec.Command("cmd", "/c", "start", "", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		// Fallback for other *nix
		cmd = exec.Command("xdg-open", target)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open '%s': %w", target, err)
	}
	
	// We don't wait for the command to finish because it usually launches a GUI app
	// that might stay open (like a browser).
	// But checking for immediate start errors is good.
	
	return nil
}

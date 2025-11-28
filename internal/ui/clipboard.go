package ui

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard attempts to copy text to clipboard using various methods
// based on the current environment, in order of preference.
func CopyToClipboard(s string) {
	if strings.TrimSpace(s) == "" {
		return
	}

	// First try platform-specific clipboard tools
	if tryPlatformClipboard(s) {
		return
	}

	// Then try OSC52 (direct terminal clipboard)
	// This now handles tmux/screen wrapping internally
	if tryOSC52(s) {
		return
	}
}

func tryPlatformClipboard(s string) bool {
	switch runtime.GOOS {
	case "linux":
		cmd, args := getLinuxClipboardCommand()
		if cmd != "" {
			return tryCommand(s, cmd, args...)
		}
	case "darwin":
		return tryCommand(s, "pbcopy")
	case "windows":
		// Try PowerShell first, then clip.exe
		if tryPowerShellClipboard(s) {
			return true
		}
		return tryCommand(s, "clip.exe")
	}
	return false
}

// getLinuxClipboardCommand returns the appropriate clipboard command for Linux systems
func getLinuxClipboardCommand() (string, []string) {
	// Check for Wayland first
	if isWayland() {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy", []string{}
		}
	}

	// Check for X11 clipboard utilities
	if _, err := exec.LookPath("xclip"); err == nil {
		return "xclip", []string{"-selection", "clipboard"}
	}

	if _, err := exec.LookPath("xsel"); err == nil {
		return "xsel", []string{"--clipboard", "--input"}
	}

	// No clipboard utility found
	return "", []string{}
}

// isWayland checks if we are running under Wayland
func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || strings.Contains(os.Getenv("XDG_SESSION_TYPE"), "wayland")
}

// tryOSC52 attempts to copy text using the OSC52 escape sequence.
// It automatically handles wrapping for tmux and screen if detected.
func tryOSC52(s string) bool {
	// Encode the string in base64
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	var seq string

	// Detect multiplexers and wrap accordingly
	if os.Getenv("TMUX") != "" {
		// Tmux requires special wrapping: \x1bPtmux;\x1b ... \x1b\\
		seq = fmt.Sprintf("\x1bPtmux;\x1b\x1b]52;c;%s\x07\x1b\\", enc)
	} else if os.Getenv("STY") != "" {
		// Screen requires special wrapping: \x1bP ... \x1b\\
		seq = fmt.Sprintf("\x1bP\x1b]52;c;%s\x07\x1b\\", enc)
	} else {
		// Standard OSC52 sequence
		seq = fmt.Sprintf("\x1b]52;c;%s\x07", enc)
	}

	_, err := os.Stderr.WriteString(seq)
	if err != nil {
		log.Printf("OSC52 copy failed: %v", err)
		return false
	}

	log.Printf("Attempted to copy to clipboard using OSC52")
	return true
}

// tryCommand attempts to copy text using an external command
func tryCommand(s string, name string, args ...string) bool {
	// Check if command exists
	if _, err := exec.LookPath(name); err != nil {
		return false
	}

	// Execute command with text as input
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(s)

	// Run command and return success status
	err := cmd.Run()
	if err != nil {
		log.Printf("Clipboard command failed: %s %v, error: %v", name, args, err)
		return false
	}

	log.Printf("Successfully copied to clipboard using: %s %v", name, args)
	return true
}

// tryPowerShellClipboard attempts to copy text using PowerShell on Windows
func tryPowerShellClipboard(s string) bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// Try PowerShell with Set-Clipboard
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-Command", fmt.Sprintf("Set-Clipboard -Value \"%s\"", s))
	err := cmd.Run()
	return err == nil
}

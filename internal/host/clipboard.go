package host

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/lucky7xz/drako/internal/proc"
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
		if tool, args, ok := linuxClipboardTool(); ok {
			return tryTool(s, tool, args...)
		}
	case "darwin":
		return tryTool(s, proc.Pbcopy)
	case "windows":
		// Try PowerShell first, then clip.exe
		if tryPowerShellClipboard(s) {
			return true
		}
		return tryTool(s, proc.ClipExe)
	}
	return false
}

// linuxClipboardTool picks the clipboard program for this Linux session, with
// the arguments that make it read the text from its input.
func linuxClipboardTool() (proc.Tool, []string, bool) {
	// Check for Wayland first
	if isWayland() && proc.WlCopy.Found() {
		return proc.WlCopy, nil, true
	}

	// Check for X11 clipboard utilities
	if proc.Xclip.Found() {
		return proc.Xclip, []string{"-selection", "clipboard"}, true
	}
	if proc.Xsel.Found() {
		return proc.Xsel, []string{"--clipboard", "--input"}, true
	}

	// No clipboard utility found
	return proc.Tool{}, nil, false
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

// tryTool attempts to copy text by feeding it to a clipboard program.
func tryTool(s string, tool proc.Tool, args ...string) bool {
	if !tool.Found() {
		return false
	}

	// Execute command with text as input
	cmd := tool.Command(args...)
	cmd.Stdin = strings.NewReader(s)

	// Run command and return success status
	err := cmd.Run()
	if err != nil {
		log.Printf("Clipboard command failed: %s %v, error: %v", tool, args, err)
		return false
	}

	log.Printf("Successfully copied to clipboard using: %s %v", tool, args)
	return true
}

// tryPowerShellClipboard attempts to copy text using PowerShell on Windows
func tryPowerShellClipboard(s string) bool {
	if runtime.GOOS != "windows" {
		return false
	}

	cmd := proc.PowerShell.Command(powerShellClipboardArgs(s)...)
	err := cmd.Run()
	return err == nil
}

// powerShellClipboardArgs passes s as base64 inside a single-quoted literal,
// so PowerShell never parses the copied text ($(...), quotes, backticks).
func powerShellClipboardArgs(s string) []string {
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	script := fmt.Sprintf("Set-Clipboard -Value ([Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s')))", enc)
	return []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", script}
}

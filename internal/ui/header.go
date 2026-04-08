package ui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// HeaderConfig holds configuration for dynamic command-based header
type HeaderConfig struct {
	Enabled  bool          // Enable command execution
	Command  string        // Command to execute (e.g., "date", "figlet", "fortune")
	Args     []string      // Command arguments
	Timeout  time.Duration // Maximum execution time (default: 2s)
	Fallback string        // Fallback text if command fails
}

// ExecuteHeaderCommand runs a command and returns its stdout as header text
func ExecuteHeaderCommand(cmd string, args []string, timeout time.Duration) (string, error) {
	// Set default timeout if not specified
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	// Create command with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	command := exec.CommandContext(ctx, cmd, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	// Execute command
	err := command.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %v (stderr: %s)", err, stderr.String())
	}

	// Return stdout, trimmed
	output := strings.TrimSpace(stdout.String())
	return output, nil
}

// RenderCommandHeader executes command and formats it as centered header
func RenderCommandHeader(cfg HeaderConfig, spinnerView string) string {
	if !cfg.Enabled || cfg.Command == "" {
		// Return default header art if command is not configured
		return renderDefaultHeaderArt(spinnerView)
	}

	// Execute the command
	output, err := ExecuteHeaderCommand(cfg.Command, cfg.Args, cfg.Timeout)

	if err != nil {
		// Use fallback if provided, otherwise use default header
		if cfg.Fallback != "" {
			output = cfg.Fallback
		} else {
			return renderDefaultHeaderArt(spinnerView)
		}
	}

	// Center the output
	var result strings.Builder
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		centered := lipgloss.NewStyle().
			Width(80).
			Align(lipgloss.Center).
			Render(line)
		result.WriteString(centered)
		result.WriteString("\n")
	}

	return result.String()
}

// renderDefaultHeaderArt is the original static ASCII art header
func renderDefaultHeaderArt(spinnerView string) string {
	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)

	logo := logoStyle.Render(`
    ██████╗ ██████╗  █████╗ ██╗  ██╗ ██████╗ 
    ██╔══██╗██╔══██╗██╔══██╗██║ ██╔╝██╔═══██╗
    ██║  ██║██████╔╝███████║█████╔╝ ██║   ██║
    ██║  ██║██╔══██╗██╔══██║██╔═██╗ ██║   ██║
    ██████╔╝██║  ██║██║  ██║██║  ██╗╚██████╔╝
    ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ 
	`)

	spinner := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D9FF")).
		Render(spinnerView)

	return lipgloss.JoinVertical(lipgloss.Center, logo, spinner)
}

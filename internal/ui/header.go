package ui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerCache      string
	headerCacheTime  time.Time
	headerCacheCmd   string
	headerCacheMutex sync.Mutex
	headerCacheTTL   = 500 * time.Millisecond
)

// HeaderConfig holds configuration for dynamic command-based header
type HeaderConfig struct {
	Enabled  bool          // Enable command execution
	Command  string        // Command to execute (e.g., "date", "figlet", "fortune")
	Args     []string      // Command arguments
	Timeout  time.Duration // Maximum execution time (default: 2s)
	Fallback string        // Fallback text if command fails
	MaxLines int           // Maximum lines to display (0 = unlimited, capped at 15)
}

// ExecuteHeaderCommand runs a command and returns its stdout as header text
func ExecuteHeaderCommand(cmd string, args []string, timeout time.Duration) (string, error) {
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	command := exec.CommandContext(ctx, cmd, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %v (stderr: %s)", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	return output, nil
}

// RenderCommandHeader executes command and formats it as centered header with caching
func RenderCommandHeader(cfg HeaderConfig, spinnerView string) string {
	if !cfg.Enabled || cfg.Command == "" {
		return renderDefaultHeaderArt(spinnerView)
	}

	headerCacheMutex.Lock()
	defer headerCacheMutex.Unlock()

	now := time.Now()
	cmdKey := cfg.Command + strings.Join(cfg.Args, "|")

	if cmdKey == headerCacheCmd && now.Sub(headerCacheTime) < headerCacheTTL {
		return formatHeaderOutput(headerCache, cfg, spinnerView)
	}

	output, err := ExecuteHeaderCommand(cfg.Command, cfg.Args, cfg.Timeout)

	if err != nil {
		if cfg.Fallback != "" {
			output = cfg.Fallback
		} else {
			return renderDefaultHeaderArt(spinnerView)
		}
	}

	headerCacheCmd = cmdKey
	headerCache = output
	headerCacheTime = now

	return formatHeaderOutput(output, cfg, spinnerView)
}

func formatHeaderOutput(output string, cfg HeaderConfig, spinnerView string) string {
	var result strings.Builder
	lines := strings.Split(output, "\n")

	maxLines := cfg.MaxLines
	if maxLines == 0 {
		maxLines = 15
	}
	if maxLines > 15 {
		maxLines = 15
	}

	for i, line := range lines {
		if i >= maxLines {
			break
		}
		centered := lipgloss.NewStyle().
			Width(80).
			Align(lipgloss.Center).
			Render(line)
		result.WriteString(centered)
		result.WriteString("\n")
	}

	return result.String()
}

// renderDefaultHeaderArt delegates to renderHeaderArt (respects header_art config)
func renderDefaultHeaderArt(spinnerView string) string {
	return renderHeaderArt(spinnerView)
}

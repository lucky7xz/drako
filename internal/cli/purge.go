package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PurgeOptions defines the scope of the purge operation
type PurgeOptions struct {
	DestroyEverything bool   // Nuke ~/.config/drako entirely
	TargetProfile     string // Delete/Move specific profile (e.g. "git" or "core")
	TargetConfig      bool   // Reset config.toml
}

// PurgeConfig executes the purge operation based on the options.
// It moves files to ~/.config/drako/trash/ instead of deleting them,
// unless DestroyEverything is true.
func PurgeConfig(configDir string, opts PurgeOptions) error {
	if opts.DestroyEverything {
		log.Printf("Starting FULL purge (destroy everything) for: %s", configDir)
		return performFullNuke(configDir)
	}

	// Ensure trash directory exists
	trashDir := filepath.Join(configDir, "trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return fmt.Errorf("failed to create trash directory: %w", err)
	}

	// Case 1: Reset Core Config (config.toml)
	if opts.TargetConfig {
		log.Printf("Purging Core config (config.toml)")
		return moveFileToTrash(configDir, "config.toml", trashDir)
	}

	// Case 2: Target Specific Profile
	if opts.TargetProfile != "" {
		log.Printf("Purging Profile: %s", opts.TargetProfile)
		// Profile could be "git.profile.toml" or just "git"
		// We should try to find the matching file
		// For now, assume simple name matching
		filename := opts.TargetProfile
		if filepath.Ext(filename) != ".toml" {
			filename = filename + ".profile.toml"
		}
		return moveFileToTrash(configDir, filename, trashDir)
	}

	// Strict Safety: If we reach here, no target was specified.
	// We DO NOT default to purging everything anymore.
	return fmt.Errorf("no target specified (use --target, --config, or --destroyeverything)")
}

// performFullNuke implements the "Destroy Everything" logic (Old --all)
func performFullNuke(configDir string) error {
	// Confirm existence
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", configDir)
	}

	fmt.Printf("\n💀 DESTROYING EVERYTHING in %s\n", configDir)
	// The caller (HandlePurgeCommand) should have asked for confirmation.

	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("failed to destroy config directory: %w", err)
	}
	return nil
}

// moveFileToTrash moves a single file from configDir to trashDir with a timestamp
func moveFileToTrash(configDir, filename, trashDir string) error {
	// SANITIZATION: Prevent path traversal
	// 1. Clean the path to resolve .. and .
	src := filepath.Clean(filepath.Join(configDir, filename))

	// 2. Ensure it starts with configDir
	// We verify that the resolved path is still inside the configDir
	// Note: We use Abs to be safe against relative configDir setups
	absConfig, err := filepath.Abs(configDir)
	if err != nil {
		return fmt.Errorf("failed to resolve config dir: %w", err)
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}

	if !strings.HasPrefix(absSrc, absConfig+string(os.PathSeparator)) && absSrc != absConfig {
		// allow exact match? No, we don't want to trash the dir itself here.
		// Strict check: must be a child.
		return fmt.Errorf("security violation: path traversal detected (%s)", filename)
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filename)
	}
	return moveToTrash(src, trashDir)
}

func moveToTrash(srcPath, trashDir string) error {
	filename := filepath.Base(srcPath)
	timestamp := time.Now().Format("20060102-150405")
	dstName := fmt.Sprintf("%s.%s", filename, timestamp)
	dstPath := filepath.Join(trashDir, dstName)

	if err := os.Rename(srcPath, dstPath); err != nil {
		return err
	}
	fmt.Printf("  ✓ Moved %s to trash\n", filename)
	return nil
}

// countFilesInDir recursively counts files in a directory
func countFilesInDir(dir string) int {
	count := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// formatSize converts bytes to human-readable format
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

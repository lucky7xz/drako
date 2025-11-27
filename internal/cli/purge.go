package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// PurgeConfig deletes everything in ~/.config/drako/ except config.toml (unless nukeAll is true)
// Shows a preview and requires confirmation
func PurgeConfig(configDir string, nukeAll bool) error {
	if nukeAll {
		log.Printf("Starting FULL purge (--all) for: %s", configDir)
	} else {
		log.Printf("Starting purge (config.toml preserved) for: %s", configDir)
	}

	// Check if config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", configDir)
	}

	// Collect all items in config dir
	items, err := collectPurgeItems(configDir, nukeAll)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("\n✓ Nothing to purge - config directory is already clean")
		log.Printf("Purge cancelled: nothing to delete")
		return nil
	}

	// Show preview
	if nukeAll {
		fmt.Printf("\n💀 FULL PURGE - The entire directory will be DELETED:\n")
		fmt.Printf("   %s\n\n", configDir)
	} else {
		fmt.Printf("\n🗑️  The following items will be DELETED from %s:\n\n", configDir)
	}

	for _, item := range items {
		info, err := os.Stat(item)
		if err != nil {
			fmt.Printf("  • %s (error reading)\n", filepath.Base(item))
			continue
		}

		if info.IsDir() {
			// Count files in directory
			count := countFilesInDir(item)
			fmt.Printf("  📁 %s/ (%d items)\n", filepath.Base(item), count)
		} else {
			// Show file size
			size := info.Size()
			sizeStr := formatSize(size)
			fmt.Printf("  📄 %s (%s)\n", filepath.Base(item), sizeStr)
		}
	}

	if nukeAll {
		fmt.Printf("\n💀 EVERYTHING will be deleted (including config.toml)\n")
	} else {
		fmt.Printf("\n✓ config.toml will be PRESERVED\n")
	}
	fmt.Printf("\nTotal: %d items will be deleted\n\n", len(items))

	// Require confirmation
	confirmMsg := "⚠️  This action cannot be undone. Proceed with purge?"
	if nukeAll {
		confirmMsg = "💀 This will DELETE EVERYTHING. Are you absolutely sure?"
	}

	if !ConfirmAction(confirmMsg) {
		log.Printf("Purge cancelled by user")
		return fmt.Errorf("operation cancelled by user")
	}

	// Execute deletion
	deleted := 0
	failed := 0
	for _, item := range items {
		if err := os.RemoveAll(item); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", filepath.Base(item), err)
			log.Printf("Failed to delete %s: %v", item, err)
			failed++
		} else {
			deleted++
		}
	}

	log.Printf("Purge completed: %d deleted, %d failed", deleted, failed)

	if failed > 0 {
		return fmt.Errorf("purge completed with %d failures", failed)
	}

	return nil
}

// collectPurgeItems scans configDir and returns all items (optionally including config.toml if nukeAll is true)
func collectPurgeItems(configDir string, nukeAll bool) ([]string, error) {
	var items []string

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip config.toml unless nukeAll is true
		if name == "config.toml" && !nukeAll {
			continue
		}

		fullPath := filepath.Join(configDir, name)
		items = append(items, fullPath)
	}

	// Sort for consistent display
	sort.Strings(items)

	return items, nil
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

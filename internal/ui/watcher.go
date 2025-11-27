package ui

import (
	"log"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// ConfigChangedMsg signals that a config file has changed
type ConfigChangedMsg struct {
	Path string
}

// startConfigWatcher watches the config directory for .toml file changes
func startConfigWatcher(configDir string) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("Failed to create file watcher: %v", err)
			return nil
		}

		// Watch the main config directory (not recursive)
		if err := watcher.Add(configDir); err != nil {
			log.Printf("Failed to watch config directory: %v", err)
			watcher.Close()
			return nil
		}

		log.Printf("File watcher started for: %s", configDir)

		// Listen for events in a goroutine
		go func() {
			defer watcher.Close()

			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}

					// Only care about Write and Create events for .toml files
					if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
						continue
					}

					// Get the filename
					filename := filepath.Base(event.Name)

					// Ignore files that aren't .toml
					if !strings.HasSuffix(filename, ".toml") {
						continue
					}

					// Ignore files in subdirectories (inventory, etc)
					if filepath.Dir(event.Name) != configDir {
						continue
					}

					log.Printf("Config file changed: %s", filename)

					// Signal the app to reload (this will be caught by bubbletea)
					// Note: We can't send tea.Msg directly from a goroutine,
					// so we'll need to handle this differently

				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Printf("File watcher error: %v", err)
				}
			}
		}()

		return nil
	}
}

// WatchConfigCmd returns a command that continuously watches for config changes
func WatchConfigCmd(configDir string) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("Failed to create file watcher: %v", err)
			return nil
		}

		if err := watcher.Add(configDir); err != nil {
			log.Printf("Failed to watch config directory: %v", err)
			watcher.Close()
			return nil
		}

		log.Printf("Watching for config changes in: %s", configDir)

		// Block and wait for the next relevant event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					watcher.Close()
					return nil
				}

				// Only care about Write and Create events
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				filename := filepath.Base(event.Name)

				// Must be a .toml file
				if !strings.HasSuffix(filename, ".toml") {
					continue
				}

				// Must be in the root config dir (not subdirs)
				if filepath.Dir(event.Name) != configDir {
					continue
				}

				log.Printf("Detected change in: %s", filename)
				watcher.Close()
				return ConfigChangedMsg{Path: event.Name}

			case err, ok := <-watcher.Errors:
				if !ok {
					watcher.Close()
					return nil
				}
				log.Printf("File watcher error: %v", err)
			}
		}
	}
}

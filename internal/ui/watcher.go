package ui

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// settleDelay lets a burst of writes (an editor's truncate, then write) finish
// before reloading, so the reload never reads a half-written profile.
const settleDelay = 150 * time.Millisecond

// ConfigChangedMsg signals that a config file has changed
type ConfigChangedMsg struct {
	Path string
}

// WatchConfig reports .toml changes in configDir to notify until stop is called.
// One watcher serves every lap, so none leak and no change lands in a gap.
func WatchConfig(configDir string, notify func(ConfigChangedMsg)) (stop func(), err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(configDir); err != nil {
		watcher.Close()
		return nil, err
	}
	log.Printf("Watching for config changes in: %s", configDir)

	go func() {
		var settled <-chan time.Time
		var last string
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if isConfigChange(event, configDir) {
					last = event.Name
					settled = time.After(settleDelay) // restart the quiet period
				}
			case <-settled:
				settled = nil
				log.Printf("Detected change in: %s", filepath.Base(last))
				notify(ConfigChangedMsg{Path: last})
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("File watcher error: %v", err)
			}
		}
	}()
	return func() { watcher.Close() }, nil
}

// isConfigChange: a .toml directly in configDir changed. Remove/Rename catch a
// profile trashed out from under an active session (MoveToTrash renames it).
func isConfigChange(event fsnotify.Event, configDir string) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}
	return strings.HasSuffix(event.Name, ".toml") && filepath.Dir(event.Name) == configDir
}

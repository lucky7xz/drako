package core

import (
	"path/filepath"
	"strings"
)

// CommandEnv builds the child environment for one command run: the whitelist
// sanitizes the inherited environment, then drako's own DRAKO_PROFILE is set
// to the active profile — always, even under a whitelist that omits it, so
// scripts can rely on it. With no active profile an externally set value
// passes through untouched.
func CommandEnv(base []string, whitelist []string, activeProfile string) []string {
	env := PrepareEnv(base, whitelist)
	if activeProfile == "" {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, "DRAKO_PROFILE=") {
			out = append(out, e)
		}
	}
	return append(out, "DRAKO_PROFILE="+activeProfile)
}

// BatchEnv builds the environment for one batch cell, which carries its own
// env in a script rather than inheriting it from a multiplexer server. A
// whitelist isolates the cell in exactly that environment; otherwise the cell
// inherits the caller's and only learns the active profile.
func BatchEnv(base []string, whitelist []string, activeProfile string) (env []string, isolate bool) {
	if len(whitelist) > 0 {
		return CommandEnv(base, whitelist, activeProfile), true
	}
	if activeProfile == "" {
		return nil, false
	}
	return []string{"DRAKO_PROFILE=" + activeProfile}, false
}

// PrepareEnv returns the environment variables to use for command execution.
// If whitelist is empty, it returns the original environment (pass-through).
// If whitelist is set, it returns only the variables that match the whitelist.
func PrepareEnv(env []string, whitelist []string) []string {
	if len(whitelist) == 0 {
		return env
	}

	var filtered []string
	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := pair[0]

		// Check if key matches any pattern in whitelist
		matched := false
		for _, pattern := range whitelist {
			// Simple exact match or basic wildcard handling could go here.
			// For now, let's do exact match (case-sensitive or insensitive? usually env vars are case-sensitive on Linux)
			// but let's allow simple globs if needed later. For now, exact match + simple suffix?
			// Let's stick to exact match or shell-glob style Match.

			if match, _ := filepath.Match(pattern, key); match {
				matched = true
				break
			}
			// Fallback for exact string match if filepath.Match is too picky about slashes (though for env keys it's fine)
			if key == pattern {
				matched = true
				break
			}
		}

		if matched {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

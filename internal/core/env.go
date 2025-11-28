package core

import (
	"path/filepath"
	"strings"
)

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

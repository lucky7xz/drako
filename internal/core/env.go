package core

import "strings"

// CommandEnv is base with DRAKO_PROFILE set to the active profile, so scripts can
// rely on it; with no active profile an inherited value passes through.
func CommandEnv(base []string, activeProfile string) []string {
	if activeProfile == "" {
		return base
	}
	out := make([]string, 0, len(base)+1)
	for _, e := range base {
		if !strings.HasPrefix(e, "DRAKO_PROFILE=") {
			out = append(out, e)
		}
	}
	return append(out, "DRAKO_PROFILE="+activeProfile)
}

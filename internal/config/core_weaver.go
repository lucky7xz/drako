package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

// WeaveConfig generates the final config.toml content by weaving the dictionary
// values into the template based on the current runtime environment.
func WeaveConfig(templateContent, dictionaryContent []byte) ([]byte, error) {
	// Parse the dictionary
	var dict map[string]map[string]string
	if _, err := toml.Decode(string(dictionaryContent), &dict); err != nil {
		return nil, fmt.Errorf("failed to decode core dictionary: %w", err)
	}

	// Detect the current runtime target key (e.g., "linux_debian", "macos", "windows")
	targetKey := detectRuntimeTarget()

	// Perform replacements on the template
	// We treat the template as a string and replace {{Key}}
	woven := string(templateContent)

	for cmdName, variants := range dict {
		placeholderRaw := fmt.Sprintf("{{%s}}", cmdName)
		placeholderQuoted := fmt.Sprintf("\"{{%s}}\"", cmdName)

		cmd, ok := variants[targetKey]
		if !ok {
			// Fallback logic
			if runtime.GOOS == "linux" {
				if val, ok := variants["linux_debian"]; ok {
					cmd = val
				} else {
					cmd = fmt.Sprintf("echo 'Command not supported on %s'", targetKey)
				}
			} else {
				cmd = fmt.Sprintf("echo 'Command not supported on %s'", targetKey)
			}
		}

		// Heuristic: Check if the placeholder is likely inside a TOML inline table (items = [ { ... } ])
		// Inline tables must remain on a single line, so we cannot use multi-line strings """...""".
		// We scan the template to see if the placeholder line contains a '{'.
		isInline := false
		lines := strings.Split(woven, "\n")
		for _, line := range lines {
			if strings.Contains(line, cmdName) && strings.Contains(line, "{") {
				isInline = true
				break
			}
		}

		// Determine replacement strategy
		var replacement string
		if strings.Contains(cmd, "\n") && !isInline {
			// Multi-line: Use triple quotes (ONLY if not in inline table)
			replacement = fmt.Sprintf("\"\"\"\n%s\"\"\"", cmd)
		} else {
			// Single-line: Use double quotes and escape newlines
			// We use escapeForTomlString but we MUST escape newlines manually for single-line strings
			escaped := escapeForTomlString(cmd)
			escaped = strings.ReplaceAll(escaped, "\n", "\\n")
			escaped = strings.ReplaceAll(escaped, "\r", "")
			replacement = fmt.Sprintf("\"%s\"", escaped)
		}

		// Apply replacement
		if strings.Contains(woven, placeholderQuoted) {
			woven = strings.ReplaceAll(woven, placeholderQuoted, replacement)
		} else {
			woven = strings.ReplaceAll(woven, placeholderRaw, replacement)
		}
	}

	return []byte(woven), nil
}

func detectRuntimeTarget() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macos"
	case "linux":
		return detectLinuxDistro()
	default:
		return "linux_debian" // Fallback
	}
}

func detectLinuxDistro() string {
	// Simple check of /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux_debian"
	}
	content := string(data)
	contentLower := strings.ToLower(content)

	if strings.Contains(contentLower, "arch") || strings.Contains(contentLower, "manjaro") || strings.Contains(contentLower, "endeavouros") {
		return "linux_arch"
	}
	if strings.Contains(contentLower, "fedora") || strings.Contains(contentLower, "rhel") || strings.Contains(contentLower, "centos") {
		return "linux_fedora"
	}
	// Default to debian/ubuntu family for everything else (Ubuntu, Pop!_OS, Mint, Debian, Kali)
	return "linux_debian"
}

func escapeForTomlString(s string) string {
	// Escape backslashes first
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Escape newlines to make it a valid single-line TOML string
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "") // Strip CR
	return s
}

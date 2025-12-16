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

		// ================================================================
		// Command Replacement
		// ================================================================
		// Priority Chain:
		// 1. Specific Target (e.g. linux_arch)
		// 2. Linux Generic (e.g. linux_generic)
		// 3. Linux Debian (Safety net ONLY if distro is unknown/generic)
		// 4. Error with Link

		cmd, ok := variants[targetKey]
		if !ok {
			// Specific key missing. Try fallbacks.
			if runtime.GOOS == "linux" {
				// Try Generic first
				if val, ok := variants["linux_generic"]; ok {
					cmd = val
				} else {
					// Try Debian only if we are treating this as an unknown/generic distro
					if targetKey == "linux_generic" {
						if val, ok := variants["linux_debian"]; ok {
							cmd = val
						} else {
							cmd = getErrorCommand(cmdName, targetKey)
						}
					} else {
						// Known distro, strictly NO debian fallback
						cmd = getErrorCommand(cmdName, targetKey)
					}
				}
			} else {
				cmd = getErrorCommand(cmdName, targetKey)
			}
		}

		// ================================================================
		// Heuristic: Check if the placeholder is likely inside a TOML inline table (items = [ { ... } ])
		// Inline tables must remain on a single line, so we cannot use multi-line strings """...""".
		// We scan the template to see if the placeholder line contains a '{'.
		// ================================================================
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

// ================================================================
// Runtime Detection
// ================================================================
// DistroKeywords maps a runtime target key to a list of identifying strings found in /etc/os-release.
// To add a new distro, simply append its keywords to the appropriate list or create a new entry.
var DistroKeywords = map[string][]string{
	"linux_arch":   {"arch", "manjaro", "endeavouros", "cachy"},
	"linux_fedora": {"fedora", "rhel", "centos", "nobara"},
	"linux_debian": {"debian", "ubuntu", "pop", "mint", "kali"},
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
		return "linux_generic"
	}
}

// ================================================================
// Linux Distro Detection
// ================================================================
func detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux_generic"
	}
	content := strings.ToLower(string(data))

	// Check against our map
	for key, keywords := range DistroKeywords {
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				return key
			}
		}
	}

	// No specific distro matched
	return "linux_generic"
}

func escapeForTomlString(s string) string {
	// Escape backslashes first
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func getErrorCommand(name, target string) string {
	link := "https://github.com/lucky7xz/drako/blob/main/internal/config/bootstrap/core_dictionary.toml"
	// utilize read to pause execution so user can see it
	return fmt.Sprintf("echo '❌ Could not find command %q for OS %q.' && echo 'Check the dictionary: %s' && read -p 'Press [Enter] to continue...'", name, target, link)
}

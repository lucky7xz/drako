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
		placeholder := fmt.Sprintf("{{%s}}", cmdName)

		cmd, ok := variants[targetKey]
		if !ok {
			// Fallback logic
			if runtime.GOOS == "linux" {
				// Try linux_debian as a generic fallback if specific distro missing
				if val, ok := variants["linux_debian"]; ok {
					cmd = val
				} else {
					cmd = fmt.Sprintf("echo 'Command not supported on %s'", targetKey)
				}
			} else {
				cmd = fmt.Sprintf("echo 'Command not supported on %s'", targetKey)
			}
		}

		// Escape double quotes in the command string since it will be inside a TOML string
		// actually, if the template has `command = "{{Key}}"`
		// and we replace `{{Key}}` with `echo "hello"`, result is `command = "echo "hello""` -> Syntax Error.
		// We should probably rely on the template having `command = "{{Key}}"` and we insert the RAW string?
		// No, usually TOML templates might look like `command = "{{Key}}"`
		// If we replace `{{Key}}` with `something`, we need to be careful about quotes.
		//
		// Strategy:
		// The dictionary values are raw strings like: `sudo apt update`
		// The template acts like: `command = "{{⬆️ System Update}}"`
		// If we blindly replace, we might break TOML syntax if the value contains quotes.
		//
		// Better approach: escape the value logic so it fits in a TOML string?
		// Or maybe the template placeholder should NOT have quotes?
		// In `core_template.toml`, it is: `command = "{{⬆️ System Update}}"`
		// So we are inside quotes. We need to escape existing quotes in our replacement.

		escapedCmd := escapeForTomlString(cmd)
		woven = strings.ReplaceAll(woven, placeholder, escapedCmd)
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
	return s
}

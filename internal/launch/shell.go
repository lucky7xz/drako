package launch

import "runtime"

// ShellArgv is the command line that runs commandStr under the configured shell,
// for single runs and batch cells alike; unknown names get the platform default.
func ShellArgv(shell, commandStr string) []string {
	switch shell {
	case "bash":
		return []string{"bash", "-lc", commandStr}
	case "sh":
		return []string{"sh", "-c", commandStr}
	case "zsh":
		return []string{"zsh", "-lc", commandStr}
	case "fish":
		return []string{"fish", "-c", commandStr}
	case "pwsh":
		return []string{"pwsh", "-NoLogo", "-NoProfile", "-Command", commandStr}
	// powershell.exe 5.1 ships with Windows; pwsh (7+) must be installed.
	case "powershell", "powershell.exe":
		return []string{"powershell", "-NoLogo", "-NoProfile", "-Command", commandStr}
	case "cmd", "cmd.exe":
		return []string{"cmd", "/C", commandStr}
	default:
		return ShellArgv(fallbackShell(runtime.GOOS), commandStr)
	}
}

// fallbackShell picks the shell for unknown/empty shell settings.
func fallbackShell(goos string) string {
	if goos == "windows" {
		return "powershell"
	}
	return "bash"
}

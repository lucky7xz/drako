package launch

import (
	"maps"
	"runtime"
	"slices"
)

// shells is every shell name a deck or config.toml may ask for, with the
// arguments that make it run one command. The single list: ShellArgv runs
// from it and drako check reads it (KnownShell).
var shells = map[string][]string{
	"bash": {"bash", "-lc"},
	"sh":   {"sh", "-c"},
	"zsh":  {"zsh", "-lc"},
	"fish": {"fish", "-c"},
	"pwsh": {"pwsh", "-NoLogo", "-NoProfile", "-Command"},
	// powershell.exe 5.1 ships with Windows; pwsh (7+) must be installed.
	"powershell":     {"powershell", "-NoLogo", "-NoProfile", "-Command"},
	"powershell.exe": {"powershell", "-NoLogo", "-NoProfile", "-Command"},
	"cmd":            {"cmd", "/C"},
	"cmd.exe":        {"cmd", "/C"},
}

// ShellArgv is the command line that runs commandStr under the configured shell,
// for single runs and batch cells alike; unknown names get the platform default.
func ShellArgv(shell, commandStr string) []string {
	args, ok := shells[shell]
	if !ok {
		args = shells[fallbackShell(runtime.GOOS)]
	}
	return append(slices.Clone(args), commandStr)
}

// KnownShell reports whether name is a shell drako can run cells in. Any
// other name runs them in FallbackShell instead.
func KnownShell(name string) bool {
	_, ok := shells[name]
	return ok
}

// ShellNames is every name KnownShell accepts, sorted.
func ShellNames() []string {
	return slices.Sorted(maps.Keys(shells))
}

// FallbackShell is the shell an unknown name runs in on this platform.
func FallbackShell() string {
	return fallbackShell(runtime.GOOS)
}

// fallbackShell picks the shell for unknown/empty shell settings.
func fallbackShell(goos string) string {
	if goos == "windows" {
		return "powershell"
	}
	return "bash"
}

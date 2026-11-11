// Package profile says what a profile file is: its cells and their dropdown
// items, how the text decodes and validates, which command a platform gets,
// and where each cell sits on the grid. It works on bytes and values only;
// reading files is the caller's job.
package profile

// CommandItem represents a single item in a command dropdown
type CommandItem struct {
	Name               string `toml:"name"`
	Command            string `toml:"command"`
	Description        string `toml:"description"`
	AutoCloseExecution *bool  `toml:"auto_close_execution"`
	DebugExecution     *bool  `toml:"debug_execution"`
	// Danger keeps drako run from executing it; only the grid can.
	Danger bool `toml:"danger"`
}

// Command represents a grid command
type Command struct {
	Name               string        `toml:"name"`
	Command            string        `toml:"command"`
	Row                int           `toml:"row"`
	Col                string        `toml:"col"`
	Description        string        `toml:"description"`
	AutoCloseExecution *bool         `toml:"auto_close_execution"`
	DebugExecution     *bool         `toml:"debug_execution"`
	Items              []CommandItem `toml:"items"`
	// Danger keeps drako run from executing it; on a folder, every item too.
	Danger bool `toml:"danger"`
}

// File is the content of a profile file (e.g. core.profile.toml), with every
// command already chosen for one platform.
type File struct {
	X         int       `toml:"x"`
	Y         int       `toml:"y"`
	Theme     string    `toml:"theme"`
	HeaderArt *string   `toml:"header_art"`
	Shell     *string   `toml:"shell"`
	Assets    *[]string `toml:"assets"`
	Commands  []Command `toml:"commands"`
}

// CopyCommands returns a new slice of the same commands, so callers can sort or
// replace cells; Items and the *bool fields still point into the source.
func CopyCommands(src []Command) []Command {
	if len(src) == 0 {
		return []Command{}
	}
	dst := make([]Command, len(src))
	copy(dst, src)
	return dst
}

package proc

import "os/exec"

// Tool is a helper program drako starts for its own purposes. Only this
// package can name one, so the variables below are the whole list.
type Tool struct{ name string }

var (
	// Summoning profiles.
	Git = Tool{"git"}

	// Batch launch.
	Tmux  = Tool{"tmux"}
	Herdr = Tool{"herdr"}

	// Clearing the screen between a finished command and the TUI.
	Clear = Tool{"clear"}

	// Opening a path or URL with the desktop's default application.
	XdgOpen = Tool{"xdg-open"}
	Open    = Tool{"open"}

	// Copying a command to the clipboard.
	WlCopy     = Tool{"wl-copy"}
	Xclip      = Tool{"xclip"}
	Xsel       = Tool{"xsel"}
	Pbcopy     = Tool{"pbcopy"}
	ClipExe    = Tool{"clip.exe"}
	PowerShell = Tool{"powershell.exe"}

	// Windows only: cls to clear the screen, start to open a path.
	Cmd = Tool{"cmd"}
)

// Command prepares t with args. Nothing runs until the caller runs it, so
// each caller wires input and output the way its job needs.
func (t Tool) Command(args ...string) *exec.Cmd {
	return exec.Command(t.name, args...)
}

// Found reports whether t is installed.
func (t Tool) Found() bool {
	_, err := LookPath(t.name)
	return err == nil
}

// String is the program's name, for messages and logs.
func (t Tool) String() string { return t.name }

// LookPath finds a program on PATH. It starts nothing.
func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

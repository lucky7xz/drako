package ui

import "fmt"

// resolveExecMeta turns a command's optional execution flags into the
// display values used by the info popup. A nil AutoCloseExecution defaults
// to true; a nil DebugExecution defaults to live (non-debug).
func resolveExecMeta(autoCloseExec, debugExec *bool) (execMode string, autoClose bool) {
	autoClose = true
	if autoCloseExec != nil {
		autoClose = *autoCloseExec
	}
	debug := false
	if debugExec != nil {
		debug = *debugExec
	}
	execMode = "live"
	if debug {
		execMode = "debug"
	}
	return execMode, autoClose
}

// newCommandDetail assembles the info-popup content for a command. The
// caller supplies the already-resolved value string (so each call site can
// keep its own "empty command" message) plus the optional execution flags
// and current working directory.
func newCommandDetail(title, value, description string, autoCloseExec, debugExec *bool, cwd string) *DetailState {
	execMode, autoClose := resolveExecMeta(autoCloseExec, debugExec)
	return &DetailState{
		Title:       title,
		KeyLabel:    "Command",
		Value:       value,
		Description: description,
		Meta: []DetailMeta{
			{Label: "Exec", Value: execMode},
			{Label: "Auto-close", Value: fmt.Sprintf("%v", autoClose)},
			{Label: "CWD", Value: cwd},
		},
	}
}

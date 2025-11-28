package ui

import (
	"time"

	"github.com/lucky7xz/drako/internal/config"
	gopsutil_net "github.com/shirou/gopsutil/v3/net"
)

// Draco is built on Bubble Tea, which follows the Elm Architecture (Model-View-Update).
// These shared types describe the pieces that move through that loop.

type navMode int

const (
	gridMode navMode = iota
	pathMode
	childMode
	inventoryMode
	dropdownMode
	infoMode
	lockedMode
)

type (
	networkStatusMsg struct {
		online   bool
		counters gopsutil_net.IOCountersStat
		t        time.Time
		err      error
	}
	pathChangedMsg        struct{}
	profileStatusClearMsg struct {
		id int
	}
	navTimeoutMsg struct{}
	lockCheckMsg  struct{}
)

// DetailState defines the content for the info/error popup.
type DetailState struct {
	Title       string
	KeyLabel    string // Label for the main value (e.g. "Command", "Error")
	Value       string // The main content (command string or error message)
	Description string
	Meta        []DetailMeta // Extra fields like "CWD", "Exec Mode"
}

// DetailMeta represents a single key-value pair in the detail view metadata section.
type DetailMeta struct {
	Label string
	Value string
}

// Type aliases to bridge the gap to internal/config temporarily
type Config = config.Config
type Command = config.Command
type CommandItem = config.CommandItem
type ProfileInfo = config.ProfileInfo
type ProfileParseError = config.ProfileParseError
type ConfigBundle = config.ConfigBundle
type InputConfig = config.InputConfig

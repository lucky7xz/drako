package main

import (
	"time"

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
)

type CommandItem struct {
	Name        string `toml:"name"`
	Command     string `toml:"command"`
	Interactive bool   `toml:"interactive"`
	HoldAfter   bool   `toml:"hold_after"`
	Description string `toml:"description"`
}

type Command struct {
	Name        string        `toml:"name"`
	Command     string        `toml:"command"`
	Row         int           `toml:"row"`
	Col         string        `toml:"col"`
	Interactive bool          `toml:"interactive"`
	HoldAfter   bool          `toml:"hold_after"`
	Description string        `toml:"description"`
	Items       []CommandItem `toml:"items"`
}

type Config struct {

	DR4koPath string              `toml:"dR4ko_path"`
	Theme     string              `toml:"theme"`
	Behavior  DracoBehaviorConfig `toml:"behavior"`
	X         int                 `toml:"x"`
	Y         int                 `toml:"y"`
	Profile   string              `toml:"profile"`
	Commands  []Command           `toml:"commands"`

}

type ProfileInfo struct {

	Name    string
	Path    string
	Overlay profileOverlay

}

type configBundle struct {

	Base        Config
	Config      Config
	Profiles    []ProfileInfo
	ActiveIndex int
	ConfigDir   string
	LockedName  string

}


type DracoBehaviorConfig struct {
	ExitConfirmation bool `toml:"exit_confirmation"`
	AutoSave         bool `toml:"auto_save"`
}

type behaviorOverlay struct {
	ExitConfirmation *bool `toml:"exit_confirmation"`
	AutoSave         *bool `toml:"auto_save"`
}

type profileOverlay struct {
	DR4koPath *string          `toml:"dR4ko_path"`
	X         *int             `toml:"x"`
	Y         *int             `toml:"y"`
	Theme     *string          `toml:"theme"`
	Behavior  *behaviorOverlay `toml:"behavior"`
	Commands  *[]Command       `toml:"commands"`
}


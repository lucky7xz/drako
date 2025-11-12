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
	navTimeoutMsg struct{}
)

type CommandItem struct {
	Name               string `toml:"name"`
	Command            string `toml:"command"`
	Description        string `toml:"description"`
	AutoCloseExecution *bool  `toml:"auto_close_execution"`
	DebugExecution     *bool  `toml:"debug_execution"`
}

type Command struct {
	Name               string        `toml:"name"`
	Command            string        `toml:"command"`
	Row                int           `toml:"row"`
	Col                string        `toml:"col"`
	Description        string        `toml:"description"`
	AutoCloseExecution *bool         `toml:"auto_close_execution"`
	DebugExecution     *bool         `toml:"debug_execution"`
	Items              []CommandItem `toml:"items"`
}

type Config struct {
	DR4koPath    string    `toml:"dR4ko_path"`
	Theme        string    `toml:"theme"`
	HeaderArt    *string   `toml:"header_art"`
	DefaultShell string    `toml:"default_shell"`
	NumbModifier string    `toml:"numb_modifier"`
	X            int       `toml:"x"`
	Y            int       `toml:"y"`
	Profile      string    `toml:"profile"`
	Commands     []Command `toml:"commands"`
}

type ProfileInfo struct {

	Name    string
	Path    string
	Overlay profileOverlay

}

type ProfileParseError struct {
	Name string
	Path string
	Err  string
}

type configBundle struct {

	Base        Config
	Config      Config
	Profiles    []ProfileInfo
	ActiveIndex int
	ConfigDir   string
	LockedName  string
	Broken      []ProfileParseError

}


type profileOverlay struct {
	DR4koPath    *string          `toml:"dR4ko_path"`
	X            *int             `toml:"x"`
	Y            *int             `toml:"y"`
	Theme        *string    `toml:"theme"`
	HeaderArt    *string    `toml:"header_art"`
	DefaultShell *string    `toml:"default_shell"`
	NumbModifier *string    `toml:"numb_modifier"`
	Assets       *[]string  `toml:"assets"`
	Commands     *[]Command `toml:"commands"`
}


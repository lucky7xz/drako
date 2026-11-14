package settings

import (
	"slices"
	"strings"
)

// The key table: which key does what on which screen. It is the one
// description of drako's keys. The TUI acts on it (a screen's handler asks
// Act), the key sheet prints it, and drako check reads it to say which
// bindings cannot work.
//
// A screen's binds stand in the order they are asked: the first that claims a
// key gets it. That order only shows when two binds claim one key, which is
// what drako check warns about.

// An Action is what a screen does with a key.
type Action int

const (
	NoAction Action = iota
	Quit            // ctrl+c
	LockSession
	Help
	PinDeck
	SwitchDeck // the modifier chord, alt+1-9
	PickDeck   // the leader, then 1-9
	PrevDeck
	NextDeck
	Leader
	Batch // the leader, then b
	Jump  // 1-9
	Back  // q, esc
	OpenInventory
	PathMode
	Move    // any direction
	MoveV   // up, down
	MoveH   // left, right
	Descend // down, from the path bar into the folders below it
	Explain
	Confirm // enter, space
	Mark
	Launch
	Search
	ToggleHidden
	Specs
	Edit
	Trash
	Scroll
	Copy
	Close
)

// A Bind is one line of the table: these keys do this.
type Bind struct {
	Setting string   // its name under [keys] in config.toml; "" for a key drako fixes
	Keys    []string // as the terminal names them
	Any     bool     // claims every key: what the screen does with the rest
	Act     Action
	Show    string // how the key sheet writes the keys; "" writes Keys
	Does    string // the key sheet's words for it
	Gated   bool   // glassroot blocks it
}

// Label is the keys as the key sheet writes them.
func (b Bind) Label() string {
	if b.Show != "" {
		return b.Show
	}
	return strings.Join(b.Keys, " ")
}

func (b Bind) claims(key string) bool {
	return b.Any || slices.Contains(b.Keys, key)
}

// A Screen is a state of the TUI with its own answer to a key.
type Screen struct {
	Name  string // as drako check calls it
	Sheet string // the key sheet section it is listed in
	Q     string // what q does here, for the footer; "" when q is not its own key
	Binds []Bind
}

// Find is the bind that gets key on this screen: the first to claim it.
func (s Screen) Find(key string) (Bind, bool) {
	for _, b := range s.Binds {
		if b.claims(key) {
			return b, true
		}
	}
	return Bind{}, false
}

// Act is what key does on this screen.
func (s Screen) Act(key string) Action {
	b, _ := s.Find(key)
	return b.Act
}

// Bind is the screen's first bind for act, for the key sheet.
func (s Screen) Bind(act Action) (Bind, bool) {
	for _, b := range s.Binds {
		if b.Act == act {
			return b, true
		}
	}
	return Bind{}, false
}

// A ScreenID names a screen of the table.
type ScreenID int

const (
	// Typing is every state where the keyboard is not the table's: a text
	// field, the lock screen, a message any key dismisses. Only the keys that
	// work everywhere remain.
	Typing ScreenID = iota
	Grid
	GridLeader // the grid, after the leader key
	Folder
	FolderLeader  // a folder, after the leader key
	FolderMarking // a folder, marking items for a batch
	BatchMarking  // the grid, marking cells for a batch
	BatchLayout   // the batch layout dialog
	PathBar
	PathChildren
	Inventory
	InventorySpecs // the inventory's spec picker
	ExplainPopup
	screenCount
)

// Screen is the key table for one screen under this config, which must have
// its defaults applied.
func (c Config) Screen(id ScreenID) Screen {
	k := c.Keys
	mod := c.NumbModifier

	set := func(name, key string, act Action, does string) Bind {
		return Bind{Setting: name, Keys: []string{key}, Act: act, Does: does}
	}
	gate := func(b Bind) Bind { b.Gated = true; return b }
	keys := func(sets ...[]string) []string {
		var all []string
		for _, s := range sets {
			all = append(all, s...)
		}
		return all
	}
	digits := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	chords := make([]string, len(digits))
	for i, d := range digits {
		chords[i] = mod + "+" + d
	}
	upDown := k.moveLabel("↑↓", "ws", "kj")
	leftRight := k.moveLabel("←→", "ad", "hl")
	moveV := func(does string) Bind {
		return Bind{Keys: keys(k.NavUp, k.NavDown), Act: MoveV, Show: upDown, Does: does}
	}
	moveH := func(does string) Bind {
		return Bind{Keys: keys(k.NavLeft, k.NavRight), Act: MoveH, Show: leftRight, Does: does}
	}
	back := func(does string) Bind {
		return Bind{Keys: []string{"q", "esc"}, Act: Back, Show: "esc q", Does: does}
	}
	confirm := func(does string) Bind {
		return Bind{Keys: []string{"enter", " "}, Act: Confirm, Show: "enter", Does: does}
	}

	// These work wherever you are, typing included.
	always := []Bind{
		{Keys: []string{"ctrl+c"}, Act: Quit, Does: "quit"},
		{Keys: []string{mod + "+" + k.SessionLock}, Act: LockSession, Does: "lock session"},
	}
	help := set("help", k.Help, Help, "this sheet")
	// These act on the deck in use, so they belong to the grid alone.
	decks := []Bind{
		gate(set("lock", k.Lock, PinDeck, "pin deck")),
		{Keys: chords, Act: SwitchDeck, Show: mod + "+1-9", Does: "switch deck"},
		set("profile_prev", k.ProfilePrev, PrevDeck, "previous deck"),
		set("profile_next", k.ProfileNext, NextDeck, "next deck"),
	}
	leader := set("leader", k.Leader, Leader, "leader")
	explain := set("explain", k.Explain, Explain, "explain")
	mark := Bind{Keys: []string{" "}, Act: Mark, Show: "space", Does: "mark"}
	launch := func(does string) Bind { return Bind{Keys: []string{"enter"}, Act: Launch, Does: does} }
	jump := func(show, does string) Bind { return Bind{Keys: digits, Act: Jump, Show: show, Does: does} }
	pathKeys := []Bind{
		set("path_search", k.PathSearch, Search, "search"),
	}
	pathRest := []Bind{
		set("path_grid_mode", k.PathGridMode, PathMode, "back to grid"),
		confirm("cd"),
		set("toggle_hidden", k.ToggleHidden, ToggleHidden, "hidden files"),
	}
	pathBack := Bind{Keys: []string{"q", "esc"}, Act: Back, Show: k.PathGridMode + " esc q", Does: "back to grid"}

	var name, sheet, q string
	var binds [][]Bind
	one := func(b ...Bind) []Bind { return b }
	switch id {
	case Typing:
		name = "a text field"
		sheet, q = "GRID", "quit"
	case Grid:
		name = "the grid"
		sheet, q = "GRID", "quit"
		binds = [][]Bind{one(help), decks, one(
			leader,
			jump("1-9 1-9", "jump: column, row"),
			Bind{Keys: []string{"q"}, Act: Back, Does: "quit"},
			gate(set("inventory", k.Inventory, OpenInventory, "inventory")),
			gate(set("path_grid_mode", k.PathGridMode, PathMode, "path mode")),
			Bind{Keys: keys(k.NavUp, k.NavDown, k.NavLeft, k.NavRight), Act: Move, Show: k.moveLabel("←↑↓→", "wasd", "hjkl"), Does: "move"},
			explain,
			confirm("run cell, open folder"),
		)}
	case GridLeader:
		name = "the grid"
		sheet, q = "GRID", "quit"
		binds = [][]Bind{decks, one(
			Bind{Keys: digits, Act: PickDeck, Show: k.Leader + " 1-9", Does: "switch deck"},
			gate(Bind{Keys: []string{"b"}, Act: Batch, Show: k.Leader + " b", Does: "batch mode"}),
		)}
	case Folder:
		name = "a folder"
		sheet, q = "FOLDER", "close"
		binds = [][]Bind{one(help, leader, jump("1-9", "jump to item"), back("close"), moveV("select"), explain, confirm("run"))}
	case FolderLeader:
		name = "a folder"
		sheet, q = "FOLDER", "close"
		binds = [][]Bind{one(gate(Bind{Keys: []string{"b"}, Act: Batch, Show: k.Leader + " b", Does: "batch these"}))}
	case FolderMarking:
		name = "a folder"
		sheet, q = "BATCH", "cancel"
		binds = [][]Bind{one(help, leader, back("cancel"), mark, launch("lay out"), jump("1-9", "jump to item"), moveV("select"), explain)}
	case BatchMarking:
		name = "batch mode"
		sheet, q = "BATCH", "cancel"
		binds = [][]Bind{one(help, jump("1-9 1-9", "jump: column, row"), back("cancel"), mark, launch("lay out, launch"),
			Bind{Keys: keys(k.NavUp, k.NavDown, k.NavLeft, k.NavRight), Act: Move, Show: k.moveLabel("←↑↓→", "wasd", "hjkl"), Does: "move"})}
	case BatchLayout:
		name = "the batch layout"
		sheet, q = "BATCH", "back"
		binds = [][]Bind{one(help, back("back"), launch("launch"), moveH("pick layout field"), moveV("adjust it"))}
	case PathBar:
		name = "path mode"
		sheet, q = "PATH", "back"
		binds = [][]Bind{one(help, pathBack), pathKeys, one(
			moveH("pick folder"),
			Bind{Keys: k.NavDown, Act: Descend, Show: k.moveLabel("↓", "s", "j"), Does: "children"},
		), pathRest}
	case PathChildren:
		name = "path mode"
		sheet, q = "PATH", "back"
		binds = [][]Bind{one(help, pathBack), pathKeys, one(moveV("pick child")), pathRest}
	case Inventory:
		name = "the inventory"
		sheet, q = "INVENTORY", "back"
		binds = [][]Bind{one(help, back("back"), moveV("switch list"), moveH("move along"),
			set("path_grid_mode", k.PathGridMode, Specs, "specs"),
			set("edit_file", k.EditFile, Edit, "edit"),
			set("delete", k.Delete, Trash, "trash"),
			confirm("lift, place"),
		)}
	case InventorySpecs:
		name = "the spec picker"
		sheet, q = "INVENTORY", "back"
		binds = [][]Bind{one(help, set("path_grid_mode", k.PathGridMode, Back, "close"), back("close"), moveV("pick spec"), confirm("stage"))}
	case ExplainPopup:
		name = "the explain popup"
		sheet, q = "EXPLAIN", ""
		binds = [][]Bind{one(
			Bind{Keys: []string{"pgup", "pgdown"}, Act: Scroll, Show: "pgup pgdn", Does: "scroll"},
			gate(Bind{Keys: []string{"y"}, Act: Copy, Does: "copy"}),
			Bind{Any: true, Act: Close, Show: "any key", Does: "close"},
		)}
	}

	s := Screen{Name: name, Sheet: sheet, Q: q, Binds: always}
	for _, b := range binds {
		s.Binds = append(s.Binds, b...)
	}
	return s
}

// Screens is the whole table, screen by screen.
func (c Config) Screens() []Screen {
	var all []Screen
	for id := Typing; id < screenCount; id++ {
		all = append(all, c.Screen(id))
	}
	return all
}

// moveLabel names a set of direction keys: the arrows, then the wasd and vim
// equivalents unless the config disables them, as InitControls does.
func (k InputConfig) moveLabel(arrows, wasd, vim string) string {
	label := arrows
	if !k.DisableWasd {
		label += " " + wasd
	}
	if !k.DisableVim {
		label += " " + vim
	}
	return label
}

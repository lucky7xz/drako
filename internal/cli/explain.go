package cli

// drako explain zooms into one grid cell: the CLI twin of the TUI's 'e'
// popup. Addresses use the same scheme drako ls prints (A0, work:B2,
// work:A1.2 for dropdown items), so the two commands can never disagree.

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lucky7xz/drako/internal/config"
	profilepkg "github.com/lucky7xz/drako/internal/profiles"
)

// HandleExplainCommand processes 'drako explain [profile:]<addr> ...'.
// Returns the process exit code: 0 when every address resolved, 1 otherwise
// (grep-style partial success — hits print, misses go to stderr).
func HandleExplainCommand(args []string) int {
	refs := args[2:]
	if len(refs) == 0 {
		fmt.Fprint(os.Stderr, "Usage: drako explain [profile:]<addr> ...\n"+
			"Examples: drako explain A0 | drako explain work:B2 | drako explain A1.2\n")
		return 1
	}

	quietLogs()
	bundle, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako: %v\n", err)
		return 1
	}

	code := 0
	printed := false
	for _, ref := range refs {
		var block strings.Builder
		if err := explainCell(&block, bundle, ref); err != nil {
			fmt.Fprintf(os.Stderr, "drako explain: %v\n", err)
			code = 1
			continue
		}
		if printed {
			fmt.Println()
		}
		fmt.Print(block.String())
		printed = true
	}
	return code
}

// cellTarget is what a [profile:]<addr>[.item] reference points at.
type cellTarget struct {
	p      config.ProfileInfo
	cmd    *config.Command
	item   *config.CommandItem // nil unless the reference names an item
	addr   string              // the cell's address, or the item's
	danger bool                // a danger folder covers its items
}

func (t cellTarget) name() string {
	if t.item != nil {
		return t.item.Name
	}
	return t.cmd.Name
}

func (t cellTarget) command() string {
	if t.item != nil {
		return t.item.Command
	}
	return t.cmd.Command
}

// lookupCell resolves a reference against the equipped profiles; explain and
// run both go through it so they agree on what an address means.
func lookupCell(bundle config.ConfigBundle, ref string) (cellTarget, error) {
	profileName, addr, itemIdx, err := parseCellRef(ref)
	if err != nil {
		return cellTarget{}, err
	}
	p, err := pickProfile(bundle, profileName)
	if err != nil {
		return cellTarget{}, err
	}
	cmd := config.FindCell(p.Profile.Commands, addr)
	if cmd == nil {
		return cellTarget{}, fmt.Errorf("no cell %s in profile %s", addr, p.Name)
	}
	if over := config.CoveredBy(p.Profile, cmd); over != nil {
		return cellTarget{}, fmt.Errorf("%s (%s) is hidden on the grid behind %s (%s)", addr, cmd.Name, over.Address(), over.Name)
	}

	t := cellTarget{p: p, cmd: cmd, addr: addr, danger: cmd.Danger}
	if itemIdx > 0 {
		if itemIdx > len(cmd.Items) {
			return cellTarget{}, fmt.Errorf("no item %d in cell %s (%d items)", itemIdx, addr, len(cmd.Items))
		}
		t.item = &cmd.Items[itemIdx-1]
		t.addr = cmd.ItemAddress(itemIdx)
		t.danger = t.danger || t.item.Danger
	}
	return t, nil
}

// explainCell resolves one [profile:]address reference against the equipped
// profiles and writes its key-value block.
func explainCell(out io.Writer, bundle config.ConfigBundle, ref string) error {
	t, err := lookupCell(bundle, ref)
	if err != nil {
		return err
	}

	kv := func(key, value string) { fmt.Fprintf(out, "%-12s %s\n", key+":", safeText(value)) }
	kv("profile", t.p.Name)
	kv("cell", t.addr)
	kv("name", t.name())

	if item := t.item; item != nil {
		kv("command", commandOrNone(item.Command))
		if item.Description != "" {
			kv("description", item.Description)
		}
		kv("auto_close", autoCloseValue(item.AutoCloseExecution))
		if t.danger {
			kv("danger", "true")
		}
		return nil
	}

	cmd := t.cmd
	// A pure folder (items, no own command) has nothing to run directly.
	runnable := cmd.Command != "" || len(cmd.Items) == 0
	if runnable {
		kv("command", commandOrNone(cmd.Command))
	}
	if cmd.Description != "" {
		kv("description", cmd.Description)
	}
	if runnable {
		kv("auto_close", autoCloseValue(cmd.AutoCloseExecution))
	}
	if t.danger {
		kv("danger", "true")
	}
	if len(cmd.Items) > 0 {
		fmt.Fprintln(out, "items:")
		for k, item := range cmd.Items {
			line := fmt.Sprintf("  %-*s %s", len(t.addr)+4, cmd.ItemAddress(k+1), safeText(item.Name))
			if item.Command != "" {
				// Summary line only — `explain A1.2` shows the full command.
				line += " — " + oneLine(safeText(item.Command), lsDescriptionWidth)
			}
			fmt.Fprintln(out, line)
		}
	}
	return nil
}

// pickProfile selects the active profile ("" qualifier) or matches an
// equipped one by normalized name — the same normalization the loader uses.
func pickProfile(bundle config.ConfigBundle, name string) (config.ProfileInfo, error) {
	if len(bundle.Profiles) == 0 {
		return config.ProfileInfo{}, fmt.Errorf("no equipped profiles")
	}
	if name == "" {
		idx := bundle.ActiveIndex
		if idx < 0 || idx >= len(bundle.Profiles) {
			idx = 0
		}
		return bundle.Profiles[idx], nil
	}
	target := profilepkg.NormalizeName(name)
	for _, p := range bundle.Profiles {
		if profilepkg.NormalizeName(p.Name) == target {
			return p, nil
		}
	}
	names := make([]string, len(bundle.Profiles))
	for i, p := range bundle.Profiles {
		names[i] = p.Name
	}
	return config.ProfileInfo{}, fmt.Errorf("no equipped profile %q (equipped: %s)", name, strings.Join(names, ", "))
}

func commandOrNone(s string) string {
	if s == "" {
		return "(none for this platform)"
	}
	return s
}

func autoCloseValue(ptr *bool) string {
	switch {
	case ptr == nil:
		return "true (default)"
	case *ptr:
		return "true"
	default:
		return "false"
	}
}

// parseCellRef splits "[profile:]COLROW[.ITEM]" into its parts. The profile
// qualifier may be empty (active profile); the address is normalized to
// upper case; item is 1-based, 0 when absent.
func parseCellRef(ref string) (profile, addr string, item int, err error) {
	rest := ref
	if i := strings.IndexByte(ref, ':'); i >= 0 {
		profile = ref[:i]
		rest = ref[i+1:]
	}

	bad := func() (string, string, int, error) {
		return "", "", 0, fmt.Errorf("bad cell address %q (expected e.g. A0, work:B2, A1.2)", ref)
	}

	cell := rest
	if i := strings.IndexByte(rest, '.'); i >= 0 {
		cell = rest[:i]
		itemPart := rest[i+1:]
		if _, err := fmt.Sscanf(itemPart+"\n", "%d\n", &item); err != nil || item < 1 {
			return bad()
		}
		// Reject trailing garbage like "A1.2.3" or "A1.2x".
		if fmt.Sprintf("%d", item) != itemPart {
			return bad()
		}
	}

	if len(cell) < 2 {
		return bad()
	}
	col := cell[0]
	if !('a' <= col && col <= 'z' || 'A' <= col && col <= 'Z') {
		return bad()
	}
	// row = -1 (last row) is addressed as written, e.g. A-1.
	rowStart := 1
	if cell[1] == '-' {
		rowStart = 2
	}
	if rowStart == len(cell) {
		return bad()
	}
	for i := rowStart; i < len(cell); i++ {
		if cell[i] < '0' || cell[i] > '9' {
			return bad()
		}
	}

	return profile, strings.ToUpper(cell), item, nil
}

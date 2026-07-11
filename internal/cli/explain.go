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
	bundle, err := config.LoadConfig(nil)
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

// explainCell resolves one [profile:]address reference against the equipped
// profiles and writes its key-value block. The address is matched against
// cellAddress — the exact scheme drako ls prints — so the two commands can
// never disagree about what "A0" means.
func explainCell(out io.Writer, bundle config.ConfigBundle, ref string) error {
	profileName, addr, itemIdx, err := parseCellRef(ref)
	if err != nil {
		return err
	}

	p, err := pickProfile(bundle, profileName)
	if err != nil {
		return err
	}

	var cmd *config.Command
	for i := range p.Profile.Commands {
		if strings.EqualFold(cellAddress(p.Profile.Commands[i]), addr) {
			cmd = &p.Profile.Commands[i]
			break
		}
	}
	if cmd == nil {
		return fmt.Errorf("no cell %s in profile %s", addr, p.Name)
	}

	kv := func(key, value string) { fmt.Fprintf(out, "%-12s %s\n", key+":", value) }

	if itemIdx > 0 {
		if len(cmd.Items) == 0 {
			return fmt.Errorf("cell %s (%s) has no items", addr, cmd.Name)
		}
		if itemIdx > len(cmd.Items) {
			return fmt.Errorf("no item %d in cell %s (%d items)", itemIdx, addr, len(cmd.Items))
		}
		item := cmd.Items[itemIdx-1]
		kv("profile", p.Name)
		kv("cell", fmt.Sprintf("%s.%d", addr, itemIdx))
		kv("name", item.Name)
		kv("command", commandOrNone(item.Command))
		if item.Description != "" {
			kv("description", item.Description)
		}
		kv("auto_close", autoCloseValue(item.AutoCloseExecution))
		return nil
	}

	kv("profile", p.Name)
	kv("cell", addr)
	kv("name", cmd.Name)
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
	if len(cmd.Items) > 0 {
		fmt.Fprintln(out, "items:")
		for k, item := range cmd.Items {
			line := fmt.Sprintf("  %s.%-3d %s", addr, k+1, item.Name)
			if item.Command != "" {
				// Summary line only — `explain A1.2` shows the full command.
				line += " — " + oneLine(item.Command, lsDescriptionWidth)
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
	for i := 1; i < len(cell); i++ {
		if cell[i] < '0' || cell[i] > '9' {
			return bad()
		}
	}

	return profile, strings.ToUpper(cell), item, nil
}

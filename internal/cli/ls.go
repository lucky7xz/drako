package cli

// drako ls prints every equipped profile with its commands and their cell
// addresses — the map a human or an AI works from before explaining or
// running a cell. Inventory is quarantine and is never listed here.

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/paths"
)

const lsDescriptionWidth = 60

// HandleLsCommand processes 'drako ls'. Returns the process exit code.
func HandleLsCommand(args []string) int {
	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "Usage: drako ls\n")
		return 1
	}

	// Keep stdout pure listing: route log output to the usual log file
	// (or drop it) — an AI or a pipe reads this output verbatim.
	log.SetOutput(io.Discard)
	if configDir, err := paths.ConfigDir(); err == nil {
		if f, err := os.OpenFile(paths.LogFile(configDir), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644); err == nil {
			defer f.Close()
			log.SetOutput(f)
		}
	}

	bundle, err := config.LoadConfig(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako: %v\n", err)
		return 1
	}
	renderLs(os.Stdout, bundle)
	return 0
}

// cellAddress renders a command's grid coordinate, e.g. "A0".
func cellAddress(c config.Command) string {
	return strings.ToUpper(c.Col) + fmt.Sprintf("%d", c.Row)
}

// deckOrder sorts commands by column letter, then row.
func deckOrder(cmds []config.Command) []config.Command {
	sorted := config.CopyCommands(cmds)
	sort.SliceStable(sorted, func(i, j int) bool {
		ci := strings.ToLower(sorted[i].Col)
		cj := strings.ToLower(sorted[j].Col)
		if ci != cj {
			return ci < cj
		}
		return sorted[i].Row < sorted[j].Row
	})
	return sorted
}

// oneLine truncates a description to a single readable line, measured in
// terminal cells so emoji-heavy text still fits its column.
func oneLine(s string, width int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return ansi.Truncate(s, width, "…")
}

// renderLs writes the equipped-deck listing.
func renderLs(out io.Writer, bundle config.ConfigBundle) {
	if len(bundle.Profiles) == 0 {
		fmt.Fprintln(out, "No equipped profiles.")
	}

	for i, p := range bundle.Profiles {
		marker := ""
		if i == bundle.ActiveIndex {
			marker = " (active)"
		}
		fmt.Fprintf(out, "\n%s%s — %dx%d\n", p.Name, marker, p.Profile.X, p.Profile.Y)

		cmds := deckOrder(p.Profile.Commands)
		if len(cmds) == 0 {
			fmt.Fprintln(out, "  (no commands)")
			continue
		}

		rows := make([][]string, 0, len(cmds))
		for _, c := range cmds {
			addr := cellAddress(c)
			rows = append(rows, []string{addr, c.Name, oneLine(c.Description, lsDescriptionWidth)})
			for k, item := range c.Items {
				rows = append(rows, []string{
					fmt.Sprintf("%s.%d", addr, k+1),
					"  " + item.Name,
					oneLine(item.Description, lsDescriptionWidth),
				})
			}
		}
		table(out, []string{"ADDR", "NAME", "DESCRIPTION"}, rows)
	}

	for _, b := range bundle.Broken {
		fmt.Fprintf(out, "\n⚠️  %s is broken and hidden: %s\n", b.Name, oneLine(b.Err, lsDescriptionWidth))
	}
}

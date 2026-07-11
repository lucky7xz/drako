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
	"golang.org/x/term"
)

const (
	// lsDescriptionWidth caps the description column even in wide terminals.
	lsDescriptionWidth = 60
	// lsMinDescription is the narrowest useful description column; below it
	// the column is dropped entirely rather than rendered as confetti.
	lsMinDescription = 12
	// lsMaxNameWidth caps the name column so one long cell name can't starve
	// the descriptions.
	lsMaxNameWidth = 32
	// lsFallbackWidth is assumed when stdout is not a terminal (pipes, AI
	// agents) — stable output regardless of the host window.
	lsFallbackWidth = 80
)

// quietLogs keeps stdout a pure listing for pipes and AI agents: log output
// goes to the usual log file, or nowhere. The file stays open for the rest of
// this short-lived CLI process.
func quietLogs() {
	log.SetOutput(io.Discard)
	if configDir, err := paths.ConfigDir(); err == nil {
		if f, err := os.OpenFile(paths.LogFile(configDir), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644); err == nil {
			log.SetOutput(f)
		}
	}
}

// lsWidth is the terminal width to fit the listing into.
func lsWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return lsFallbackWidth
}

// HandleLsCommand processes 'drako ls'. Returns the process exit code.
func HandleLsCommand(args []string) int {
	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "Usage: drako ls\n")
		return 1
	}

	quietLogs()

	bundle, err := config.LoadConfig(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako: %v\n", err)
		return 1
	}
	renderLs(os.Stdout, bundle, lsWidth())
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

// renderLs writes the equipped-deck listing, fitted to width terminal cells:
// the description column shrinks to the remaining space and is dropped
// entirely when a useful width can't be met.
func renderLs(out io.Writer, bundle config.ConfigBundle, width int) {
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

		type rawRow struct{ addr, name, desc string }
		raws := make([]rawRow, 0, len(cmds))
		for _, c := range cmds {
			addr := cellAddress(c)
			raws = append(raws, rawRow{addr, c.Name, c.Description})
			for k, item := range c.Items {
				raws = append(raws, rawRow{fmt.Sprintf("%s.%d", addr, k+1), "  " + item.Name, item.Description})
			}
		}

		// Fit: fixed ADDR column, capped NAME column, DESCRIPTION takes what
		// remains. Per column the table adds 3 chars of chrome, plus 1.
		addrW := ansi.StringWidth("ADDR")
		nameW := ansi.StringWidth("NAME")
		for _, r := range raws {
			addrW = max(addrW, ansi.StringWidth(r.addr))
			nameW = max(nameW, ansi.StringWidth(r.name))
		}
		nameW = min(nameW, lsMaxNameWidth)

		descW := min(lsDescriptionWidth, width-addrW-nameW-10)
		withDesc := descW >= lsMinDescription
		if !withDesc {
			// Two columns: give the name what the terminal has left.
			nameW = min(nameW, max(width-addrW-7, 8))
		}

		rows := make([][]string, 0, len(raws))
		for _, r := range raws {
			// Names keep their leading indent (dropdown items), so truncate
			// without oneLine's TrimSpace.
			row := []string{r.addr, ansi.Truncate(r.name, nameW, "…")}
			if withDesc {
				row = append(row, oneLine(r.desc, descW))
			}
			rows = append(rows, row)
		}
		headers := []string{"ADDR", "NAME"}
		if withDesc {
			headers = append(headers, "DESCRIPTION")
		}
		table(out, headers, rows)
	}

	for _, b := range bundle.Broken {
		line := fmt.Sprintf("⚠️  %s is broken and hidden: %s", b.Name, oneLine(b.Err, lsDescriptionWidth))
		fmt.Fprintf(out, "\n%s\n", ansi.Truncate(line, width, "…"))
	}
}

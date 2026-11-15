package profile

import (
	"errors"
	"fmt"
	"strings"
)

// letterToColumn translates a column letter ('a'-'y') to its zero-based
// index. 'z' means "last column" and maps to -1, resolved against the grid
// width by the caller.
func letterToColumn(s string) (int, error) {
	if len(s) != 1 {
		return 0, errors.New("column must be a single letter")
	}
	char := strings.ToLower(s)[0]

	if char == 'z' {
		return -1, nil
	}

	if char < 'a' || char >= 'z' {
		return 0, errors.New("column must be a letter from 'a' to 'y', or 'z' for the last column")
	}
	return int(char - 'a'), nil
}

// Address is the cell's address for drako ls, explain and run: column and row
// as the deck writes them, e.g. "A0".
func (c Command) Address() string {
	return fmt.Sprintf("%s%d", strings.ToUpper(c.Col), c.Row)
}

// ItemAddress addresses item n (1-based) of the cell's dropdown, e.g. "C2.6".
func (c Command) ItemAddress(n int) string {
	return fmt.Sprintf("%s.%d", c.Address(), n)
}

// FindCell returns the command at addr, ignoring case, or nil. Like BuildGrid,
// the last command placed at an address is the one that counts.
func FindCell(cmds []Command, addr string) *Command {
	for i := len(cmds) - 1; i >= 0; i-- {
		if strings.EqualFold(cmds[i].Address(), addr) {
			return &cmds[i]
		}
	}
	return nil
}

// CoveredBy returns the command the grid shows on c's square instead of c, or
// nil when c is visible. c must point into p.Commands.
func CoveredBy(p File, c *Command) *Command {
	row, col, err := place(*c, p.X, p.Y)
	if err != nil {
		return nil
	}
	for j := len(p.Commands) - 1; j >= 0 && &p.Commands[j] != c; j-- {
		if r, cl, err := place(p.Commands[j], p.X, p.Y); err == nil && r == row && cl == col {
			return &p.Commands[j]
		}
	}
	return nil
}

// place says which square of a grid of x columns and y rows a command stands
// on, or why it stands on none. A row of -1 or column 'z' means "last". This
// is the one place that decides it: the grid, the cursor, drako check and
// drako run all ask here.
func place(cmd Command, x, y int) (row, col int, err error) {
	if cmd.Col == "" {
		return 0, 0, errors.New("has no col")
	}
	col, err = letterToColumn(cmd.Col)
	if err != nil {
		return 0, 0, fmt.Errorf("has invalid col %q: %v", cmd.Col, err)
	}
	row = cmd.Row
	if row < -1 {
		return 0, 0, fmt.Errorf("has invalid row %d (rows start at 0; -1 is the last)", row)
	}
	if row == -1 {
		row = y - 1
	}
	if col == -1 {
		col = x - 1
	}
	switch {
	case col >= x || col < 0:
		return 0, 0, fmt.Errorf("sits in column %q but the grid is only %d wide (x = %d)", cmd.Col, x, x)
	case row >= y || row < 0:
		return 0, 0, fmt.Errorf("sits in row %d but the grid is only %d tall (y = %d)", cmd.Row, y, y)
	}
	return row, col, nil
}

// Placed reports why c stands on no square of p's grid, or nil when it does.
func Placed(p File, c Command) error {
	_, _, err := place(c, p.X, p.Y)
	return err
}

// ShownAt is the index of the command the grid shows on the square at row and
// col (both from 0), or -1 when the square is empty. The last command placed
// on a square is the one that counts.
func ShownAt(cmds []Command, x, y, row, col int) int {
	for i := len(cmds) - 1; i >= 0; i-- {
		if r, c, err := place(cmds[i], x, y); err == nil && r == row && c == col {
			return i
		}
	}
	return -1
}

// BuildGrid places command names into a y-by-x grid of cells. A command that
// stands on no square is left out; Flaws names it.
func BuildGrid(x, y int, cmds []Command) [][]string {
	grid := make([][]string, max(y, 0))
	for i := range grid {
		grid[i] = make([]string, max(x, 0))
	}
	for _, cmd := range cmds {
		if row, col, err := place(cmd, x, y); err == nil {
			grid[row][col] = cmd.Name
		}
	}
	return grid
}

// Flaws lists what is wrong with a deck that still loads: cells and folder
// items without a name, cells that stand on no square, cells sharing a
// square, and names used twice. The loader keeps
// such a deck and passes these on; drako check prints the same list.
func Flaws(p File) []Finding {
	var fs []Finding
	label := func(i int) string {
		if name := strings.TrimSpace(p.Commands[i].Name); name != "" {
			return fmt.Sprintf("command %q", name)
		}
		return fmt.Sprintf("command #%d", i+1)
	}

	for i, c := range p.Commands {
		if strings.TrimSpace(c.Name) == "" {
			fs = append(fs, errorf("%s has no name — it draws empty", label(i)))
		}
		for j, item := range c.Items {
			if strings.TrimSpace(item.Name) == "" {
				fs = append(fs, errorf("%s, item #%d has no name — it draws empty", label(i), j+1))
			}
		}
	}

	seenNames := map[string]bool{}
	for _, c := range p.Commands {
		name := strings.TrimSpace(c.Name)
		if name != "" && seenNames[name] {
			fs = append(fs, warnf("duplicate command name %q — the cells look the same on the grid and in the history", name))
		}
		seenNames[name] = true
	}

	if p.X < 1 || p.Y < 1 {
		return fs // no grid to stand on; the loader has refused the deck for it
	}
	first := map[[2]int]int{} // square -> first command on it
	for i, c := range p.Commands {
		row, col, err := place(c, p.X, p.Y)
		if err != nil {
			fs = append(fs, errorf("%s %v — it is not on the grid and cannot be run", label(i), err))
			continue
		}
		sq := [2]int{row, col}
		if j, taken := first[sq]; taken {
			fs = append(fs, errorf("%s and %s share cell %c%d — only the last one placed is visible", label(i), label(j), 'A'+col, row))
		} else {
			first[sq] = i
		}
	}
	return fs
}

// ParseAddress splits "[profile:]COLROW[.ITEM]" into its parts. The profile
// qualifier may be empty (active profile); the address is normalized to
// upper case; item is 1-based, 0 when absent.
func ParseAddress(ref string) (name, addr string, item int, err error) {
	rest := ref
	if i := strings.IndexByte(ref, ':'); i >= 0 {
		name = ref[:i]
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

	return name, strings.ToUpper(cell), item, nil
}

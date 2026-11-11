package profile

import (
	"errors"
	"fmt"
	"log"
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
	row, col, err := resolveCell(*c, p.X, p.Y)
	if err != nil {
		return nil
	}
	for j := len(p.Commands) - 1; j >= 0 && &p.Commands[j] != c; j-- {
		if r, cl, err := resolveCell(p.Commands[j], p.X, p.Y); err == nil && r == row && cl == col {
			return &p.Commands[j]
		}
	}
	return nil
}

// resolveCell resolves a command's grid position against the grid dimensions
// (x columns, y rows). A row or column of -1 (column 'z') means "last" and is
// resolved here to y-1 / x-1. It errors only on an unparseable column letter;
// bounds-checking against the grid is left to the caller.
func resolveCell(cmd Command, x, y int) (row, col int, err error) {
	col, err = letterToColumn(cmd.Col)
	if err != nil {
		return 0, 0, err
	}
	row = cmd.Row
	if row == -1 {
		row = y - 1
	}
	if col == -1 {
		col = x - 1
	}
	return row, col, nil
}

// ValidateGrid checks that every command fits a grid of x columns and y rows.
// It returns an error for the first command that is out of bounds.
func ValidateGrid(x, y int, cmds []Command) error {
	for _, cmd := range cmds {
		row, col, err := resolveCell(cmd, x, y)
		if err != nil {
			return fmt.Errorf("command %q has invalid column %q: %v", cmd.Name, cmd.Col, err)
		}
		if row < 0 {
			return fmt.Errorf("command %q has invalid row %d (rows start at 0; -1 is the last)", cmd.Name, cmd.Row)
		}
		if row >= y {
			return fmt.Errorf("command %q needs y >= %d (have %d)", cmd.Name, row+1, y)
		}
		if col >= x {
			return fmt.Errorf("command %q needs x >= %d (have %d)", cmd.Name, col+1, x)
		}
	}
	return nil
}

// BuildGrid places command names into a y-by-x grid of cells.
func BuildGrid(x, y int, cmds []Command) [][]string {
	grid := make([][]string, y)
	for i := range grid {
		grid[i] = make([]string, x)
	}
	for _, cmd := range cmds {
		row, col, err := resolveCell(cmd, x, y)
		if err != nil {
			// ValidateGrid already reports this at load time; rendering
			// must never kill the process, so skip the cell.
			log.Printf("skipping command %q: %v", cmd.Name, err)
			continue
		}
		if row >= 0 && row < y && col >= 0 && col < x {
			grid[row][col] = cmd.Name
		}
	}
	return grid
}

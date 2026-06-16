package config

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

// resolveCell resolves a command's grid position against the grid dimensions.
// A row or column of -1 (column 'z') means "last" and is resolved here to
// Y-1 / X-1. It errors only on an unparseable column letter; bounds-checking
// against the grid is left to the caller.
func resolveCell(cmd Command, cfg Config) (row, col int, err error) {
	col, err = letterToColumn(cmd.Col)
	if err != nil {
		return 0, 0, err
	}
	row = cmd.Row
	if row == -1 {
		row = cfg.Y - 1
	}
	if col == -1 {
		col = cfg.X - 1
	}
	return row, col, nil
}

// ValidateConfig checks if the configuration is logically valid.
// It returns an error if any command is out of bounds for the grid size.
func ValidateConfig(cfg Config) error {
	for _, cmd := range cfg.Commands {
		row, col, err := resolveCell(cmd, cfg)
		if err != nil {
			return fmt.Errorf("command %q has invalid column %q: %v", cmd.Name, cmd.Col, err)
		}
		if row >= cfg.Y {
			return fmt.Errorf("command %q needs y >= %d (have %d)", cmd.Name, row+1, cfg.Y)
		}
		if col >= cfg.X {
			return fmt.Errorf("command %q needs x >= %d (have %d)", cmd.Name, col+1, cfg.X)
		}
	}
	return nil
}

// BuildGrid places command names into a Y-by-X grid of cells.
func BuildGrid(config Config) [][]string {
	grid := make([][]string, config.Y)
	for i := range grid {
		grid[i] = make([]string, config.X)
	}
	for _, cmd := range config.Commands {
		row, col, err := resolveCell(cmd, config)
		if err != nil {
			fatalf("invalid column value for command %q: %v", cmd.Name, err)
		}
		if row >= 0 && row < config.Y && col >= 0 && col < config.X {
			grid[row][col] = cmd.Name
		}
	}
	return grid
}

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

// ValidateConfig checks if the configuration is logically valid.
// It returns an error if any command is out of bounds for the grid size.
func ValidateConfig(cfg Config) error {
	for _, cmd := range cfg.Commands {
		row := cmd.Row
		col, err := letterToColumn(cmd.Col)
		if err != nil {
			return fmt.Errorf("command %q has invalid column %q: %v", cmd.Name, cmd.Col, err)
		}

		// -1 means last row/column
		if row == -1 {
			row = cfg.Y - 1
		}
		if col == -1 {
			col = cfg.X - 1
		}

		if row >= cfg.Y {
			return fmt.Errorf("command %q at row %d exceeds grid height %d", cmd.Name, row, cfg.Y)
		}
		if col >= cfg.X {
			return fmt.Errorf("command %q at column %q exceeds grid width %d", cmd.Name, cmd.Col, cfg.X)
		}
	}
	return nil
}

// BuildGrid places command names into a Y-by-X grid of cells.
func BuildGrid(config Config) [][]string {
	ClampConfig(&config)
	grid := make([][]string, config.Y)
	for i := range grid {
		grid[i] = make([]string, config.X)
	}
	for _, cmd := range config.Commands {
		row := cmd.Row
		col, err := letterToColumn(cmd.Col)
		if err != nil {
			fatalf("invalid column value for command %q: %v", cmd.Name, err)
		}

		if row == -1 {
			row = config.Y - 1
		}
		if col == -1 {
			col = config.X - 1
		}
		if row >= 0 && row < config.Y && col >= 0 && col < config.X {
			grid[row][col] = cmd.Name
		}
	}
	return grid
}

package ui

// findLastPopulatedCol returns the index of the last column that has any content in any row.
func findLastPopulatedCol(grid [][]string) int {
	lastCol := -1
	if len(grid) == 0 {
		return lastCol
	}
	for r := 0; r < len(grid); r++ {
		for c := 0; c < len(grid[r]); c++ {
			if grid[r][c] != "" && c > lastCol {
				lastCol = c
			}
		}
	}
	return lastCol
}

// findLastPopulatedRow returns the index of the last row in the specified column that has content.
func findLastPopulatedRow(grid [][]string, col int) int {
	lastRow := -1
	if len(grid) == 0 || col < 0 {
		return lastRow
	}
	for r := 0; r < len(grid); r++ {
		if col < len(grid[r]) {
			if grid[r][col] != "" {
				lastRow = r
			}
		}
	}
	return lastRow
}

// findFirstPopulatedRow returns the index of the first row in the specified column that has content.
func findFirstPopulatedRow(grid [][]string, col int) int {
	if len(grid) == 0 || col < 0 {
		return 0
	}
	for r := 0; r < len(grid); r++ {
		if col < len(grid[r]) {
			if grid[r][col] != "" {
				return r
			}
		}
	}
	return 0 // Fallback
}

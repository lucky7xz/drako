package ui

// Directions for the lock pump mechanism
const (
	directionNone  = 0
	directionLeft  = -1
	directionRight = 1
)

// pumpDirectionForKey returns the direction associated with a given key press.
// Returns directionLeft for 'left', 'h', 'a'.
// Returns directionRight for 'right', 'l', 'd'.
// Returns directionNone (0) for any other key.
func pumpDirectionForKey(key string) int {
	switch key {
	case "left", "h", "a":
		return directionLeft
	case "right", "l", "d":
		return directionRight
	default:
		return directionNone
	}
}

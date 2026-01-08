package core

// Directions for the lock pump mechanism
const (
	DirectionNone  = 0
	DirectionLeft  = -1
	DirectionRight = 1
)

// PumpDirectionForKey returns the direction associated with a given key press.
// Returns DirectionLeft for 'left', 'h', 'a'.
// Returns DirectionRight for 'right', 'l', 'd'.
// Returns DirectionNone (0) for any other key.
func PumpDirectionForKey(key string) int {
	switch key {
	case "left", "h", "a":
		return DirectionLeft
	case "right", "l", "d":
		return DirectionRight
	default:
		return DirectionNone
	}
}

package core

// CalculateNextProfileIndex determines the next profile index based on direction and wrapping.
// direction should be 1 (next) or -1 (previous).
func CalculateNextProfileIndex(currentIndex, direction, totalProfiles int) int {
	if totalProfiles <= 0 {
		return 0
	}
	next := (currentIndex + direction) % totalProfiles
	if next < 0 {
		next += totalProfiles
	}
	return next
}

package core

import "github.com/muesli/termenv"

// ColorProfileName gives a short, human name for a termenv color profile.
func ColorProfileName(p termenv.Profile) string {
	switch p {
	case termenv.TrueColor:
		return "truecolor"
	case termenv.ANSI256:
		return "256-color"
	case termenv.ANSI:
		return "16-color"
	default:
		return "no-color"
	}
}

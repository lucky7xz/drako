// Package glyph holds drako's rules for text it puts on the screen that
// someone else wrote: deck labels, and file and folder names. Such text is
// data. It may not steer the terminal (Check), and drako draws it only in a
// form every terminal gives the same width (Clean).
package glyph

import (
	"fmt"
	"strings"
	"unicode"
)

// Check reports the first character in s that would steer the terminal: a
// control character other than line break and tab (ESC starts the terminal's
// own commands), or a mark that reverses the direction of the text around it.
func Check(s string) error {
	for _, r := range s {
		if steers(r) {
			return fmt.Errorf("contains %U, a character that steers the terminal", r)
		}
	}
	return nil
}

// CheckName reports why name cannot be a name drako draws and you type, such
// as a deck's file name: it steers the terminal (Check), or it holds an
// invisible character drako would leave out when drawing it, so what you see
// is not what you would have to type.
func CheckName(name string) error {
	if err := Check(name); err != nil {
		return err
	}
	for _, r := range name {
		if invisible(r) {
			return fmt.Errorf("contains %U, which drako does not draw as written", r)
		}
	}
	return nil
}

// Clean is s as drako draws it. A character that would steer the terminal
// becomes U+FFFD. The invisible characters terminals disagree about are
// dropped: joiners, variation selectors and other format characters. A
// joined emoji (woman, joiner, school) comes out as its two parts, which
// every terminal draws at the same width.
func Clean(s string) string {
	if strings.IndexFunc(s, func(r rune) bool { return steers(r) || invisible(r) }) < 0 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case steers(r):
			b.WriteRune(unicode.ReplacementChar)
		case invisible(r):
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func steers(r rune) bool {
	if r == '\n' || r == '\t' {
		return false
	}
	return unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r)
}

func invisible(r rune) bool {
	return unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Variation_Selector, r)
}

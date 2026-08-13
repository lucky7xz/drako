package ui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// Fixed dracula-like palette for shell highlighting in the explain overlay.
// Only foreground colors are set; the popup background is preserved by the
// caller-supplied base style.
const (
	hlComment  = lipgloss.Color("#6272a4")
	hlString   = lipgloss.Color("#50fa7b")
	hlKeyword  = lipgloss.Color("#ff79c6")
	hlVariable = lipgloss.Color("#8be9fd")
	hlNumber   = lipgloss.Color("#ffb86c")
)

// shellKeywords are the shell control-flow words we color. Builtins are left
// alone on purpose, to keep the coloring quiet.
var shellKeywords = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"for": true, "while": true, "until": true, "do": true, "done": true,
	"case": true, "esac": true, "in": true, "function": true, "select": true,
}

// highlightShell renders one already-wrapped script line with shell syntax
// coloring. It mirrors valueStyle (Item.Background, which is Padding(0,1)) by
// framing the content in a single background space on each side, so the result
// has the same display width as valueStyle.Render(line): the scrollbar math in
// viewInfoMode relies on that. Every input rune is emitted exactly once.
func highlightShell(line string, bg lipgloss.TerminalColor) string {
	base := lipgloss.NewStyle().Background(bg)

	var b strings.Builder
	b.WriteString(base.Render(" ")) // left padding (matches Item's Padding(0,1))

	emit := func(s string, col lipgloss.TerminalColor) {
		if col == nil {
			b.WriteString(base.Render(s))
		} else {
			b.WriteString(base.Foreground(col).Render(s))
		}
	}

	r := []rune(line)
	n := len(r)
	i := 0
	for i < n {
		c := r[i]

		switch {
		// Comment: '#' at line start or after whitespace runs to end of line.
		case c == '#' && (i == 0 || isSpace(r[i-1])):
			emit(string(r[i:]), hlComment)
			i = n

		// Single-quoted string: no escapes inside single quotes.
		case c == '\'':
			j := i + 1
			for j < n && r[j] != '\'' {
				j++
			}
			if j < n {
				j++ // include the closing quote
			}
			emit(string(r[i:j]), hlString)
			i = j

		// Double-quoted string: honor \" escapes; the whole run is one span.
		case c == '"':
			j := i + 1
			for j < n {
				if r[j] == '\\' && j+1 < n {
					j += 2
					continue
				}
				if r[j] == '"' {
					j++
					break
				}
				j++
			}
			emit(string(r[i:j]), hlString)
			i = j

		// Variable / command substitution.
		case c == '$' && i+1 < n:
			nx := r[i+1]
			switch {
			case nx == '{':
				j := i + 2
				for j < n && r[j] != '}' {
					j++
				}
				if j < n {
					j++ // include '}'
				}
				emit(string(r[i:j]), hlVariable)
				i = j
			case nx == '(':
				j := i + 1
				depth := 0
				for j < n {
					if r[j] == '(' {
						depth++
					} else if r[j] == ')' {
						depth--
						if depth == 0 {
							j++
							break
						}
					}
					j++
				}
				emit(string(r[i:j]), hlVariable)
				i = j
			case isSpecialVar(nx):
				emit(string(r[i:i+2]), hlVariable)
				i += 2
			case isVarStart(nx):
				j := i + 1
				for j < n && isVarChar(r[j]) {
					j++
				}
				emit(string(r[i:j]), hlVariable)
				i = j
			default:
				emit(string(c), nil)
				i++
			}

		// Word: letters/underscore, extending over digits. Keyword or default.
		case isWordStart(c):
			j := i
			for j < n && isWordChar(r[j]) {
				j++
			}
			w := string(r[i:j])
			if shellKeywords[w] {
				emit(w, hlKeyword)
			} else {
				emit(w, nil)
			}
			i = j

		// Standalone number.
		case unicode.IsDigit(c):
			j := i
			for j < n && (unicode.IsDigit(r[j]) || r[j] == '.') {
				j++
			}
			emit(string(r[i:j]), hlNumber)
			i = j

		default:
			emit(string(c), nil)
			i++
		}
	}

	b.WriteString(base.Render(" ")) // right padding
	return b.String()
}

func isSpace(c rune) bool     { return c == ' ' || c == '\t' }
func isWordStart(c rune) bool { return unicode.IsLetter(c) || c == '_' }
func isWordChar(c rune) bool  { return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' }
func isVarStart(c rune) bool  { return unicode.IsLetter(c) || c == '_' }
func isVarChar(c rune) bool   { return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' }

// isSpecialVar reports whether c is a one-character special shell parameter
// ($?, $#, $@, $*, $$, $!, $0-$9).
func isSpecialVar(c rune) bool {
	switch c {
	case '?', '#', '@', '*', '$', '!':
		return true
	}
	return c >= '0' && c <= '9'
}

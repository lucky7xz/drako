package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// WrapWord splits a word into chunks that fit within the given width.
func WrapWord(word string, width int) []string {
	if width <= 0 {
		return []string{word}
	}
	var out []string
	var b strings.Builder
	cur := 0
	for _, r := range word {
		w := lipgloss.Width(string(r))
		if cur+w > width && b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
			cur = 0
		}
		b.WriteRune(r)
		cur += w
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// WrapLine wraps a single line of text to the specified width, preserving words where possible.
func WrapLine(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return []string{""}
	}
	var lines []string
	var line string
	for _, word := range fields {
		ww := lipgloss.Width(word)
		if line == "" {
			if ww <= width {
				line = word
			} else {
				lines = append(lines, WrapWord(word, width)...)
				line = ""
			}
			continue
		}
		if lipgloss.Width(line)+1+ww <= width {
			line += " " + word
		} else {
			lines = append(lines, line)
			if ww <= width {
				line = word
			} else {
				lines = append(lines, WrapWord(word, width)...)
				line = ""
			}
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// WrapText wraps a multi-line string to the specified width.
// It handles existing newlines by processing each paragraph separately.
func WrapText(text string, width int) []string {
	var result []string
	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			result = append(result, "") // Preserve empty lines
			continue
		}
		result = append(result, WrapLine(para, width)...)
	}
	return result
}


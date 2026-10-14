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

// WrapScript wraps a command for display without reflowing it: indentation
// and inner spacing survive, and only the space a line breaks at is dropped.
func WrapScript(text string, width int) []string {
	var result []string
	for _, line := range strings.Split(text, "\n") {
		result = append(result, wrapScriptLine(line, width)...)
	}
	return result
}

func wrapScriptLine(line string, width int) []string {
	if width <= 0 || lipgloss.Width(line) <= width {
		return []string{line}
	}
	var lines []string
	cur := ""
	for _, tok := range splitSpaceRuns(line) {
		if lipgloss.Width(cur)+lipgloss.Width(tok) <= width {
			cur += tok
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
		}
		cur = ""
		if strings.TrimSpace(tok) == "" {
			continue
		}
		chunks := WrapWord(tok, width)
		lines = append(lines, chunks[:len(chunks)-1]...)
		cur = chunks[len(chunks)-1]
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// splitSpaceRuns cuts s into alternating runs of spaces and non-spaces.
func splitSpaceRuns(s string) []string {
	var toks []string
	start := 0
	for i := 1; i <= len(s); i++ {
		if i == len(s) || (s[i] == ' ') != (s[i-1] == ' ') {
			toks = append(toks, s[start:i])
			start = i
		}
	}
	return toks
}

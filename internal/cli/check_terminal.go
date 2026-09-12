package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/muesli/termenv"
)

// HandleCheckTerminalCommand processes 'drako check-terminal': reports the
// color profile drako will render with.
func HandleCheckTerminalCommand(args []string) int {
	profile := lipgloss.ColorProfile()

	fmt.Printf("Detected color profile: %s\n", core.ColorProfileName(profile))
	if profile == termenv.TrueColor {
		fmt.Println("✓ Full color — drako's theme colors render as authored.")
	} else {
		fmt.Println("⚠ Not truecolor — drako's hex theme colors are being")
		fmt.Println("  approximated to the nearest color this terminal supports.")
	}
	fmt.Printf("\n  TERM=%s\n", envOrUnset("TERM"))
	fmt.Printf("  COLORTERM=%s\n", envOrUnset("COLORTERM"))
	return 0
}

func envOrUnset(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return "(not set)"
}

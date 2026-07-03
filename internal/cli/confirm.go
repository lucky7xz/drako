package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RealUI implements UIInterface using stdin/stdout
type RealUI struct{}

func (ui *RealUI) Confirm(prompt string) bool {
	return ConfirmAction(prompt)
}

// ConfirmAction prompts the user to confirm an action
func ConfirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

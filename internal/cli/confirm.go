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

// confirmReader is shared across prompts: a fresh bufio.Reader per call
// would buffer-and-discard piped input, so only the first of several
// confirmations could ever be answered non-interactively.
var confirmReader = bufio.NewReader(os.Stdin)

// ConfirmAction prompts the user to confirm an action
func ConfirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	response, err := confirmReader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

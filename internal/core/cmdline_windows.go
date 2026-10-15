package core

import (
	"os/exec"
	"syscall"
)

// passCmdExeLine hands cmd.exe its command verbatim: it doesn't parse the
// quoting exec.Command applies, and /S strips exactly the outer quotes.
func passCmdExeLine(cmd *exec.Cmd, argv []string) {
	if len(argv) == 3 && argv[0] == "cmd" && argv[1] == "/C" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /S /C "` + argv[2] + `"`}
	}
}

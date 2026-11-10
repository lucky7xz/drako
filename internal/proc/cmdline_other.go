//go:build !windows

package proc

import "os/exec"

// passCmdExeLine is a no-op: only Windows has cmd.exe's quoting rules.
func passCmdExeLine(*exec.Cmd, []string) {}

//go:build !unix

package qwencode

import (
	"os"
	"os/exec"
	"syscall"
)

func configureCommandProcessGroup(cmd *exec.Cmd) {
}

func signalCommandProcessTree(process *os.Process, sig syscall.Signal) error {
	if process == nil {
		return nil
	}
	return process.Signal(sig)
}

func killCommandProcessTree(process *os.Process) error {
	if process == nil {
		return nil
	}
	return process.Kill()
}

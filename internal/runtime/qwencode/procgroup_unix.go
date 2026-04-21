//go:build unix

package qwencode

import (
	"os"
	"os/exec"
	"syscall"
)

func configureCommandProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalCommandProcessTree(process *os.Process, sig syscall.Signal) error {
	if process == nil || process.Pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-process.Pid, sig); err == nil {
		return nil
	}
	return process.Signal(sig)
}

func killCommandProcessTree(process *os.Process) error {
	if process == nil || process.Pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return process.Kill()
}

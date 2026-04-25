//go:build !unix

package providercommon

import (
	"os"
	"os/exec"
)

func configureCommandProcessGroup(cmd *exec.Cmd) {
}

func killCommandProcessTree(process *os.Process) error {
	if process == nil {
		return nil
	}
	return process.Kill()
}

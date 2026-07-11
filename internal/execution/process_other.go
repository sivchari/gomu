//go:build !unix

package execution

import "os/exec"

func newProcessGroupCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

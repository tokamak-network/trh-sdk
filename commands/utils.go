package commands

import (
	"os/exec"
	"strings"
)

func executeCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

package utils

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// ExecuteCommandStream executes a command and streams its output in real-time
func ExecuteCommandStream(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the command execution
	if err := cmd.Start(); err != nil {
		return err
	}

	// Stream stdout and stderr concurrently
	go streamOutput(stdout)
	go streamOutput(stderr)

	// Wait for the command to complete
	return cmd.Wait()
}

// streamOutput reads and prints the command output line by line
func streamOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text()) // Print each line in real-time
	}
}

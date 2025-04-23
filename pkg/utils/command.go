package utils

import (
	"bufio"
	"io"
	"os/exec"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
)

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	logging.Info(strings.TrimSpace(string(output)))
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

	// Create channels to signal goroutine completion
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})

	// Stream stdout concurrently
	go func() {
		streamOutput(stdout)
		close(stdoutDone)
	}()

	// Stream stderr concurrently
	go func() {
		streamOutput(stderr)
		close(stderrDone)
	}()

	// Wait for both streamOutput goroutines to finish
	<-stdoutDone
	<-stderrDone

	// Wait for the command to complete
	return cmd.Wait()
}

// streamOutput reads and prints the command output line by line
func streamOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		logging.Info(scanner.Text()) // Print each line in real-time
	}
}

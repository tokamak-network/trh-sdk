package utils

import (
	"bufio"
	"io"
	"os/exec"
	"strings"

	"github.com/creack/pty"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
)

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()

	trimmedOutput := strings.TrimSpace(string(output))
	logging.Info(trimmedOutput)

	return trimmedOutput, err
}

func ExecuteCommandStream(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	// Start the command with a pseudo-terminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		logging.Errorf("Failed to start command: %v", err)
		return err
	}
	defer ptmx.Close()

	err = streamOutput(ptmx)
	if err != nil {
		return err
	}
	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		logging.Errorf("Command execution failed: %v", err)
		return err
	}

	// Check the exit code
	if cmd.ProcessState.ExitCode() != 0 {
		logging.Errorf("Command exited with non-zero status: %d", cmd.ProcessState.ExitCode())
		return err
	}

	return nil
}

// streamOutput reads and prints the command output line by line
func streamOutput(pipe io.ReadCloser) error {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		logging.Info(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		logging.Errorf("Reading scanner output failed: %v", err)
		return err
	}

	return nil
}

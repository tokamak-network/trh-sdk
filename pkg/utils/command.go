package utils

import (
	"bufio"
	"fmt"
	"os/exec"
)

//	func ExecuteCommand(command string, args ...string) (string, error) {
//		cmd := exec.Command(command, args...)
//		output, err := cmd.CombinedOutput()
//		return strings.TrimSpace(string(output)), err
//	}
func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	// Get the stdout and stderr streams
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("STDERR:", scanner.Text())
		}
	}()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return "", nil
}

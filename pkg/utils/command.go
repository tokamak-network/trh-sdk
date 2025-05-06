package utils

import (
	"bufio"
	"fmt"
	"github.com/creack/pty"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
)

var (
	ErrorPseudoTerminalExist = "read /dev/ptmx: input/output error"
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

	var (
		scanner *bufio.Scanner
		ptmx    *os.File
		err     error
		wg      sync.WaitGroup
	)

	// Try to start with PTY
	ptmx, err = pty.Start(cmd)
	if err == nil {
		scanner = bufio.NewScanner(ptmx)
		defer func() {
			wg.Wait()
			_ = ptmx.Close()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := streamOutput(scanner); err != nil {
				logging.Errorf("Error streaming PTY output: %v", err)
			}
		}()
	} else {
		// Fallback to stdout pipe
		var rd io.ReadCloser
		rd, err = cmd.StdoutPipe()
		if err != nil {
			logging.Errorf("Error creating StdoutPipe: %v", err)
			return err
		}
		scanner = bufio.NewScanner(rd)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := streamOutput(scanner); err != nil {
				logging.Errorf("Error streaming output: %v", err)
			}
		}()
	}

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		logging.Errorf("Command execution failed: %v", err)
		return err
	}

	// Wait for stream to finish
	wg.Wait()

	// Check exit code
	if cmd.ProcessState.ExitCode() != 0 {
		logging.Errorf("Command exited with non-zero status: %d", cmd.ProcessState.ExitCode())
		return fmt.Errorf("command exited with status: %d", cmd.ProcessState.ExitCode())
	}

	return nil
}

// streamOutput reads and prints the command output line by line
func streamOutput(scanner *bufio.Scanner) error {
	for scanner.Scan() {
		logging.Info(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		// when using github.com/creack/pty,
		// it's common to encounter an input/output error (/dev/ptmx: input/output error) when the process running in the pseudo-terminal (PTY) exits.
		// This happens because once the subprocess is done and the PTY master has no more data to read,
		// Read() returns an I/O error instead of the expected EOF. This is behavior by design in some Linux versions and kernels.
		if !(ErrorPseudoTerminalExist == err.Error()) {
			return err
		}
	}
	return nil
}

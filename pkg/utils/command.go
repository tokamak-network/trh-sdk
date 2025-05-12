package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
)

var (
	ErrorPseudoTerminalExist = "read /dev/ptmx: input/output error"
)

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()

	trimmedOutput := strings.TrimSpace(string(output))

	return trimmedOutput, err
}

func ExecuteCommandStream(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	var (
		ptmx *os.File
		err  error
		wg   sync.WaitGroup
		r    io.Reader
	)

	// Try to start with PTY
	ptmx, err = pty.Start(cmd)
	if err == nil {
		r = ptmx
		defer func() {
			wg.Wait()
			_ = ptmx.Close()
		}()
	} else {
		// Fallback to StdoutPipe
		var rd io.ReadCloser
		rd, err = cmd.StdoutPipe()
		if err != nil {
			logging.Errorf("Error creating StdoutPipe: %v", err)
			return err
		}
		r = rd
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := streamOutput(r); err != nil {
			if err.Error() != ErrorPseudoTerminalExist {
				logging.Errorf("Error streaming output: %v", err)
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
		logging.Errorf("Command execution failed: %v", err)
		return err
	}

	wg.Wait()

	// Check exit code
	if cmd.ProcessState.ExitCode() != 0 {
		logging.Errorf("Command exited with non-zero status: %d", cmd.ProcessState.ExitCode())
		return fmt.Errorf("command exited with status: %d", cmd.ProcessState.ExitCode())
	}

	return nil
}

// streamOutput reads and prints the command output line by line
func streamOutput(r io.Reader) error {
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n') // or use a custom delimiter
		if line != "" {
			logging.Info(strings.TrimSuffix(line, "\n"))
		}
		if err != nil {
			if err == io.EOF || err.Error() == "EOF" {
				return nil
			}
			return err
		}
	}
}

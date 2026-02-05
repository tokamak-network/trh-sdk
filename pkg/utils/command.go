package utils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"go.uber.org/zap"
)

var (
	ErrorPseudoTerminalExist = "read /dev/ptmx: input/output error"
)

func ExecuteCommand(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()

	trimmedOutput := strings.TrimSpace(string(output))

	// Handle cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	return trimmedOutput, err
}

// ExecuteCommandInDir executes a command in a specific directory.
// This avoids shell injection vulnerabilities by not using "bash -c" with string interpolation.
func ExecuteCommandInDir(ctx context.Context, dir string, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	trimmedOutput := strings.TrimSpace(string(output))

	// Handle cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	return trimmedOutput, err
}

func ExecuteCommandStream(ctx context.Context, l *zap.SugaredLogger, command string, args ...string) error {
	return ExecuteCommandStreamInDir(ctx, l, "", command, args...)
}

// ExecuteCommandStreamInDir executes a command in a specific directory with streaming output.
// This avoids shell injection vulnerabilities by not using "bash -c" with string interpolation.
func ExecuteCommandStreamInDir(ctx context.Context, l *zap.SugaredLogger, dir string, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var (
		ptmx    *os.File
		err     error
		wg      sync.WaitGroup
		r       io.Reader
		started bool
	)

	// Try to start with PTY
	ptmx, err = pty.Start(cmd)
	if err == nil {
		r = ptmx
		started = true // command is already started by pty.Start
		defer func() {
			wg.Wait()
			_ = ptmx.Close()
		}()
	} else {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Fallback to StdoutPipe
		var rd io.ReadCloser
		rd, err = cmd.StdoutPipe()
		if err != nil {
			l.Errorf("Error creating StdoutPipe: %v", err)
			return err
		}
		r = rd
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := streamOutput(r, l); err != nil {
			if err.Error() != ErrorPseudoTerminalExist {
				l.Errorf("Error streaming output: %v", err)
			}
		}
	}()

	if !started {
		// Only start if not already started by pty.Start
		if err := cmd.Start(); err != nil {
			return err
		}
	}

	err = cmd.Wait()
	wg.Wait()

	if ctx.Err() != nil {
		l.Info("Command was cancelled via context")
		return ctx.Err()
	}

	if err != nil {
		l.Errorf("Command execution failed: %v", err)
		return err
	}

	if cmd.ProcessState.ExitCode() != 0 {
		l.Errorf("Command exited with non-zero status: %d", cmd.ProcessState.ExitCode())
		return fmt.Errorf("command exited with status: %d", cmd.ProcessState.ExitCode())
	}

	return nil
}

// streamOutput reads and prints the command output line by line
func streamOutput(r io.Reader, l *zap.SugaredLogger) error {
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n') // or use a custom delimiter
		if line != "" {
			l.Info(strings.TrimSuffix(line, "\n"))
		}
		if err != nil {
			if err == io.EOF || err.Error() == "EOF" {
				return nil
			}
			return err
		}
	}
}

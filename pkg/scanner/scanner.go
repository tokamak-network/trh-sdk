package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// ReadPassword reads a secret string from stdin without echoing characters to the terminal.
func ReadPassword() (string, error) {
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println() // restore newline suppressed by ReadPassword
	return strings.TrimSpace(string(raw)), nil
}

// stdinReader is a shared reader to avoid losing buffered data between calls.
var stdinReader = bufio.NewReader(os.Stdin)

func ScanBool(defaultResponse bool) (bool, error) {
	input, err := stdinReader.ReadString('\n')
	if err != nil {
		// Allow EOF on empty line to mean default
		if len(strings.TrimSpace(input)) == 0 {
			return defaultResponse, nil
		}
		return false, err
	}

	response := strings.TrimSpace(input)

	// Blank input, use default
	if response == "" {
		return defaultResponse, nil
	}

	switch strings.ToLower(response) {
	case "y":
		return true, nil
	case "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input: must be 'y' or 'n'")
	}
}

func ScanString() (string, error) {
	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil

}

func ScanFloat() (float64, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}

	input = strings.TrimSpace(input)

	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value: %v", err)
	}

	return value, nil
}

func ScanInt() (int, error) {
	input, err := ScanString()
	if err != nil {
		return 0, err
	}

	if input == "" {
		return 0, nil
	}

	var num int
	_, err = fmt.Sscanf(input, "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("invalid input: %s", input)
	}

	return num, nil
}

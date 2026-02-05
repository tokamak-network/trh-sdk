package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func ScanBool(defaultResponse bool) (bool, error) {
	var response string
	n, err := fmt.Scanln(&response)

	// Blank input, default No
	if n == 0 {
		return defaultResponse, nil
	}

	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y":
		return true, nil
	case "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input: must be 'y' or 'n'")
	}
}

func ScanString() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
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

// ScanPassword reads a password/secret from stdin with masked input (no echo)
func ScanPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Print newline after password input
	return strings.TrimSpace(string(bytePassword)), nil
}

// ScanPasswordWithConfirmation reads a password with confirmation (both masked)
func ScanPasswordWithConfirmation() (string, error) {
	for {
		fmt.Print("Enter password: ")
		password, err := ScanPassword()
		if err != nil {
			return "", err
		}

		if password == "" {
			fmt.Println("Password cannot be empty. Please try again.")
			continue
		}

		fmt.Print("Confirm password: ")
		confirmPassword, err := ScanPassword()
		if err != nil {
			return "", err
		}

		if password != confirmPassword {
			fmt.Println("Passwords do not match. Please try again.")
			continue
		}

		return password, nil
	}
}

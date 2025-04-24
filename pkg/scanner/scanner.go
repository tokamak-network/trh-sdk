package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ScanBool(defaultResponse bool) (bool, error) {
	var response string
	n, err := fmt.Scanln(&response)
	// Blank input, default No
	if n == 0 {
		return defaultResponse, nil
	}

	if strings.ToLower(response) != "n" && strings.ToLower(response) != "y" {
		return false, fmt.Errorf("invalid input")
	}
	if err != nil {
		return false, err
	}

	if strings.ToLower(response) == "y" {
		return true, nil
	}
	return defaultResponse, nil
}

func ScanString() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil

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

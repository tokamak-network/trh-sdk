package scanner

import (
	"fmt"
	"strings"
)

func ScanBool() (bool, error) {
	var response string
	n, err := fmt.Scanln(&response)
	// Blank input, default No
	if n == 0 {
		return false, nil
	}

	if strings.ToLower(response) != "n" && strings.ToLower(response) != "y" {
		return false, fmt.Errorf("Invalid input")
	}
	if err != nil {
		return false, err
	}

	if strings.ToLower(response) == "y" {
		return true, nil
	}
	return false, nil
}

func ScanString() (string, error) {
	var response string
	n, err := fmt.Scanln(&response)
	// Blank input, default No
	if n == 0 {
		return "", nil
	}

	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil

}

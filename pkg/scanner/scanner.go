package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ScanBool() (bool, error) {
	var response string
	n, err := fmt.Scanln(&response)

	if err != nil {
		return false, err
	}

	// Blank input, default No
	if n == 0 {
		return false, nil
	}

	if strings.ToLower(response) != "n" && strings.ToLower(response) != "y" {
		return false, fmt.Errorf("Invalid input")
	}

	if strings.ToLower(response) == "y" {
		return true, nil
	}
	return false, nil
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

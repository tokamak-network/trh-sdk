package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
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

	if strings.ToLower(response) != "n" && strings.ToLower(response) != "y" {
		return false, fmt.Errorf("invalid input")
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

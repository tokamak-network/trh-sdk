package utils

import (
	"fmt"
	"os/exec"
	"regexp"
)

func GetGoVersion() (string, error) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return "", err
	}

	versionRegex := regexp.MustCompile(`go(\d+\.\d+\.\d+)`)
	matches := versionRegex.FindStringSubmatch(string(output))

	if len(matches) < 2 {
		fmt.Println("Could not determine Go version.")
		return "", fmt.Errorf("could not determine Go version.")
	}

	currentVersion := matches[1]
	return currentVersion, nil
}

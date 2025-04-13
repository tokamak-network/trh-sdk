package utils

import "os"

func GetShellConfigDefault() string {
	var shellConfigFile string
	if os.Getenv("SHELL") == "/bin/zsh" || os.Getenv("SHELL") == "/usr/bin/zsh" {
		shellConfigFile = "~/.zshrc"
	} else {
		shellConfigFile = "~/.bashrc"
	}
	return shellConfigFile
}

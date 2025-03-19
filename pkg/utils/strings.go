package utils

import "strings"

func ConvertToHyphen(input string) string {
	return strings.ToLower(strings.ReplaceAll(input, " ", "-"))
}

package utils

import "net/url"

// isValidURL checks if the given input is a valid URL
func isValidURL(input string) bool {
	parsedURL, err := url.ParseRequestURI(input)
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}

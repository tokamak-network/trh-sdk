package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var reservedKeywords = map[string]bool{
	"select": true, "insert": true, "delete": true, "update": true, "table": true, "user": true, "role": true,
	"database": true, "public": true, "grant": true, "revoke": true, "index": true, "view": true, "schema": true,
	"session_user": true, "current_user": true, "pg_signal_backend": true, "pg_execute_server_program": true,
	"pg_monitor": true, "pg_read_all_settings": true, "pg_read_all_stats": true, "pg_stat_scan_tables": true,
}

// Restricted PostgreSQL usernames
var restrictedUsernames = map[string]bool{
	"postgres": true, "admin": true, "root": true, "test": true, "guest": true, "anonymous": true, "demo": true,
}

var validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidatePostgresUsername(username string) error {
	username = strings.ToLower(username) // Convert to lowercase for case-insensitive comparison

	if len(username) < 1 || len(username) > 63 {
		return fmt.Errorf("username length must be between 1 and 63 characters")
	}

	if !validUsernameRegex.MatchString(username) {
		return fmt.Errorf("username contains invalid characters; only letters, numbers, and underscores are allowed")
	}

	if reservedKeywords[username] {
		return fmt.Errorf("username '%s' is a reserved SQL keyword", username)
	}

	if restrictedUsernames[username] {
		return fmt.Errorf("username '%s' is restricted and should not be used", username)
	}

	if strings.HasPrefix(username, "pg_") {
		return fmt.Errorf("username cannot start with 'pg_' as it is reserved for PostgreSQL system roles")
	}

	return nil // Username is valid
}

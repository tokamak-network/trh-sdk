package utils

import (
	"regexp"
	"strings"
)

var rdsUsernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,62}$`)

func IsValidRDSUsername(username string) bool {
	return rdsUsernameRegex.MatchString(username)
}

// IsValidRDSPassword validates RDS password for PostgreSQL according to AWS requirements:
// - Can include any printable ASCII character except /, ', ", @, or a space
// - Length: 8–128 characters
// Master password:
// The password for the database master user can include any printable ASCII character except /, ', ", @, or a space.
// For Oracle, & is an additional character limitation.
// The password can contain the following number of printable ASCII characters depending on the DB engine:
// - Db2: 8–255
// - MariaDB and MySQL: 8–41
// - Oracle: 8–30
// - SQL Server and PostgreSQL: 8–128
func IsValidRDSPassword(password string) bool {
	// PostgreSQL length requirements: 8–128
	const minLen, maxLen = 8, 128

	// Check length first
	if len(password) < minLen || len(password) > maxLen {
		return false
	}

	// Forbidden characters for PostgreSQL: space, ", ', /, @
	const forbiddenChars = " \"'/@"

	// Validate each character
	for _, char := range password {
		// Ensure it's printable ASCII (0x20-0x7E, inclusive)
		if char < 0x20 || char > 0x7E {
			return false
		}

		// Check if it's a forbidden character
		if strings.ContainsRune(forbiddenChars, char) {
			return false
		}
	}

	return true
}

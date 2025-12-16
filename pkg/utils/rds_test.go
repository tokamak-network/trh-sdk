package utils

import (
	"strings"
	"testing"
)

func TestIsValidRDSPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		// Valid passwords
		{
			name:     "valid 8 characters",
			password: "password",
			want:     true,
		},
		{
			name:     "valid with numbers",
			password: "password123",
			want:     true,
		},
		{
			name:     "valid with special characters",
			password: "Pass#123$Word",
			want:     true,
		},
		{
			name:     "valid with mixed case and symbols",
			password: "MyP!ss#W0rd$2024",
			want:     true,
		},
		{
			name:     "valid 128 characters",
			password: "aBc123!$%^&*()_+-=[]{}|;:,.<>?`~" + strings.Repeat("x", 95),
			want:     true,
		},
		{
			name:     "valid with underscore",
			password: "my_password_123",
			want:     true,
		},
		{
			name:     "valid with backtick",
			password: "pass`word123",
			want:     true,
		},
		{
			name:     "valid with brackets",
			password: "pass[word]123",
			want:     true,
		},
		{
			name:     "valid with braces",
			password: "pass{word}123",
			want:     true,
		},
		{
			name:     "valid with pipe",
			password: "pass|word123",
			want:     true,
		},
		{
			name:     "valid with tilde",
			password: "pass~word123",
			want:     true,
		},
		// Invalid passwords - too short
		{
			name:     "too short 7 characters",
			password: "passwor",
			want:     false,
		},
		{
			name:     "empty string",
			password: "",
			want:     false,
		},
		// Invalid passwords - too long
		{
			name:     "too long 129 characters",
			password: strings.Repeat("a", 129),
			want:     false,
		},
		// Invalid passwords - forbidden characters
		{
			name:     "contains space",
			password: "pass word123",
			want:     false,
		},
		{
			name:     "contains double quote",
			password: "pass\"word123",
			want:     false,
		},
		{
			name:     "contains single quote",
			password: "pass'word123",
			want:     false,
		},
		{
			name:     "contains forward slash",
			password: "pass/word123",
			want:     false,
		},
		{
			name:     "contains at sign",
			password: "pass@word123",
			want:     false,
		},
		{
			name:     "starts with forbidden character",
			password: "@password123",
			want:     false,
		},
		{
			name:     "ends with forbidden character",
			password: "password123@",
			want:     false,
		},
		// Invalid passwords - non-printable ASCII
		{
			name:     "contains newline",
			password: "pass\nword123",
			want:     false,
		},
		{
			name:     "contains tab",
			password: "pass\tword123",
			want:     false,
		},
		{
			name:     "contains null character",
			password: "pass\x00word123",
			want:     false,
		},
		{
			name:     "contains non-ASCII unicode",
			password: "passwörd123",
			want:     false,
		},
		{
			name:     "contains emoji",
			password: "pass😀word123",
			want:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsValidRDSPassword(test.password); got != test.want {
				t.Errorf("IsValidRDSPassword(%q) = %v, want %v", test.password, got, test.want)
			}
		})
	}
}

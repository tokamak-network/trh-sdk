package thanos

import "testing"

func TestChainNameRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid chain name with letters and numbers",
			input:    "Chain123",
			expected: true,
		},
		{
			name:     "Valid chain name with spaces",
			input:    "Chain Name",
			expected: true,
		},
		{
			name:     "Invalid chain name starting with a number",
			input:    "123Chain",
			expected: false,
		},
		{
			name:     "Invalid chain name with special characters",
			input:    "Chain@Name!",
			expected: false,
		},
		{
			name:     "Empty chain name",
			input:    "",
			expected: false,
		},
		{
			name:     "Valid chain name with uppercase letters",
			input:    "CHAIN",
			expected: true,
		},
		{
			name:     "Valid chain name with lowercase letters",
			input:    "chain",
			expected: true,
		},
		{
			name:     "Valid chain name with special characters",
			input:    "chain1-",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chainNameRegex.MatchString(tt.input)
			if result != tt.expected {
				t.Errorf("chainNameRegex.MatchString(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

package utils

import (
	"testing"
)

func TestCheckDirExists(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{"Exist file", "/home/tiennam/work/tokamak/trh-sdk/tokamak-thanos-stack/terraform/backend/../.envrc.example", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckFileExists(tt.path)
			if got != tt.expect {
				t.Errorf("checkDirExists(%s) = %v; want %v", tt.path, got, tt.expect)
			}
		})
	}
}

package handler

import "testing"

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"owner", true},
		{"admin", true},
		{"operator", true},
		{"viewer", true},
		{"", false},
		{"read", false},
		{"write", false},
		{"superadmin", false},
		{"root", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := isValidRole(tt.role)
			if got != tt.want {
				t.Errorf("isValidRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

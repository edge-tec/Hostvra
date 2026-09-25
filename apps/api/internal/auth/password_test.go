package auth

import "testing"

func TestValidatePasswordComplexity(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{"Valid password", "Hostvra#2026Secure", false},
		{"Too short", "Sh0rt!", true},
		{"No uppercase", "lowercase123!@#", true},
		{"No lowercase", "UPPERCASE123!@#", true},
		{"No number", "NoNumbersHere!@#", true},
		{"No special char", "NoSpecialChars123", true},
		{"Valid edge case", "A1!aaaaa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordComplexity(tt.password)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePasswordComplexity(%q) error = %v, expectError = %v", tt.password, err, tt.expectError)
			}
		})
	}
}

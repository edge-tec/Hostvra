package auth

import (
	"errors"
	"unicode"
)

var (
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters long")
	ErrPasswordNoUpper      = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower      = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber     = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial    = errors.New("password must contain at least one special character")
)

// ValidatePasswordComplexity validates password strength against standard enterprise criteria:
// - At least 8 characters
// - At least 1 uppercase letter
// - At least 1 lowercase letter
// - At least 1 digit
// - At least 1 special character / punctuation
func ValidatePasswordComplexity(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}

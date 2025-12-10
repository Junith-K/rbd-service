package utils

import (
	"errors"
	"regexp"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,16}$`)

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 16 {
		return errors.New("username must be between 3 and 16 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, and underscores")
	}
	return nil
}

// ValidatePassword validates password format
func ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}

package users

import (
	"net/mail"
	"strings"
)

func validateUpdateProfileInput(input UpdateProfileInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrInvalidName
	}

	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	return nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}

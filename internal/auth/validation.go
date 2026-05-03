package auth

import (
	"net/mail"
	"strings"
)

func validateFirebaseLoginInput(input FirebaseLoginInput) error {
	if strings.TrimSpace(input.IDToken) == "" {
		return ErrInvalidFirebaseToken
	}

	switch strings.TrimSpace(input.Platform) {
	case "android", "ios", "web":
		return nil
	default:
		return ErrInvalidFirebaseToken
	}
}

func validateEmailAvailabilityInput(input EmailAvailabilityInput) error {
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	return nil
}

func validateRefreshTokenInput(input RefreshTokenInput) error {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return ErrRefreshTokenRequired
	}

	return nil
}

func validateLogoutInput(input LogoutInput) error {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return ErrRefreshTokenRequired
	}

	return nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}

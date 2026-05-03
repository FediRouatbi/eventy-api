package auth

import "errors"

var (
	ErrInvalidEmail             = errors.New("email must be a valid email address")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrEmailAlreadyExists       = errors.New("email is already in use")
	ErrInvalidFirebaseToken     = errors.New("invalid firebase token")
	ErrFirebaseEmailNotVerified = errors.New("firebase email is not verified")
	ErrRefreshTokenRequired     = errors.New("refresh token is required")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
	ErrSessionExpired           = errors.New("session has expired")
	ErrSessionRevoked           = errors.New("session has been revoked")
)

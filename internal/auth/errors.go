package auth

import "errors"

var (
	ErrInvalidName           = errors.New("name is required")
	ErrInvalidEmail          = errors.New("email must be a valid email address")
	ErrInvalidPassword       = errors.New("password must be at least 8 characters")
	ErrInvalidOTP            = errors.New("otp must be a 6-digit code")
	ErrEmailAlreadyExists    = errors.New("email is already in use")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrPendingOTPNotFound    = errors.New("no pending registration found for this email")
	ErrOTPExpired            = errors.New("otp has expired")
	ErrOTPDoesNotMatch       = errors.New("invalid otp")
	ErrPasswordResetNotFound = errors.New("no password reset request found for this email")
	ErrPasswordResetExpired  = errors.New("password reset token has expired")
	ErrPasswordResetInvalid  = errors.New("invalid password reset token")
	ErrRefreshTokenRequired  = errors.New("refresh token is required")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrSessionExpired        = errors.New("session has expired")
	ErrSessionRevoked        = errors.New("session has been revoked")
)

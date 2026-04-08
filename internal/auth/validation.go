package auth

import (
	"net/mail"
	"regexp"
	"strings"
)

var otpPattern = regexp.MustCompile(`^\d{6}$`)

func validateRegisterInput(input RegisterInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrInvalidName
	}

	if len(strings.TrimSpace(input.Password)) < 8 {
		return ErrInvalidPassword
	}

	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	return nil
}

func validateLoginInput(input LoginInput) error {
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	if strings.TrimSpace(input.Password) == "" {
		return ErrInvalidCredentials
	}

	return nil
}

func validateVerifyRegisterOTPInput(input VerifyRegisterOTPInput) error {
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	if !otpPattern.MatchString(strings.TrimSpace(input.OTP)) {
		return ErrInvalidOTP
	}

	return nil
}

func validateResendRegisterOTPInput(input ResendRegisterOTPInput) error {
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	return nil
}

func validateForgotPasswordInput(input ForgotPasswordInput) error {
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	return nil
}

func validateResetPasswordInput(input ResetPasswordInput) error {
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	if !otpPattern.MatchString(strings.TrimSpace(input.Token)) {
		return ErrInvalidOTP
	}

	if len(strings.TrimSpace(input.NewPassword)) < 8 {
		return ErrInvalidPassword
	}

	return nil
}

func validateChangePasswordInput(input ChangePasswordInput) error {
	if strings.TrimSpace(input.CurrentPassword) == "" {
		return ErrCurrentPasswordWrong
	}

	if len(strings.TrimSpace(input.NewPassword)) < 8 {
		return ErrInvalidPassword
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

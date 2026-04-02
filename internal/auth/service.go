package auth

import (
	"context"
	"eventy-api/internal/platform/jwt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type registrationMailer interface {
	SendRegistrationOTP(toEmail string, otpCode string) error
	SendPasswordResetOTP(toEmail string, otpCode string) error
}

type Service struct {
	repository   *Repository
	tokenManager *jwt.Manager
	mailer       registrationMailer
	registerTTL  time.Duration
	refreshTTL   time.Duration
}

func NewService(repository *Repository, tokenManager *jwt.Manager, mailer registrationMailer, registerTTL time.Duration, refreshTTL time.Duration) *Service {
	return &Service{
		repository:   repository,
		tokenManager: tokenManager,
		mailer:       mailer,
		registerTTL:  registerTTL,
		refreshTTL:   refreshTTL,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (MessageResponse, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = normalizeEmail(input.Email)

	if err := validateRegisterInput(input); err != nil {
		return MessageResponse{}, err
	}

	exists, err := s.repository.UserEmailExists(ctx, input.Email)
	if err != nil {
		return MessageResponse{}, err
	}
	if exists {
		return MessageResponse{}, ErrEmailAlreadyExists
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return MessageResponse{}, err
	}

	otpCode, err := generateOTPCode()
	if err != nil {
		return MessageResponse{}, err
	}

	expiresAt := time.Now().Add(s.registerTTL)
	err = s.repository.SavePendingRegistration(ctx, input, passwordHash, otpCode, expiresAt)
	if err != nil {
		return MessageResponse{}, err
	}

	if err := s.mailer.SendRegistrationOTP(input.Email, otpCode); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{
		Message: "registration otp sent successfully",
	}, nil
}

func (s *Service) ResendRegisterOTP(ctx context.Context, input ResendRegisterOTPInput) (MessageResponse, error) {
	input.Email = normalizeEmail(input.Email)

	if err := validateResendRegisterOTPInput(input); err != nil {
		return MessageResponse{}, err
	}

	pending, err := s.repository.GetPendingRegistrationByEmail(ctx, input.Email)
	if err != nil {
		return MessageResponse{}, err
	}

	otpCode, err := generateOTPCode()
	if err != nil {
		return MessageResponse{}, err
	}

	expiresAt := time.Now().Add(s.registerTTL)
	if err := s.repository.UpdatePendingRegistrationOTP(ctx, pending, otpCode, expiresAt); err != nil {
		return MessageResponse{}, err
	}

	if err := s.mailer.SendRegistrationOTP(input.Email, otpCode); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{
		Message: "registration otp resent successfully",
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	input.Email = normalizeEmail(input.Email)

	if err := validateLoginInput(input); err != nil {
		return AuthResult{}, err
	}

	user, passwordHash, err := s.repository.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return AuthResult{}, err
	}

	if err := comparePassword(passwordHash, input.Password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	return s.createAuthResult(ctx, user)
}

func (s *Service) VerifyRegisterOTP(ctx context.Context, input VerifyRegisterOTPInput) (AuthResult, error) {
	input.Email = normalizeEmail(input.Email)
	input.OTP = strings.TrimSpace(input.OTP)

	if err := validateVerifyRegisterOTPInput(input); err != nil {
		return AuthResult{}, err
	}

	pending, err := s.repository.GetPendingRegistrationByEmail(ctx, input.Email)
	if err != nil {
		return AuthResult{}, err
	}

	if time.Now().After(pending.ExpiresAt) {
		return AuthResult{}, ErrOTPExpired
	}

	if pending.OTPCode != input.OTP {
		return AuthResult{}, ErrOTPDoesNotMatch
	}

	user, err := s.repository.CreateUser(ctx, RegisterInput{
		Name:     pending.Name,
		Email:    pending.Email,
		Password: "",
	}, pending.PasswordHash)
	if err != nil {
		return AuthResult{}, err
	}

	if err := s.repository.DeletePendingRegistrationByEmail(ctx, input.Email); err != nil {
		return AuthResult{}, err
	}

	return s.createAuthResult(ctx, user)
}

func (s *Service) ForgotPassword(ctx context.Context, input ForgotPasswordInput) (MessageResponse, error) {
	input.Email = normalizeEmail(input.Email)

	if err := validateForgotPasswordInput(input); err != nil {
		return MessageResponse{}, err
	}

	user, _, err := s.repository.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return MessageResponse{}, err
	}

	token, err := generateOTPCode()
	if err != nil {
		return MessageResponse{}, err
	}

	expiresAt := time.Now().Add(s.registerTTL)
	if err := s.repository.SavePasswordResetToken(ctx, user, token, expiresAt); err != nil {
		return MessageResponse{}, err
	}

	if err := s.mailer.SendPasswordResetOTP(user.Email, token); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{
		Message: "password reset otp sent successfully",
	}, nil
}

func (s *Service) ResetPassword(ctx context.Context, input ResetPasswordInput) (MessageResponse, error) {
	input.Email = normalizeEmail(input.Email)
	input.Token = strings.TrimSpace(input.Token)

	if err := validateResetPasswordInput(input); err != nil {
		return MessageResponse{}, err
	}

	resetToken, err := s.repository.GetPasswordResetTokenByEmail(ctx, input.Email)
	if err != nil {
		return MessageResponse{}, err
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return MessageResponse{}, ErrPasswordResetExpired
	}

	if resetToken.Token != input.Token {
		return MessageResponse{}, ErrPasswordResetInvalid
	}

	passwordHash, err := hashPassword(input.NewPassword)
	if err != nil {
		return MessageResponse{}, err
	}

	if err := s.repository.UpdateUserPasswordByEmail(ctx, input.Email, passwordHash); err != nil {
		return MessageResponse{}, err
	}

	if err := s.repository.DeletePasswordResetTokenByEmail(ctx, input.Email); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{
		Message: "password reset successfully",
	}, nil
}

func (s *Service) RefreshSession(ctx context.Context, input RefreshTokenInput) (AuthResult, error) {
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)

	if err := validateRefreshTokenInput(input); err != nil {
		return AuthResult{}, err
	}

	session, err := s.repository.GetSessionByRefreshTokenHash(ctx, hashRefreshToken(input.RefreshToken))
	if err != nil {
		return AuthResult{}, err
	}

	if session.RevokedAt.Valid {
		return AuthResult{}, ErrSessionRevoked
	}

	if time.Now().After(session.ExpiresAt) {
		return AuthResult{}, ErrSessionExpired
	}

	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return AuthResult{}, err
	}

	user, _, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return AuthResult{}, err
	}

	return s.rotateSessionAndIssueTokens(ctx, session.ID, user)
}

func (s *Service) Logout(ctx context.Context, input LogoutInput) (MessageResponse, error) {
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)

	if err := validateLogoutInput(input); err != nil {
		return MessageResponse{}, err
	}

	session, err := s.repository.GetSessionByRefreshTokenHash(ctx, hashRefreshToken(input.RefreshToken))
	if err != nil {
		return MessageResponse{}, err
	}

	if session.RevokedAt.Valid {
		return MessageResponse{}, ErrSessionRevoked
	}

	if err := s.repository.RevokeSession(ctx, session.ID); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{Message: "logged out successfully"}, nil
}

func (s *Service) createAuthResult(ctx context.Context, user User) (AuthResult, error) {
	accessToken, accessExpiresAt, err := s.tokenManager.Generate(user.ID, user.Email, user.Role, user.OrganizerID)
	if err != nil {
		return AuthResult{}, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}

	refreshExpiresAt := time.Now().Add(s.refreshTTL)
	if err := s.repository.CreateSession(ctx, user, hashRefreshToken(refreshToken), refreshExpiresAt); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
		User:             user,
	}, nil
}

func (s *Service) rotateSessionAndIssueTokens(ctx context.Context, sessionID string, user User) (AuthResult, error) {
	accessToken, accessExpiresAt, err := s.tokenManager.Generate(user.ID, user.Email, user.Role, user.OrganizerID)
	if err != nil {
		return AuthResult{}, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}

	refreshExpiresAt := time.Now().Add(s.refreshTTL)
	if err := s.repository.RotateSession(ctx, sessionID, hashRefreshToken(refreshToken), refreshExpiresAt); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
		User:             user,
	}, nil
}

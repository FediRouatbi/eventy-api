package auth

import (
	"context"
	"errors"
	eventyfirebase "eventy-api/internal/platform/firebase"
	"eventy-api/internal/platform/jwt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FirebaseVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (eventyfirebase.FirebaseIdentity, error)
}

type Service struct {
	repository       *Repository
	tokenManager     *jwt.Manager
	firebaseVerifier FirebaseVerifier
	refreshTTL       time.Duration
}

func NewService(repository *Repository, tokenManager *jwt.Manager, firebaseVerifier FirebaseVerifier, refreshTTL time.Duration) *Service {
	return &Service{
		repository:       repository,
		tokenManager:     tokenManager,
		firebaseVerifier: firebaseVerifier,
		refreshTTL:       refreshTTL,
	}
}

func (s *Service) LoginWithFirebase(ctx context.Context, input FirebaseLoginInput) (AuthResult, error) {
	input.IDToken = strings.TrimSpace(input.IDToken)
	input.Platform = strings.TrimSpace(input.Platform)

	if err := validateFirebaseLoginInput(input); err != nil {
		return AuthResult{}, err
	}

	if s.firebaseVerifier == nil {
		return AuthResult{}, ErrInvalidFirebaseToken
	}

	identity, err := s.firebaseVerifier.VerifyIDToken(ctx, input.IDToken)
	if err != nil {
		return AuthResult{}, ErrInvalidFirebaseToken
	}

	email := normalizeEmail(identity.Email)
	name := strings.TrimSpace(identity.Name)
	if !isValidEmail(email) {
		return AuthResult{}, ErrInvalidFirebaseToken
	}
	if strings.TrimSpace(identity.UID) == "" {
		return AuthResult{}, ErrInvalidFirebaseToken
	}
	if !identity.EmailVerified {
		return AuthResult{}, ErrFirebaseEmailNotVerified
	}
	if name == "" {
		name = email
	}

	user, _, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, ErrInvalidCredentials) {
			return AuthResult{}, err
		}

		passwordHash, hashErr := hashPassword(uuid.NewString())
		if hashErr != nil {
			return AuthResult{}, hashErr
		}

		user, err = s.repository.CreateUser(ctx, CreateUserInput{
			Name:        name,
			Email:       email,
			FirebaseUID: identity.UID,
		}, passwordHash)
		if err != nil {
			return AuthResult{}, err
		}
	} else if user.FirebaseUID == nil || *user.FirebaseUID != identity.UID {
		user, err = s.repository.UpdateUserFirebaseUID(ctx, user.ID, identity.UID)
		if err != nil {
			return AuthResult{}, err
		}
	}

	return s.createAuthResult(ctx, user)
}

func (s *Service) CheckEmailAvailability(ctx context.Context, input EmailAvailabilityInput) (EmailAvailabilityResult, error) {
	input.Email = normalizeEmail(input.Email)

	if err := validateEmailAvailabilityInput(input); err != nil {
		return EmailAvailabilityResult{}, err
	}

	exists, err := s.repository.UserEmailExists(ctx, input.Email)
	if err != nil {
		return EmailAvailabilityResult{}, err
	}

	return EmailAvailabilityResult{Available: !exists}, nil
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

func (s *Service) Logout(ctx context.Context, userID uuid.UUID, input LogoutInput) (MessageResponse, error) {
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

	sessionUserID, err := uuid.Parse(session.UserID)
	if err != nil {
		return MessageResponse{}, err
	}
	if sessionUserID != userID {
		return MessageResponse{}, ErrInvalidRefreshToken
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

package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"eventy-api/internal/platform/db/sqlc"
	"eventy-api/internal/platform/roles"

	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) CreateUser(ctx context.Context, input CreateUserInput, passwordHash string) (User, error) {
	userID := uuid.New()

	dbUser, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID.String(),
		Name:         strings.TrimSpace(input.Name),
		Email:        normalizeEmail(input.Email),
		FirebaseUID:  sql.NullString{String: strings.TrimSpace(input.FirebaseUID), Valid: strings.TrimSpace(input.FirebaseUID) != ""},
		PasswordHash: passwordHash,
		Role:         roles.User,
		OrganizerID:  sql.NullString{},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailAlreadyExists
		}

		return User{}, err
	}

	return mapUser(dbUser)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", ErrInvalidCredentials
		}

		return User{}, "", err
	}

	user, err := mapUser(dbUser)
	if err != nil {
		return User{}, "", err
	}

	return user, dbUser.PasswordHash, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (User, string, error) {
	dbUser, err := r.queries.GetUserByID(ctx, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", ErrInvalidCredentials
		}

		return User{}, "", err
	}

	user, err := mapUser(dbUser)
	if err != nil {
		return User{}, "", err
	}

	return user, dbUser.PasswordHash, nil
}

func (r *Repository) UserEmailExists(ctx context.Context, email string) (bool, error) {
	return r.queries.CheckUserEmailExists(ctx, normalizeEmail(email))
}

func (r *Repository) UpdateUserFirebaseUID(ctx context.Context, userID uuid.UUID, firebaseUID string) (User, error) {
	if err := r.queries.UpdateUserFirebaseUID(ctx, strings.TrimSpace(firebaseUID), userID.String()); err != nil {
		return User{}, err
	}

	user, _, err := r.GetUserByID(ctx, userID)
	return user, err
}

func (r *Repository) CreateSession(ctx context.Context, user User, refreshTokenHash string, expiresAt time.Time) error {
	return r.queries.CreateAuthSession(ctx, sqlc.CreateAuthSessionParams{
		ID:               uuid.New().String(),
		UserID:           user.ID.String(),
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        expiresAt,
	})
}

func (r *Repository) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (sqlc.AuthSession, error) {
	session, err := r.queries.GetAuthSessionByRefreshTokenHash(ctx, refreshTokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.AuthSession{}, ErrInvalidRefreshToken
		}

		return sqlc.AuthSession{}, err
	}

	return session, nil
}

func (r *Repository) RotateSession(ctx context.Context, sessionID string, refreshTokenHash string, expiresAt time.Time) error {
	return r.queries.UpdateAuthSessionRefreshToken(ctx, refreshTokenHash, expiresAt, sessionID)
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID string) error {
	return r.queries.RevokeAuthSessionByID(ctx, sessionID)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func mapUser(dbUser sqlc.User) (User, error) {
	userID, err := uuid.Parse(dbUser.ID)
	if err != nil {
		return User{}, err
	}

	var organizerID *uuid.UUID
	if dbUser.OrganizerID.Valid {
		parsedOrganizerID, err := uuid.Parse(dbUser.OrganizerID.String)
		if err != nil {
			return User{}, err
		}

		organizerID = &parsedOrganizerID
	}

	return User{
		ID:          userID,
		Name:        dbUser.Name,
		Email:       dbUser.Email,
		FirebaseUID: nullableStringPtr(dbUser.FirebaseUID),
		Role:        dbUser.Role,
		OrganizerID: organizerID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
	}, nil
}

func nullableStringPtr(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	result := value.String
	return &result
}

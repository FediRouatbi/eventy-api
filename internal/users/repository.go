package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"eventy-api/internal/platform/db/sqlc"

	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetProfileByID(ctx context.Context, userID uuid.UUID) (Profile, error) {
	dbUser, err := r.queries.GetUserByID(ctx, userID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, ErrUserNotFound
		}

		return Profile{}, err
	}

	parsedUserID, err := uuid.Parse(dbUser.ID)
	if err != nil {
		return Profile{}, err
	}

	var organizerID *uuid.UUID
	if dbUser.OrganizerID.Valid {
		parsedOrganizerID, err := uuid.Parse(dbUser.OrganizerID.String)
		if err != nil {
			return Profile{}, err
		}

		organizerID = &parsedOrganizerID
	}

	return Profile{
		ID:          parsedUserID,
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

func (r *Repository) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Profile, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	name := strings.TrimSpace(input.Name)

	existingUser, err := r.queries.GetUserByEmail(ctx, email)
	if err == nil && existingUser.ID != userID.String() {
		return Profile{}, ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Profile{}, err
	}

	err = r.queries.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		Name:  name,
		Email: email,
		ID:    userID.String(),
	})
	if err != nil {
		return Profile{}, err
	}

	return r.GetProfileByID(ctx, userID)
}

func (r *Repository) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.GetProfileByID(ctx, userID); err != nil {
		return err
	}

	return r.queries.DeleteUserByID(ctx, userID.String())
}

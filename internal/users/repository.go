package users

import (
	"context"
	"database/sql"
	"errors"

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
		Role:        dbUser.Role,
		OrganizerID: organizerID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
	}, nil
}

package notifications

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertDeviceToken(ctx context.Context, userID uuid.UUID, input RegisterDeviceTokenInput) error {
	const query = `
INSERT INTO notification_device_tokens (
	id,
	user_id,
	token,
	provider,
	platform,
	device_name,
	revoked_at
) VALUES (
	?,
	?,
	?,
	?,
	?,
	?,
	NULL
)
ON DUPLICATE KEY UPDATE
	user_id = VALUES(user_id),
	provider = VALUES(provider),
	platform = VALUES(platform),
	device_name = VALUES(device_name),
	revoked_at = NULL,
	updated_at = CURRENT_TIMESTAMP
`

	_, err := r.db.ExecContext(
		ctx,
		query,
		uuid.New().String(),
		userID.String(),
		strings.TrimSpace(input.Token),
		strings.TrimSpace(input.Provider),
		strings.TrimSpace(input.Platform),
		sql.NullString{
			String: strings.TrimSpace(input.DeviceName),
			Valid:  strings.TrimSpace(input.DeviceName) != "",
		},
	)

	return err
}

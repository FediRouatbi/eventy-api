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

// ListActiveTokensByUserEmail returns the active push tokens registered by the
// user with the given email (used to notify a specific buyer).
func (r *Repository) ListActiveTokensByUserEmail(ctx context.Context, email string) ([]string, error) {
	const query = `
SELECT t.token
FROM notification_device_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.revoked_at IS NULL AND u.email = ?
`
	return r.queryTokens(ctx, query, strings.ToLower(strings.TrimSpace(email)))
}

// ListActiveTokensByEventID returns the active push tokens of every user who
// holds a ticket for the given event (the event's audience).
func (r *Repository) ListActiveTokensByEventID(ctx context.Context, eventID uuid.UUID) ([]string, error) {
	const query = `
SELECT DISTINCT t.token
FROM notification_device_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.revoked_at IS NULL
	AND u.email IN (
		SELECT DISTINCT customer_email FROM tickets WHERE event_id = ?
	)
`
	return r.queryTokens(ctx, query, eventID.String())
}

// ListAllActiveTokens returns every active push token (used for broadcasts such
// as a newly published event).
func (r *Repository) ListAllActiveTokens(ctx context.Context) ([]string, error) {
	const query = `SELECT token FROM notification_device_tokens WHERE revoked_at IS NULL`
	return r.queryTokens(ctx, query)
}

// RevokeToken marks a token as revoked so it is skipped on future sends. Used to
// prune tokens that Firebase reports as unregistered/invalid.
func (r *Repository) RevokeToken(ctx context.Context, token string) error {
	const query = `UPDATE notification_device_tokens SET revoked_at = CURRENT_TIMESTAMP WHERE token = ?`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// RevokeUserDeviceToken revokes a token owned by a specific user (used when the
// user disables notifications for their device).
func (r *Repository) RevokeUserDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	const query = `UPDATE notification_device_tokens SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = ? AND token = ?`
	_, err := r.db.ExecContext(ctx, query, userID.String(), strings.TrimSpace(token))
	return err
}

func (r *Repository) queryTokens(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]string, 0)
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, rows.Err()
}

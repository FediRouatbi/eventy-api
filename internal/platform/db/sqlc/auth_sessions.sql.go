package sqlc

import (
	"context"
	"time"
)

const createAuthSessionQuery = `
INSERT INTO auth_sessions (
    id,
    user_id,
    refresh_token_hash,
    expires_at
) VALUES (
    ?,
    ?,
    ?,
    ?
)
`

const getAuthSessionByRefreshTokenHashQuery = `
SELECT id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, updated_at
FROM auth_sessions
WHERE refresh_token_hash = ?
LIMIT 1
`

const updateAuthSessionRefreshTokenQuery = `
UPDATE auth_sessions
SET refresh_token_hash = ?, expires_at = ?, revoked_at = NULL
WHERE id = ?
`

const revokeAuthSessionByIDQuery = `
UPDATE auth_sessions
SET revoked_at = CURRENT_TIMESTAMP
WHERE id = ?
`

func (q *Queries) CreateAuthSession(ctx context.Context, arg CreateAuthSessionParams) error {
	_, err := q.db.ExecContext(ctx, createAuthSessionQuery, arg.ID, arg.UserID, arg.RefreshTokenHash, arg.ExpiresAt)
	return err
}

func (q *Queries) GetAuthSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (AuthSession, error) {
	row := q.db.QueryRowContext(ctx, getAuthSessionByRefreshTokenHashQuery, refreshTokenHash)

	var session AuthSession
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	return session, err
}

func (q *Queries) UpdateAuthSessionRefreshToken(ctx context.Context, refreshTokenHash string, expiresAt time.Time, id string) error {
	_, err := q.db.ExecContext(ctx, updateAuthSessionRefreshTokenQuery, refreshTokenHash, expiresAt, id)
	return err
}

func (q *Queries) RevokeAuthSessionByID(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, revokeAuthSessionByIDQuery, id)
	return err
}

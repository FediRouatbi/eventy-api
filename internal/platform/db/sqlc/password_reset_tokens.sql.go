package sqlc

import "context"

const upsertPasswordResetTokenQuery = `
INSERT INTO password_reset_tokens (
    id,
    user_id,
    email,
    token,
    expires_at
) VALUES (
    UUID(),
    ?,
    ?,
    ?,
    ?
)
ON DUPLICATE KEY UPDATE
    user_id = VALUES(user_id),
    token = VALUES(token),
    expires_at = VALUES(expires_at),
    updated_at = CURRENT_TIMESTAMP
`

const getPasswordResetTokenByEmailQuery = `
SELECT id, user_id, email, token, expires_at, created_at, updated_at
FROM password_reset_tokens
WHERE email = ?
LIMIT 1
`

const deletePasswordResetTokenByEmailQuery = `
DELETE FROM password_reset_tokens
WHERE email = ?
`

func (q *Queries) UpsertPasswordResetToken(ctx context.Context, arg UpsertPasswordResetTokenParams) error {
	_, err := q.db.ExecContext(ctx, upsertPasswordResetTokenQuery, arg.UserID, arg.Email, arg.Token, arg.ExpiresAt)
	return err
}

func (q *Queries) GetPasswordResetTokenByEmail(ctx context.Context, email string) (PasswordResetToken, error) {
	row := q.db.QueryRowContext(ctx, getPasswordResetTokenByEmailQuery, email)

	var resetToken PasswordResetToken
	err := row.Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.Email,
		&resetToken.Token,
		&resetToken.ExpiresAt,
		&resetToken.CreatedAt,
		&resetToken.UpdatedAt,
	)

	return resetToken, err
}

func (q *Queries) DeletePasswordResetTokenByEmail(ctx context.Context, email string) error {
	_, err := q.db.ExecContext(ctx, deletePasswordResetTokenByEmailQuery, email)
	return err
}

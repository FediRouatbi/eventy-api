package sqlc

import (
	"context"
)

const upsertPendingRegistrationQuery = `
INSERT INTO pending_registrations (
    id,
    name,
    email,
    password_hash,
    otp_code,
    expires_at
) VALUES (
    UUID(),
    ?,
    ?,
    ?,
    ?,
    ?
)
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    password_hash = VALUES(password_hash),
    otp_code = VALUES(otp_code),
    expires_at = VALUES(expires_at),
    updated_at = CURRENT_TIMESTAMP
`

const getPendingRegistrationByEmailQuery = `
SELECT id, name, email, password_hash, otp_code, expires_at, created_at, updated_at
FROM pending_registrations
WHERE email = ?
LIMIT 1
`

const deletePendingRegistrationByEmailQuery = `
DELETE FROM pending_registrations
WHERE email = ?
`

func (q *Queries) UpsertPendingRegistration(ctx context.Context, arg UpsertPendingRegistrationParams) error {
	_, err := q.db.ExecContext(ctx, upsertPendingRegistrationQuery, arg.Name, arg.Email, arg.PasswordHash, arg.OTPCode, arg.ExpiresAt)
	return err
}

func (q *Queries) GetPendingRegistrationByEmail(ctx context.Context, email string) (PendingRegistration, error) {
	row := q.db.QueryRowContext(ctx, getPendingRegistrationByEmailQuery, email)

	var pending PendingRegistration
	err := row.Scan(
		&pending.ID,
		&pending.Name,
		&pending.Email,
		&pending.PasswordHash,
		&pending.OTPCode,
		&pending.ExpiresAt,
		&pending.CreatedAt,
		&pending.UpdatedAt,
	)

	return pending, err
}

func (q *Queries) DeletePendingRegistrationByEmail(ctx context.Context, email string) error {
	_, err := q.db.ExecContext(ctx, deletePendingRegistrationByEmailQuery, email)
	return err
}

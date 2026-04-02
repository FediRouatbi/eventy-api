-- name: UpsertPasswordResetToken :exec
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
    updated_at = CURRENT_TIMESTAMP;

-- name: GetPasswordResetTokenByEmail :one
SELECT id, user_id, email, token, expires_at, created_at, updated_at
FROM password_reset_tokens
WHERE email = ?
LIMIT 1;

-- name: DeletePasswordResetTokenByEmail :exec
DELETE FROM password_reset_tokens
WHERE email = ?;

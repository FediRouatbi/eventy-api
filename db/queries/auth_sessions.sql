-- name: CreateAuthSession :exec
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
);

-- name: GetAuthSessionByRefreshTokenHash :one
SELECT id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, updated_at
FROM auth_sessions
WHERE refresh_token_hash = ?
LIMIT 1;

-- name: UpdateAuthSessionRefreshToken :exec
UPDATE auth_sessions
SET refresh_token_hash = ?, expires_at = ?, revoked_at = NULL
WHERE id = ?;

-- name: RevokeAuthSessionByID :exec
UPDATE auth_sessions
SET revoked_at = CURRENT_TIMESTAMP
WHERE id = ?;

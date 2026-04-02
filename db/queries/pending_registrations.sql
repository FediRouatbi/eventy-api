-- name: UpsertPendingRegistration :exec
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
    updated_at = CURRENT_TIMESTAMP;

-- name: GetPendingRegistrationByEmail :one
SELECT id, name, email, password_hash, otp_code, expires_at, created_at, updated_at
FROM pending_registrations
WHERE email = ?
LIMIT 1;

-- name: DeletePendingRegistrationByEmail :exec
DELETE FROM pending_registrations
WHERE email = ?;

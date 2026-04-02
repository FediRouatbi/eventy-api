-- name: CreateUser :one
INSERT INTO users (
    id,
    name,
    email,
    password_hash,
    role,
    organizer_id
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: UpdateUserPasswordByEmail :exec
UPDATE users
SET password_hash = ?
WHERE email = ?;

-- name: CheckUserEmailExists :one
SELECT EXISTS(
    SELECT 1
    FROM users
    WHERE email = ?
);

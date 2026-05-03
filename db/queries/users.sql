-- name: CreateUser :one
INSERT INTO users (
    id,
    name,
    email,
    firebase_uid,
    password_hash,
    role,
    organizer_id
) VALUES (
    ?,
    NULLIF(?, ''),
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetUserByEmail :one
SELECT id, name, email, firebase_uid, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, name, email, firebase_uid, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: DeleteUserByID :exec
DELETE FROM users
WHERE id = ?;

-- name: UpdateUserPasswordByEmail :exec
UPDATE users
SET password_hash = ?
WHERE email = ?;

-- name: UpdateUserFirebaseUID :exec
UPDATE users
SET firebase_uid = NULLIF(?, '')
WHERE id = ?;

-- name: CheckUserEmailExists :one
SELECT EXISTS(
    SELECT 1
    FROM users
    WHERE email = ?
);

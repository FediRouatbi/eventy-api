package sqlc

import (
	"context"
	"database/sql"
	"errors"
)

const createUserQuery = `
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
)
`

const getUserByEmailQuery = `
SELECT id, name, email, firebase_uid, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1
`

const getUserByIDQuery = `
SELECT id, name, email, firebase_uid, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1
`

const deleteUserByIDQuery = `
DELETE FROM users
WHERE id = ?
`

const updateUserPasswordByEmailQuery = `
UPDATE users
SET password_hash = ?
WHERE email = ?
`

const updateUserFirebaseUIDQuery = `
UPDATE users
SET firebase_uid = NULLIF(?, '')
WHERE id = ?
`

const updateUserPasswordByIDQuery = `
UPDATE users
SET password_hash = ?
WHERE id = ?
`

const updateUserProfileQuery = `
UPDATE users
SET name = ?,
    email = ?
WHERE id = ?
`

const checkUserEmailExistsQuery = `
SELECT EXISTS(
    SELECT 1
    FROM users
    WHERE email = ?
)
`

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	_, err := q.db.ExecContext(ctx, createUserQuery, arg.ID, arg.Name, arg.Email, arg.FirebaseUID, arg.PasswordHash, arg.Role, arg.OrganizerID)
	if err != nil {
		return User{}, err
	}

	return q.GetUserByID(ctx, arg.ID)
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmailQuery, email)

	var user User
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.FirebaseUID,
		&user.PasswordHash,
		&user.Role,
		&user.OrganizerID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, sql.ErrNoRows
	}

	return user, err
}

func (q *Queries) GetUserByID(ctx context.Context, id string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByIDQuery, id)

	var user User
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.FirebaseUID,
		&user.PasswordHash,
		&user.Role,
		&user.OrganizerID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, sql.ErrNoRows
	}

	return user, err
}

func (q *Queries) DeleteUserByID(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, deleteUserByIDQuery, id)
	return err
}

func (q *Queries) CheckUserEmailExists(ctx context.Context, email string) (bool, error) {
	row := q.db.QueryRowContext(ctx, checkUserEmailExistsQuery, email)

	var exists bool
	err := row.Scan(&exists)

	return exists, err
}

func (q *Queries) UpdateUserPasswordByEmail(ctx context.Context, passwordHash string, email string) error {
	_, err := q.db.ExecContext(ctx, updateUserPasswordByEmailQuery, passwordHash, email)
	return err
}

func (q *Queries) UpdateUserFirebaseUID(ctx context.Context, firebaseUID string, id string) error {
	_, err := q.db.ExecContext(ctx, updateUserFirebaseUIDQuery, firebaseUID, id)
	return err
}

func (q *Queries) UpdateUserPasswordByID(ctx context.Context, passwordHash string, id string) error {
	_, err := q.db.ExecContext(ctx, updateUserPasswordByIDQuery, passwordHash, id)
	return err
}

type UpdateUserProfileParams struct {
	Name  string
	Email string
	ID    string
}

func (q *Queries) UpdateUserProfile(ctx context.Context, arg UpdateUserProfileParams) error {
	_, err := q.db.ExecContext(ctx, updateUserProfileQuery, arg.Name, arg.Email, arg.ID)
	return err
}

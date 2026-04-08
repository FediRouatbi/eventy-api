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
)
`

const getUserByEmailQuery = `
SELECT id, name, email, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1
`

const getUserByIDQuery = `
SELECT id, name, email, password_hash, role, organizer_id, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1
`

const updateUserPasswordByEmailQuery = `
UPDATE users
SET password_hash = ?
WHERE email = ?
`

const updateUserPasswordByIDQuery = `
UPDATE users
SET password_hash = ?
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
	_, err := q.db.ExecContext(ctx, createUserQuery, arg.ID, arg.Name, arg.Email, arg.PasswordHash, arg.Role, arg.OrganizerID)
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

func (q *Queries) UpdateUserPasswordByID(ctx context.Context, passwordHash string, id string) error {
	_, err := q.db.ExecContext(ctx, updateUserPasswordByIDQuery, passwordHash, id)
	return err
}

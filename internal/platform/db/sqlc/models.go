package sqlc

import (
	"database/sql"
	"time"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	OrganizerID  sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateUserParams struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	OrganizerID  sql.NullString
}

type PendingRegistration struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	OTPCode      string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UpsertPendingRegistrationParams struct {
	Name         string
	Email        string
	PasswordHash string
	OTPCode      string
	ExpiresAt    time.Time
}

type PasswordResetToken struct {
	ID        string
	UserID    string
	Email     string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UpsertPasswordResetTokenParams struct {
	UserID    string
	Email     string
	Token     string
	ExpiresAt time.Time
}

type AuthSession struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
	RevokedAt        sql.NullTime
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateAuthSessionParams struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
}

type Category struct {
	ID        string
	Name      string
	Slug      string
	ImageURL  sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateCategoryParams struct {
	ID       string
	Name     string
	Slug     string
	ImageURL sql.NullString
}

type Event struct {
	ID           string
	OrganizerID  string
	CategoryID   string
	Title        string
	Slug         string
	Description  string
	VenueName    string
	VenueAddress string
	City         string
	Country      string
	BannerURL    sql.NullString
	PosterURL    sql.NullString
	Status       string
	Currency     string
	IsFeatured   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateEventParams struct {
	ID           string
	OrganizerID  string
	CategoryID   string
	Title        string
	Slug         string
	Description  string
	VenueName    string
	VenueAddress string
	City         string
	Country      string
	BannerURL    sql.NullString
	PosterURL    sql.NullString
	Status       string
	Currency     string
	IsFeatured   bool
}

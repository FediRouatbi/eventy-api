package sqlc

import (
	"database/sql"
	"time"
)

type User struct {
	ID           string
	Name         string
	Email        string
	FirebaseUID  sql.NullString
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
	FirebaseUID  sql.NullString
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
	ID          string
	Name        string
	Slug        string
	Description sql.NullString
	ImageURL    sql.NullString
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateCategoryParams struct {
	ID          string
	Name        string
	Slug        string
	Description sql.NullString
	ImageURL    sql.NullString
}

type UpdateCategoryParams struct {
	Name        string
	Slug        string
	Description sql.NullString
	ImageURL    sql.NullString
	ID          string
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
	Latitude     sql.NullFloat64
	Longitude    sql.NullFloat64
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
	Latitude     sql.NullFloat64
	Longitude    sql.NullFloat64
	BannerURL    sql.NullString
	PosterURL    sql.NullString
	Status       string
	Currency     string
	IsFeatured   bool
}

type UpdateEventParams struct {
	CategoryID   string
	Title        string
	Slug         string
	Description  string
	VenueName    string
	VenueAddress string
	City         string
	Country      string
	Latitude     sql.NullFloat64
	Longitude    sql.NullFloat64
	BannerURL    sql.NullString
	PosterURL    sql.NullString
	Status       string
	Currency     string
	IsFeatured   bool
	ID           string
}

type EventSession struct {
	ID            string
	EventID       string
	StartsAt      time.Time
	EndsAt        time.Time
	SalesStartsAt sql.NullTime
	SalesEndsAt   sql.NullTime
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateEventSessionParams struct {
	ID            string
	EventID       string
	StartsAt      time.Time
	EndsAt        time.Time
	SalesStartsAt sql.NullTime
	SalesEndsAt   sql.NullTime
	Status        string
}

type UpdateEventSessionParams struct {
	StartsAt      time.Time
	EndsAt        time.Time
	SalesStartsAt sql.NullTime
	SalesEndsAt   sql.NullTime
	Status        string
	ID            string
}

type TicketType struct {
	ID             string
	EventSessionID string
	Name           string
	Description    sql.NullString
	Price          float64
	Quantity       int32
	MaxPerOrder    int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateTicketTypeParams struct {
	ID             string
	EventSessionID string
	Name           string
	Description    sql.NullString
	Price          float64
	Quantity       int32
	MaxPerOrder    int32
}

type UpdateTicketTypeParams struct {
	Name        string
	Description sql.NullString
	Price       float64
	Quantity    int32
	MaxPerOrder int32
	ID          string
}

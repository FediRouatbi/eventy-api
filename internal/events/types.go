package events

import (
	"time"

	"github.com/google/uuid"
)

type CreateEventInput struct {
	OrganizerID  string   `json:"organizer_id,omitempty"`
	CategoryID   string   `json:"category_id"`
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	VenueName    string   `json:"venue_name"`
	VenueAddress string   `json:"venue_address"`
	City         string   `json:"city"`
	Country      string   `json:"country"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	BannerURL    string   `json:"banner_url"`
	PosterURL    string   `json:"poster_url"`
	Status       string   `json:"status"`
	Currency     string   `json:"currency"`
	IsFeatured   bool     `json:"is_featured"`
}

type UpdateEventInput struct {
	CategoryID   string   `json:"category_id"`
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	VenueName    string   `json:"venue_name"`
	VenueAddress string   `json:"venue_address"`
	City         string   `json:"city"`
	Country      string   `json:"country"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	BannerURL    string   `json:"banner_url"`
	PosterURL    string   `json:"poster_url"`
	Status       string   `json:"status"`
	Currency     string   `json:"currency"`
	IsFeatured   bool     `json:"is_featured"`
}

type CreateEventSessionInput struct {
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        time.Time  `json:"ends_at"`
	SalesStartsAt *time.Time `json:"sales_starts_at,omitempty"`
	SalesEndsAt   *time.Time `json:"sales_ends_at,omitempty"`
	Status        string     `json:"status"`
}

type UpdateEventSessionInput struct {
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        time.Time  `json:"ends_at"`
	SalesStartsAt *time.Time `json:"sales_starts_at,omitempty"`
	SalesEndsAt   *time.Time `json:"sales_ends_at,omitempty"`
	Status        string     `json:"status"`
}

type CreateTicketTypeInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int32   `json:"quantity"`
	MaxPerOrder int32   `json:"max_per_order"`
}

type UpdateTicketTypeInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int32   `json:"quantity"`
	MaxPerOrder int32   `json:"max_per_order"`
}

type Event struct {
	ID           uuid.UUID `json:"id"`
	OrganizerID  uuid.UUID `json:"organizer_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	VenueName    string    `json:"venue_name"`
	VenueAddress string    `json:"venue_address"`
	City         string    `json:"city"`
	Country      string    `json:"country"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	BannerURL    string    `json:"banner_url,omitempty"`
	PosterURL    string    `json:"poster_url,omitempty"`
	Status       string    `json:"status"`
	Currency     string    `json:"currency"`
	IsFeatured   bool      `json:"is_featured"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EventListItem struct {
	Event
	OrganizerName             string     `json:"organizer_name"`
	OrganizerSlug             string     `json:"organizer_slug"`
	CategoryName              string     `json:"category_name"`
	CategorySlug              string     `json:"category_slug"`
	SessionCount              int        `json:"session_count"`
	TicketTypeCount           int        `json:"ticket_type_count"`
	NextSessionStartsAt       *time.Time `json:"next_session_starts_at,omitempty"`
	HasSessionsWithoutTickets bool       `json:"has_sessions_without_tickets"`
}

type EventSession struct {
	ID            uuid.UUID  `json:"id"`
	EventID       uuid.UUID  `json:"event_id"`
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        time.Time  `json:"ends_at"`
	SalesStartsAt *time.Time `json:"sales_starts_at,omitempty"`
	SalesEndsAt   *time.Time `json:"sales_ends_at,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TicketType struct {
	ID             uuid.UUID `json:"id"`
	EventSessionID uuid.UUID `json:"event_session_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Price          float64   `json:"price"`
	Quantity       int32     `json:"quantity"`
	MaxPerOrder    int32     `json:"max_per_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type EventSessionDetail struct {
	EventSession
	TicketTypes []TicketType `json:"ticket_types"`
}

type EventDetail struct {
	Event
	Sessions []EventSessionDetail `json:"sessions"`
}

type PublicEvent struct {
	ID                  uuid.UUID            `json:"id"`
	OrganizerID         uuid.UUID            `json:"organizer_id"`
	OrganizerName       string               `json:"organizer_name"`
	OrganizerSlug       string               `json:"organizer_slug"`
	CategoryID          uuid.UUID            `json:"category_id"`
	CategoryName        string               `json:"category_name"`
	CategorySlug        string               `json:"category_slug"`
	CategoryDescription string               `json:"category_description,omitempty"`
	CategoryImageURL    string               `json:"category_image_url,omitempty"`
	Title               string               `json:"title"`
	Slug                string               `json:"slug"`
	Summary             string               `json:"summary"`
	Description         string               `json:"description"`
	VenueName           string               `json:"venue_name"`
	VenueAddress        string               `json:"venue_address"`
	City                string               `json:"city"`
	Country             string               `json:"country"`
	Latitude            *float64             `json:"latitude,omitempty"`
	Longitude           *float64             `json:"longitude,omitempty"`
	BannerURL           string               `json:"banner_url,omitempty"`
	PosterURL           string               `json:"poster_url,omitempty"`
	Status              string               `json:"status"`
	Currency            string               `json:"currency"`
	IsFeatured          bool                 `json:"is_featured"`
	NextSessionID       *uuid.UUID           `json:"next_session_id,omitempty"`
	NextSessionStartsAt *time.Time           `json:"next_session_starts_at,omitempty"`
	NextSessionEndsAt   *time.Time           `json:"next_session_ends_at,omitempty"`
	NextSalesStartsAt   *time.Time           `json:"next_sales_starts_at,omitempty"`
	NextSalesEndsAt     *time.Time           `json:"next_sales_ends_at,omitempty"`
	PriceFrom           float64              `json:"price_from"`
	TicketsLeft         int32                `json:"tickets_left"`
	Sessions            []EventSessionDetail `json:"sessions,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

type PublicEventDetail struct {
	PublicEvent
}

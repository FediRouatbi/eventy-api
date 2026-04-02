package events

import (
	"time"

	"github.com/google/uuid"
)

type CreateEventInput struct {
	OrganizerID  string `json:"organizer_id,omitempty"`
	CategoryID   string `json:"category_id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	VenueName    string `json:"venue_name"`
	VenueAddress string `json:"venue_address"`
	City         string `json:"city"`
	Country      string `json:"country"`
	BannerURL    string `json:"banner_url"`
	PosterURL    string `json:"poster_url"`
	Status       string `json:"status"`
	Currency     string `json:"currency"`
	IsFeatured   bool   `json:"is_featured"`
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
	BannerURL    string    `json:"banner_url,omitempty"`
	PosterURL    string    `json:"poster_url,omitempty"`
	Status       string    `json:"status"`
	Currency     string    `json:"currency"`
	IsFeatured   bool      `json:"is_featured"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

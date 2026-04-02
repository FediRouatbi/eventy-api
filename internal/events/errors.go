package events

import "errors"

var (
	ErrInvalidOrganizerID     = errors.New("organizer_id is required and must be a valid uuid")
	ErrInvalidCategoryID      = errors.New("category_id is required and must be a valid uuid")
	ErrInvalidTitle           = errors.New("title is required")
	ErrInvalidSlug            = errors.New("slug is required")
	ErrInvalidDescription     = errors.New("description is required")
	ErrInvalidVenueName       = errors.New("venue_name is required")
	ErrInvalidVenueAddress    = errors.New("venue_address is required")
	ErrInvalidCity            = errors.New("city is required")
	ErrInvalidCountry         = errors.New("country is required")
	ErrInvalidCurrency        = errors.New("currency is required")
	ErrInvalidStatus          = errors.New("status must be one of: draft, published, cancelled")
	ErrEventNotFound          = errors.New("event not found")
	ErrEventSlugAlreadyExists = errors.New("event slug already exists")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrOrganizerNotFound      = errors.New("organizer not found")
	ErrForbidden              = errors.New("forbidden")
	ErrOrganizerScopeRequired = errors.New("organizer scope is required for this account")
	ErrUnsupportedRole        = errors.New("role is not allowed to manage events")
)

package events

import "strings"

func validateCreateEventInput(input CreateEventInput) error {
	if strings.TrimSpace(input.CategoryID) == "" {
		return ErrInvalidCategoryID
	}

	if strings.TrimSpace(input.Title) == "" {
		return ErrInvalidTitle
	}

	if strings.TrimSpace(input.Slug) == "" {
		return ErrInvalidSlug
	}

	if strings.TrimSpace(input.Description) == "" {
		return ErrInvalidDescription
	}

	if strings.TrimSpace(input.VenueName) == "" {
		return ErrInvalidVenueName
	}

	if strings.TrimSpace(input.VenueAddress) == "" {
		return ErrInvalidVenueAddress
	}

	if strings.TrimSpace(input.City) == "" {
		return ErrInvalidCity
	}

	if strings.TrimSpace(input.Country) == "" {
		return ErrInvalidCountry
	}

	if strings.TrimSpace(input.Currency) == "" {
		return ErrInvalidCurrency
	}

	switch strings.TrimSpace(input.Status) {
	case "draft", "published", "cancelled":
		return nil
	default:
		return ErrInvalidStatus
	}
}

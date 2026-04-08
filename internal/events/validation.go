package events

import (
	"strings"
	"time"
)

func validateCreateEventInput(input CreateEventInput) error {
	return validateEventFields(
		input.CategoryID,
		input.Title,
		input.Slug,
		input.Description,
		input.VenueName,
		input.VenueAddress,
		input.City,
		input.Country,
		input.Latitude,
		input.Longitude,
		input.Currency,
		input.Status,
	)
}

func validateUpdateEventInput(input UpdateEventInput) error {
	return validateEventFields(
		input.CategoryID,
		input.Title,
		input.Slug,
		input.Description,
		input.VenueName,
		input.VenueAddress,
		input.City,
		input.Country,
		input.Latitude,
		input.Longitude,
		input.Currency,
		input.Status,
	)
}

func validateEventFields(categoryID, title, slug, description, venueName, venueAddress, city, country string, latitude, longitude *float64, currency, status string) error {
	if strings.TrimSpace(categoryID) == "" {
		return ErrInvalidCategoryID
	}

	if strings.TrimSpace(title) == "" {
		return ErrInvalidTitle
	}

	if strings.TrimSpace(slug) == "" {
		return ErrInvalidSlug
	}

	if strings.TrimSpace(description) == "" {
		return ErrInvalidDescription
	}

	if strings.TrimSpace(venueName) == "" {
		return ErrInvalidVenueName
	}

	if strings.TrimSpace(venueAddress) == "" {
		return ErrInvalidVenueAddress
	}

	if strings.TrimSpace(city) == "" {
		return ErrInvalidCity
	}

	if strings.TrimSpace(country) == "" {
		return ErrInvalidCountry
	}

	if (latitude == nil) != (longitude == nil) {
		return ErrInvalidCoordinates
	}

	if latitude != nil && (*latitude < -90 || *latitude > 90) {
		return ErrInvalidLatitude
	}

	if longitude != nil && (*longitude < -180 || *longitude > 180) {
		return ErrInvalidLongitude
	}

	if strings.TrimSpace(currency) == "" {
		return ErrInvalidCurrency
	}

	switch strings.TrimSpace(status) {
	case "draft", "published", "cancelled":
		return nil
	default:
		return ErrInvalidStatus
	}
}

func validateEventSessionInput(startsAt time.Time, endsAt time.Time, salesStartsAt *time.Time, salesEndsAt *time.Time, status string) error {
	if startsAt.IsZero() {
		return ErrInvalidStartsAt
	}

	if !endsAt.After(startsAt) {
		return ErrInvalidEndsAt
	}

	if salesStartsAt != nil && salesEndsAt != nil && salesEndsAt.Before(*salesStartsAt) {
		return ErrInvalidSalesWindow
	}

	if salesStartsAt != nil && salesStartsAt.After(startsAt) {
		return ErrInvalidSalesWindow
	}

	if salesEndsAt != nil && salesEndsAt.After(endsAt) {
		return ErrInvalidSalesWindow
	}

	switch strings.TrimSpace(status) {
	case "scheduled", "completed", "cancelled":
		return nil
	default:
		return ErrInvalidSessionStatus
	}
}

func validateTicketTypeInput(name string, price float64, quantity int32, maxPerOrder int32) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidTicketName
	}

	if price < 0 {
		return ErrInvalidTicketPrice
	}

	if quantity <= 0 {
		return ErrInvalidTicketQuantity
	}

	if maxPerOrder <= 0 || maxPerOrder > quantity {
		return ErrInvalidMaxPerOrder
	}

	return nil
}

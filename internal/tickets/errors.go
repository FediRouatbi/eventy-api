package tickets

import "errors"

var (
	ErrInvalidTicketCode     = errors.New("invalid ticket code")
	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketNotPaid         = errors.New("ticket is not paid")
	ErrOrganizerScopeMissing = errors.New("organizer scope required")
	ErrForbidden             = errors.New("forbidden")
)

package admins

import "errors"

var (
	ErrInvalidOrganizerName   = errors.New("organizer_name is required")
	ErrInvalidOrganizerSlug   = errors.New("organizer_slug is required")
	ErrInvalidOrganizerID     = errors.New("organizer_id must be a valid uuid")
	ErrInvalidAdminID         = errors.New("admin_id must be a valid uuid")
	ErrInvalidAdminName       = errors.New("admin_name is required")
	ErrInvalidAdminEmail      = errors.New("admin_email must be a valid email")
	ErrInvalidAdminPassword   = errors.New("admin_password must be at least 8 characters")
	ErrInvalidLimit           = errors.New("limit must be a positive integer")
	ErrOrganizerSlugExists    = errors.New("organizer slug already exists")
	ErrOrganizerNotFound      = errors.New("organizer not found")
	ErrAdminEmailExists       = errors.New("admin email already exists")
	ErrOrganizerAdminNotFound = errors.New("organizer admin not found")
	ErrLastOrganizerAdmin     = errors.New("cannot delete the last organizer admin")
	ErrOrganizerScopeRequired = errors.New("organizer scope is required for this account")
)

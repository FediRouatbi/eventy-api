package admins

import (
	"net/mail"
	"strings"
)

func validateCreateOrganizerAdminInput(input CreateOrganizerAdminInput) error {
	if strings.TrimSpace(input.OrganizerName) == "" {
		return ErrInvalidOrganizerName
	}

	if strings.TrimSpace(input.OrganizerSlug) == "" {
		return ErrInvalidOrganizerSlug
	}

	if strings.TrimSpace(input.AdminName) == "" {
		return ErrInvalidAdminName
	}

	if !isValidEmail(input.AdminEmail) {
		return ErrInvalidAdminEmail
	}

	return nil
}

func validateUpdateOrganizerInput(input UpdateOrganizerInput) error {
	if strings.TrimSpace(input.OrganizerName) == "" {
		return ErrInvalidOrganizerName
	}

	if strings.TrimSpace(input.OrganizerSlug) == "" {
		return ErrInvalidOrganizerSlug
	}

	return nil
}

func validateUpdateOrganizerAdminInput(input UpdateOrganizerAdminInput) error {
	if strings.TrimSpace(input.AdminName) == "" {
		return ErrInvalidAdminName
	}

	if !isValidEmail(input.AdminEmail) {
		return ErrInvalidAdminEmail
	}

	return nil
}

func validateAddOrganizerAdminInput(input AddOrganizerAdminInput) error {
	if strings.TrimSpace(input.AdminName) == "" {
		return ErrInvalidAdminName
	}

	if !isValidEmail(input.AdminEmail) {
		return ErrInvalidAdminEmail
	}

	return nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}

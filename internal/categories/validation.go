package categories

import "strings"

func validateCreateCategoryInput(input CreateCategoryInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrInvalidName
	}

	if strings.TrimSpace(input.Slug) == "" {
		return ErrInvalidSlug
	}

	return nil
}

func validateUpdateCategoryInput(input UpdateCategoryInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrInvalidName
	}

	if strings.TrimSpace(input.Slug) == "" {
		return ErrInvalidSlug
	}

	return nil
}

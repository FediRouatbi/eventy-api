package categories

import "errors"

var (
	ErrInvalidName      = errors.New("name is required")
	ErrInvalidSlug      = errors.New("slug is required")
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryHasEvents = errors.New("category cannot be deleted while events still use it")
	ErrCategoryExists   = errors.New("category already exists")
	ErrCategoryNameUsed = errors.New("category name already exists")
	ErrCategorySlugUsed = errors.New("category slug already exists")
)

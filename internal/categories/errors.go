package categories

import "errors"

var (
	ErrInvalidName = errors.New("name is required")
	ErrInvalidSlug = errors.New("slug is required")
)

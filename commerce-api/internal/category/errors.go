package category

import "errors"

var (
	// ErrCategoryNotFound means the requested active category
	// does not exist.
	ErrCategoryNotFound = errors.New("category not found")

	// ErrInvalidCategorySlug means the caller supplied an empty slug.
	ErrInvalidCategorySlug = errors.New("category slug is required")
)

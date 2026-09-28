package search

import "errors"

var (
	ErrInvalidQuery           = errors.New("invalid search query")
	ErrInvalidPage            = errors.New("invalid search page")
	ErrInvalidLimit           = errors.New("invalid search limit")
	ErrInvalidImage           = errors.New("invalid search image")
	ErrImageTooLarge          = errors.New("search image too large")
	ErrUnsupportedImageType   = errors.New("unsupported search image type")
	ErrImageSearchUnavailable = errors.New("image search unavailable")
)

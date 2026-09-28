package pagination

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20

	MaxPage  = 100000
	MaxLimit = 100
)

var (
	ErrInvalidPage = errors.New(
		"invalid pagination page",
	)

	ErrInvalidLimit = errors.New(
		"invalid pagination limit",
	)
)

type Params struct {
	Page  int
	Limit int
}

type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`

	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`
}

func Parse(
	pageValue string,
	limitValue string,
) (
	Params,
	error,
) {
	page := DefaultPage
	limit := DefaultLimit

	pageValue = strings.TrimSpace(
		pageValue,
	)
	if pageValue != "" {
		parsed, err := strconv.Atoi(
			pageValue,
		)
		if err != nil {
			return Params{},
				fmt.Errorf(
					"%w: page must be an integer",
					ErrInvalidPage,
				)
		}

		page = parsed
	}

	limitValue = strings.TrimSpace(
		limitValue,
	)
	if limitValue != "" {
		parsed, err := strconv.Atoi(
			limitValue,
		)
		if err != nil {
			return Params{},
				fmt.Errorf(
					"%w: limit must be an integer",
					ErrInvalidLimit,
				)
		}

		limit = parsed
	}

	return New(
		page,
		limit,
	)
}

func New(
	page int,
	limit int,
) (
	Params,
	error,
) {
	if page < 1 ||
		page > MaxPage {
		return Params{},
			fmt.Errorf(
				"%w: page must be between 1 and %d",
				ErrInvalidPage,
				MaxPage,
			)
	}

	if limit < 1 ||
		limit > MaxLimit {
		return Params{},
			fmt.Errorf(
				"%w: limit must be between 1 and %d",
				ErrInvalidLimit,
				MaxLimit,
			)
	}

	return Params{
			Page:  page,
			Limit: limit,
		},
		nil
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

func (p Params) Offset64() int64 {
	return int64(
		p.Page-1,
	) * int64(
		p.Limit,
	)
}

func NewMeta(
	params Params,
	total int64,
) Meta {
	if total < 0 {
		total = 0
	}

	totalPages := int64(0)

	if total > 0 {
		limit := int64(
			params.Limit,
		)

		totalPages = total / limit

		if total%limit != 0 {
			totalPages++
		}
	}

	currentPage := int64(
		params.Page,
	)

	return Meta{
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,

		HasPrevious: params.Page > 1,
		HasNext: totalPages > 0 &&
			currentPage < totalPages,
	}
}

package search

import (
	"strings"
	"unicode/utf8"
)

const (
	defaultSearchPage  = 1
	defaultSearchLimit = 20
	maxSearchLimit     = 100
	maxSearchPage      = 100000
	maxSearchQueryRune = 120
)

type normalizedInput struct {
	DisplayQuery    string
	NormalizedQuery string
	Page            int
	Limit           int
}

func normalizeInput(input Input) (normalizedInput, error) {
	displayQuery := strings.Join(
		strings.Fields(input.Query),
		" ",
	)

	if displayQuery == "" ||
		utf8.RuneCountInString(displayQuery) > maxSearchQueryRune {
		return normalizedInput{}, ErrInvalidQuery
	}

	page, limit, err := normalizePaging(
		input.Page,
		input.Limit,
	)
	if err != nil {
		return normalizedInput{}, err
	}

	return normalizedInput{
		DisplayQuery: displayQuery,
		NormalizedQuery: strings.ToLower(
			displayQuery,
		),
		Page:  page,
		Limit: limit,
	}, nil
}

func normalizePaging(
	page int,
	limit int,
) (int, int, error) {
	if page == 0 {
		page = defaultSearchPage
	}

	if page < 1 || page > maxSearchPage {
		return 0, 0, ErrInvalidPage
	}

	if limit == 0 {
		limit = defaultSearchLimit
	}

	if limit < 1 || limit > maxSearchLimit {
		return 0, 0, ErrInvalidLimit
	}

	return page, limit, nil
}

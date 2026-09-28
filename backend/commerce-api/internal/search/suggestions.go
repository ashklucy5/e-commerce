package search

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	defaultSuggestionLimit = 6
	maxSuggestionLimit     = 10
	minSuggestionQueryRune = 2
)

type SuggestionsResult struct {
	Query      string            `json:"query"`
	Products   []ProductResult   `json:"products"`
	Categories []CategorySummary `json:"categories"`
	Brands     []string          `json:"brands"`
}

func (s *Service) Suggestions(
	ctx context.Context,
	query string,
	limit int,
) (SuggestionsResult, error) {
	displayQuery := strings.Join(
		strings.Fields(query),
		" ",
	)

	if utf8.RuneCountInString(displayQuery) >
		maxSearchQueryRune {
		return SuggestionsResult{},
			ErrInvalidQuery
	}

	if limit == 0 {
		limit = defaultSuggestionLimit
	}

	if limit < 1 ||
		limit > maxSuggestionLimit {
		return SuggestionsResult{},
			ErrInvalidLimit
	}

	result := SuggestionsResult{
		Query: displayQuery,

		Products: make(
			[]ProductResult,
			0,
		),

		Categories: make(
			[]CategorySummary,
			0,
		),

		Brands: make(
			[]string,
			0,
		),
	}

	// Do not hit PostgreSQL for one-character
	// keystrokes such as "p".
	if utf8.RuneCountInString(displayQuery) <
		minSuggestionQueryRune {
		return result, nil
	}

	// Fetch a few more candidates than the number
	// displayed because products are also used to
	// derive unique category and brand suggestions.
	candidateLimit := limit * 2

	if candidateLimit > 20 {
		candidateLimit = 20
	}

	page, err := s.store.SearchProducts(
		ctx,
		strings.ToLower(displayQuery),
		1,
		candidateLimit,
	)
	if err != nil {
		return SuggestionsResult{}, err
	}

	categorySeen := make(
		map[string]struct{},
	)

	brandSeen := make(
		map[string]struct{},
	)

	for _, item := range page.Items {
		if len(result.Products) < limit {
			result.Products = append(
				result.Products,
				item,
			)
		}

		categorySlug := strings.TrimSpace(
			item.Category.Slug,
		)

		if categorySlug != "" &&
			len(result.Categories) < 4 {
			key := strings.ToLower(
				categorySlug,
			)

			if _, exists := categorySeen[key]; !exists {
				categorySeen[key] = struct{}{}

				result.Categories = append(
					result.Categories,
					item.Category,
				)
			}
		}

		if item.Brand != nil &&
			len(result.Brands) < 4 {
			brand := strings.TrimSpace(
				*item.Brand,
			)

			if brand == "" {
				continue
			}

			key := strings.ToLower(
				brand,
			)

			if _, exists := brandSeen[key]; exists {
				continue
			}

			brandSeen[key] = struct{}{}

			result.Brands = append(
				result.Brands,
				brand,
			)
		}
	}

	return result, nil
}

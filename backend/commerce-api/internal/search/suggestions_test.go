package search

import (
	"context"
	"errors"
	"testing"
)

func TestSuggestionsShortQueryReturnsEmpty(t *testing.T) {
	service := &Service{}

	result, err := service.Suggestions(
		context.Background(),
		"p",
		6,
	)
	if err != nil {
		t.Fatalf("Suggestions() error = %v", err)
	}

	if result.Query != "p" {
		t.Fatalf(
			"Query = %q, want %q",
			result.Query,
			"p",
		)
	}

	if len(result.Products) != 0 {
		t.Fatalf(
			"Products = %d, want 0",
			len(result.Products),
		)
	}

	if len(result.Categories) != 0 {
		t.Fatalf(
			"Categories = %d, want 0",
			len(result.Categories),
		)
	}

	if len(result.Brands) != 0 {
		t.Fatalf(
			"Brands = %d, want 0",
			len(result.Brands),
		)
	}
}

func TestSuggestionsRejectsInvalidLimit(t *testing.T) {
	service := &Service{}

	_, err := service.Suggestions(
		context.Background(),
		"pant",
		maxSuggestionLimit+1,
	)

	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf(
			"Suggestions() error = %v, want %v",
			err,
			ErrInvalidLimit,
		)
	}
}

func TestSuggestionsRejectsNegativeLimit(t *testing.T) {
	service := &Service{}

	_, err := service.Suggestions(
		context.Background(),
		"pant",
		-1,
	)

	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf(
			"Suggestions() error = %v, want %v",
			err,
			ErrInvalidLimit,
		)
	}
}

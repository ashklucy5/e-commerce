package pagination

import (
	"errors"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	params, err := Parse(
		"",
		"",
	)
	if err != nil {
		t.Fatalf(
			"parse defaults: %v",
			err,
		)
	}

	if params.Page != DefaultPage {
		t.Fatalf(
			"expected page %d, got %d",
			DefaultPage,
			params.Page,
		)
	}

	if params.Limit != DefaultLimit {
		t.Fatalf(
			"expected limit %d, got %d",
			DefaultLimit,
			params.Limit,
		)
	}
}

func TestParseRejectsInvalidLimit(t *testing.T) {
	_, err := Parse(
		"1",
		"101",
	)
	if err == nil {
		t.Fatal(
			"expected invalid limit error",
		)
	}

	if !errors.Is(
		err,
		ErrInvalidLimit,
	) {
		t.Fatalf(
			"expected ErrInvalidLimit, got %v",
			err,
		)
	}
}

func TestNewMeta(t *testing.T) {
	params, err := New(
		2,
		20,
	)
	if err != nil {
		t.Fatalf(
			"new params: %v",
			err,
		)
	}

	meta := NewMeta(
		params,
		45,
	)

	if meta.TotalPages != 3 {
		t.Fatalf(
			"expected 3 total pages, got %d",
			meta.TotalPages,
		)
	}

	if !meta.HasPrevious {
		t.Fatal(
			"expected previous page",
		)
	}

	if !meta.HasNext {
		t.Fatal(
			"expected next page",
		)
	}

	if params.Offset() != 20 {
		t.Fatalf(
			"expected offset 20, got %d",
			params.Offset(),
		)
	}
}

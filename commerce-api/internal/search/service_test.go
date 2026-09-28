package search

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	page SearchPage
	err  error

	semanticPage SearchPage
	semanticErr  error

	searchQuery string
	searchPage  int
	searchLimit int

	semanticCalls      int
	semanticVector     string
	semanticModel      string
	semanticPageNumber int
	semanticLimit      int
	semanticPrimaryMin float64
	semanticSimilarMin float64

	historyCalls      int
	historyCustomerID string
	historyQuery      string
	historyNormalized string
	historyResult     int64
	historyCategories []HistoryCategorySignal
	historyErr        error
}

func (f *fakeStore) SearchProducts(
	_ context.Context,
	normalizedQuery string,
	page int,
	limit int,
) (SearchPage, error) {
	f.searchQuery =
		normalizedQuery

	f.searchPage =
		page

	f.searchLimit =
		limit

	if f.err != nil {
		return SearchPage{},
			f.err
	}

	return f.page, nil
}

func (f *fakeStore) SearchSemanticProducts(
	_ context.Context,
	queryVector string,
	model string,
	page int,
	limit int,
	primaryMin float64,
	similarMin float64,
) (SearchPage, error) {
	f.semanticCalls++

	f.semanticVector =
		queryVector

	f.semanticModel =
		model

	f.semanticPageNumber =
		page

	f.semanticLimit =
		limit

	f.semanticPrimaryMin =
		primaryMin

	f.semanticSimilarMin =
		similarMin

	if f.semanticErr != nil {
		return SearchPage{},
			f.semanticErr
	}

	return f.semanticPage, nil
}

func (f *fakeStore) RecordSearchHistory(
	_ context.Context,
	customerID string,
	query string,
	normalizedQuery string,
	resultCount int64,
	categories []HistoryCategorySignal,
) error {
	f.historyCalls++

	f.historyCustomerID =
		customerID

	f.historyQuery =
		query

	f.historyNormalized =
		normalizedQuery

	f.historyResult =
		resultCount

	f.historyCategories =
		append(
			[]HistoryCategorySignal(nil),
			categories...,
		)

	return f.historyErr
}

type fakeEmbedder struct {
	model string

	queryEmbedding []float32
	queryErr       error

	queryCalls int
	lastQuery  string
}

func (f *fakeEmbedder) Model() string {
	if f.model == "" {
		return defaultEmbeddingModel
	}

	return f.model
}

func (f *fakeEmbedder) EmbedQuery(
	_ context.Context,
	text string,
) ([]float32, error) {
	f.queryCalls++
	f.lastQuery =
		text

	if f.queryErr != nil {
		return nil,
			f.queryErr
	}

	return f.queryEmbedding,
		nil
}

func (f *fakeEmbedder) EmbedDocuments(
	_ context.Context,
	_ []string,
) ([][]float32, error) {
	return nil,
		errors.New(
			"not implemented by test embedder",
		)
}

func testEmbedding() []float32 {
	result :=
		make(
			[]float32,
			searchEmbeddingDimensions,
		)

	result[0] =
		1

	return result
}

func lexicalTestService(
	store Store,
) *Service {
	return newServiceWithSemantic(
		store,
		nil,
		semanticConfig{},
	)
}

func TestSearchNormalizesAndRecordsPrimaryHistory(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Total:        3,
				PrimaryTotal: 2,
				SimilarTotal: 1,

				Items: []ProductResult{
					{
						ID: "product-1",

						SearchTier: 1,

						MatchType: "primary",

						Category: CategorySummary{
							ID: "category-shirts",

							Name: "Shirts",

							Slug: "shirts",
						},
					},
					{
						ID: "product-2",

						SearchTier: 1,

						MatchType: "primary",

						Category: CategorySummary{
							ID: "category-formal",

							Name: "Formal Shirts",

							Slug: "formal-shirts",
						},
					},
					{
						ID: "product-3",

						SearchTier: 2,

						MatchType: "related",

						Category: CategorySummary{
							ID: "category-polos",

							Name: "Polos",

							Slug: "polos",
						},
					},
				},
			},
		}

	service :=
		lexicalTestService(
			store,
		)

	result, err :=
		service.Search(
			context.Background(),
			"customer-1",
			Input{
				Query: "  Formal   Shirt  ",
			},
		)
	if err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if store.searchQuery !=
		"formal shirt" {
		t.Fatalf(
			"unexpected normalized query %q",
			store.searchQuery,
		)
	}

	if store.searchPage != 1 ||
		store.searchLimit != 20 {
		t.Fatalf(
			"unexpected paging page=%d limit=%d",
			store.searchPage,
			store.searchLimit,
		)
	}

	if result.Meta.PrimaryTotal != 2 ||
		result.Meta.SimilarTotal != 1 ||
		result.Meta.NoMatch ||
		!result.Meta.ShowingSimilar {
		t.Fatalf(
			"unexpected meta: %#v",
			result.Meta,
		)
	}

	if store.historyCalls != 1 {
		t.Fatalf(
			"expected one history call, got %d",
			store.historyCalls,
		)
	}

	if store.historyResult != 2 {
		t.Fatalf(
			"history must learn primary_total, got %d",
			store.historyResult,
		)
	}

	if len(
		store.historyCategories,
	) != 2 {
		t.Fatalf(
			"expected only primary categories, got %#v",
			store.historyCategories,
		)
	}

	if store.historyCategories[0].CategoryID !=
		"category-shirts" ||
		store.historyCategories[0].RelevanceWeight !=
			100 ||
		store.historyCategories[1].CategoryID !=
			"category-formal" ||
		store.historyCategories[1].RelevanceWeight !=
			95 {
		t.Fatalf(
			"unexpected history categories: %#v",
			store.historyCategories,
		)
	}
}

func TestSearchGuestDoesNotRecordHistory(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Items: []ProductResult{},

				Total: 0,

				PrimaryTotal: 0,

				SimilarTotal: 0,
			},
		}

	service :=
		lexicalTestService(
			store,
		)

	if _, err :=
		service.Search(
			context.Background(),
			"",
			Input{
				Query: "shirt",
			},
		); err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if store.historyCalls != 0 {
		t.Fatalf(
			"guest search recorded history %d times",
			store.historyCalls,
		)
	}
}

func TestSearchPageTwoDoesNotRecordHistory(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Items: []ProductResult{},

				Total: 35,

				PrimaryTotal: 35,
			},
		}

	service :=
		lexicalTestService(
			store,
		)

	result, err :=
		service.Search(
			context.Background(),
			"customer-1",
			Input{
				Query: "t shirt",

				Page: 2,

				Limit: 20,
			},
		)
	if err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if result.Meta.TotalPages != 2 {
		t.Fatalf(
			"expected 2 pages, got %d",
			result.Meta.TotalPages,
		)
	}

	if store.historyCalls != 0 {
		t.Fatalf(
			"page 2 recorded history %d times",
			store.historyCalls,
		)
	}
}

func TestSearchNoPrimaryButSimilarSetsNoMatch(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Total: 1,

				PrimaryTotal: 0,

				SimilarTotal: 1,

				Items: []ProductResult{
					{
						ID: "similar-1",

						SearchTier: 2,

						MatchType: "related",

						Category: CategorySummary{
							ID: "related-category",
						},
					},
				},
			},
		}

	service :=
		lexicalTestService(
			store,
		)

	result, err :=
		service.Search(
			context.Background(),
			"customer-1",
			Input{
				Query: "missing item",
			},
		)
	if err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if !result.Meta.NoMatch {
		t.Fatal(
			"expected no_match=true",
		)
	}

	if !result.Meta.ShowingSimilar {
		t.Fatal(
			"expected showing_similar=true",
		)
	}

	if len(
		store.historyCategories,
	) != 0 {
		t.Fatalf(
			"similar results must not become recommendation intent: %#v",
			store.historyCategories,
		)
	}

	if store.historyResult != 0 {
		t.Fatalf(
			"expected history result_count 0, got %d",
			store.historyResult,
		)
	}
}

func TestSearchPrimaryBeforeSimilarPageDoesNotClaimShowingSimilar(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Total: 40,

				PrimaryTotal: 35,

				SimilarTotal: 5,

				Items: []ProductResult{
					{
						ID: "primary-1",

						SearchTier: 1,

						MatchType: "primary",
					},
				},
			},
		}

	service :=
		lexicalTestService(
			store,
		)

	result, err :=
		service.Search(
			context.Background(),
			"",
			Input{
				Query: "t shirt",

				Page: 1,

				Limit: 20,
			},
		)
	if err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if result.Meta.ShowingSimilar {
		t.Fatal(
			"page containing only primary results must not show similar flag",
		)
	}

	if result.Meta.TotalPages != 2 {
		t.Fatalf(
			"expected 2 total pages, got %d",
			result.Meta.TotalPages,
		)
	}
}

func TestSearchBlankQueryFails(
	t *testing.T,
) {
	service :=
		lexicalTestService(
			&fakeStore{},
		)

	_, err :=
		service.Search(
			context.Background(),
			"",
			Input{
				Query: "   ",
			},
		)

	if !errors.Is(
		err,
		ErrInvalidQuery,
	) {
		t.Fatalf(
			"expected ErrInvalidQuery, got %v",
			err,
		)
	}
}

func TestSearchHistoryFailureDoesNotFailSearch(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Items: []ProductResult{},

				Total: 0,

				PrimaryTotal: 0,
			},

			historyErr: errors.New(
				"database unavailable",
			),
		}

	service :=
		lexicalTestService(
			store,
		)

	if _, err :=
		service.Search(
			context.Background(),
			"customer-1",
			Input{
				Query: "shirt",
			},
		); err != nil {
		t.Fatalf(
			"history failure must not fail search: %v",
			err,
		)
	}
}

func TestSearchUsesSemanticFallbackWhenLexicalHasNoMatch(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Items: []ProductResult{},

				Total: 0,

				PrimaryTotal: 0,

				SimilarTotal: 0,
			},

			semanticPage: SearchPage{
				Items: []ProductResult{
					{
						ID: "toothpaste-1",

						SearchTier: 1,

						MatchType: "primary",

						Category: CategorySummary{
							ID: "oral-care",
						},
					},
				},

				Total: 1,

				PrimaryTotal: 1,

				SimilarTotal: 0,
			},
		}

	embedder :=
		&fakeEmbedder{
			queryEmbedding: testEmbedding(),
		}

	service :=
		newServiceWithSemantic(
			store,
			embedder,
			semanticConfig{
				Enabled: true,

				PrimaryMin: 0.62,

				SimilarMin: 0.50,
			},
		)

	result, err :=
		service.Search(
			context.Background(),
			"",
			Input{
				Query: "teeth cleaner",
			},
		)
	if err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if embedder.queryCalls != 1 {
		t.Fatalf(
			"expected one semantic query call, got %d",
			embedder.queryCalls,
		)
	}

	if store.semanticCalls != 1 {
		t.Fatalf(
			"expected one semantic repository call, got %d",
			store.semanticCalls,
		)
	}

	if result.Meta.NoMatch {
		t.Fatal(
			"semantic primary result must clear no_match",
		)
	}

	if result.Meta.PrimaryTotal != 1 {
		t.Fatalf(
			"expected semantic primary total 1, got %d",
			result.Meta.PrimaryTotal,
		)
	}

	if len(result.Items) != 1 ||
		result.Items[0].ID !=
			"toothpaste-1" {
		t.Fatalf(
			"unexpected semantic results: %#v",
			result.Items,
		)
	}
}

func TestSearchDoesNotUseSemanticWhenLexicalPrimaryExists(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Items: []ProductResult{
					{
						ID: "shirt-1",

						SearchTier: 1,

						MatchType: "primary",
					},
				},

				Total: 1,

				PrimaryTotal: 1,
			},
		}

	embedder :=
		&fakeEmbedder{
			queryEmbedding: testEmbedding(),
		}

	service :=
		newServiceWithSemantic(
			store,
			embedder,
			semanticConfig{
				Enabled: true,

				PrimaryMin: 0.62,

				SimilarMin: 0.50,
			},
		)

	if _, err :=
		service.Search(
			context.Background(),
			"",
			Input{
				Query: "shirt",
			},
		); err != nil {
		t.Fatalf(
			"search: %v",
			err,
		)
	}

	if embedder.queryCalls != 0 {
		t.Fatalf(
			"semantic provider called despite lexical match: %d",
			embedder.queryCalls,
		)
	}

	if store.semanticCalls != 0 {
		t.Fatalf(
			"semantic repository called despite lexical match: %d",
			store.semanticCalls,
		)
	}
}

func TestSearchSemanticFailureFallsBackToLexicalResult(
	t *testing.T,
) {
	store :=
		&fakeStore{
			page: SearchPage{
				Items: []ProductResult{},

				Total: 0,

				PrimaryTotal: 0,
			},
		}

	embedder :=
		&fakeEmbedder{
			queryErr: errors.New(
				"provider unavailable",
			),
		}

	service :=
		newServiceWithSemantic(
			store,
			embedder,
			semanticConfig{
				Enabled: true,

				PrimaryMin: 0.62,

				SimilarMin: 0.50,
			},
		)

	result, err :=
		service.Search(
			context.Background(),
			"",
			Input{
				Query: "unknown semantic request",
			},
		)
	if err != nil {
		t.Fatalf(
			"semantic failure must not fail search: %v",
			err,
		)
	}

	if !result.Meta.NoMatch {
		t.Fatal(
			"expected lexical no-match fallback",
		)
	}

	if store.semanticCalls != 0 {
		t.Fatalf(
			"semantic repository must not run without embedding, calls=%d",
			store.semanticCalls,
		)
	}
}

package search

import (
	"context"
	"errors"
	"testing"
)

type fakeImageEmbedder struct {
	*fakeEmbedder

	imageEmbedding []float32
	imageErr       error

	imageCalls int

	lastMediaType string

	lastImageBytes int
}

func (f *fakeImageEmbedder) EmbedImageQuery(
	_ context.Context,
	mediaType string,
	data []byte,
) ([]float32, error) {
	f.imageCalls++

	f.lastMediaType =
		mediaType

	f.lastImageBytes =
		len(data)

	if f.imageErr != nil {
		return nil,
			f.imageErr
	}

	return f.imageEmbedding,
		nil
}

func TestSearchImageUsesSemanticRetrieval(
	t *testing.T,
) {
	store :=
		&fakeStore{
			semanticPage: SearchPage{
				Items: []ProductResult{
					{
						ID: "shirt-1",

						SearchTier: 1,

						MatchType: "primary",

						Category: CategorySummary{
							ID: "shirts",
						},
					},
				},

				Total: 1,

				PrimaryTotal: 1,

				SimilarTotal: 0,
			},
		}

	embedder :=
		&fakeImageEmbedder{
			fakeEmbedder: &fakeEmbedder{
				model: "test-model",
			},

			imageEmbedding: testEmbedding(),
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
		service.SearchImage(
			context.Background(),
			"",
			ImageInput{
				MediaType: "image/jpeg",

				Data: []byte{
					1,
					2,
					3,
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"image search: %v",
			err,
		)
	}

	if embedder.imageCalls != 1 {
		t.Fatalf(
			"expected one image embedding call, got %d",
			embedder.imageCalls,
		)
	}

	if store.semanticCalls != 1 {
		t.Fatalf(
			"expected one semantic repository call, got %d",
			store.semanticCalls,
		)
	}

	if result.Meta.Mode !=
		"image" {
		t.Fatalf(
			"expected image mode, got %q",
			result.Meta.Mode,
		)
	}

	if result.Meta.NoMatch {
		t.Fatal(
			"image primary result must clear no_match",
		)
	}

	if len(result.Items) != 1 ||
		result.Items[0].ID !=
			"shirt-1" {
		t.Fatalf(
			"unexpected image search results: %#v",
			result.Items,
		)
	}
}

func TestSearchImageNoPrimaryButSimilar(
	t *testing.T,
) {
	store :=
		&fakeStore{
			semanticPage: SearchPage{
				Items: []ProductResult{
					{
						ID: "similar-1",

						SearchTier: 2,

						MatchType: "similar",
					},
				},

				Total: 1,

				PrimaryTotal: 0,

				SimilarTotal: 1,
			},
		}

	embedder :=
		&fakeImageEmbedder{
			fakeEmbedder: &fakeEmbedder{
				model: "test-model",
			},

			imageEmbedding: testEmbedding(),
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
		service.SearchImage(
			context.Background(),
			"",
			ImageInput{
				MediaType: "image/png",

				Data: []byte{
					1,
					2,
					3,
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"image search: %v",
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
}

func TestSearchImageRequiresImageCapableEmbedder(
	t *testing.T,
) {
	service :=
		newServiceWithSemantic(
			&fakeStore{},
			&fakeEmbedder{
				queryEmbedding: testEmbedding(),
			},
			semanticConfig{
				Enabled: true,

				PrimaryMin: 0.62,

				SimilarMin: 0.50,
			},
		)

	_, err :=
		service.SearchImage(
			context.Background(),
			"",
			ImageInput{
				MediaType: "image/jpeg",

				Data: []byte{
					1,
				},
			},
		)

	if !errors.Is(
		err,
		ErrImageSearchUnavailable,
	) {
		t.Fatalf(
			"expected ErrImageSearchUnavailable, got %v",
			err,
		)
	}
}

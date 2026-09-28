package search

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

type Store interface {
	SearchProducts(
		ctx context.Context,
		normalizedQuery string,
		page int,
		limit int,
	) (SearchPage, error)

	SearchSemanticProducts(
		ctx context.Context,
		queryVector string,
		model string,
		page int,
		limit int,
		primaryMin float64,
		similarMin float64,
	) (SearchPage, error)

	RecordSearchHistory(
		ctx context.Context,
		customerID string,
		query string,
		normalizedQuery string,
		resultCount int64,
		categories []HistoryCategorySignal,
	) error
}

type Service struct {
	store Store

	semanticConfig semanticConfig
	embedder       Embedder
}

func NewService(store Store) *Service {
	service := &Service{
		store: store,
	}

	cfg, err := loadSemanticConfigFromEnv()
	if err != nil {
		log.Printf(
			"semantic search disabled: %v",
			err,
		)

		return service
	}

	service.semanticConfig = cfg

	if !cfg.Enabled {
		return service
	}

	service.embedder = newVoyageEmbedder(cfg)

	return service
}

func newServiceWithSemantic(
	store Store,
	embedder Embedder,
	cfg semanticConfig,
) *Service {
	return &Service{
		store:          store,
		embedder:       embedder,
		semanticConfig: cfg,
	}
}

func (s *Service) Search(
	ctx context.Context,
	customerID string,
	input Input,
) (Result, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return Result{}, err
	}

	pageResult, err := s.store.SearchProducts(
		ctx,
		normalized.NormalizedQuery,
		normalized.Page,
		normalized.Limit,
	)
	if err != nil {
		return Result{}, err
	}

	if pageResult.PrimaryTotal == 0 &&
		s.embedder != nil &&
		s.semanticConfig.Enabled &&
		shouldTrySemantic(normalized.NormalizedQuery) {
		embedding, embeddingErr := s.embedder.EmbedQuery(
			ctx,
			normalized.NormalizedQuery,
		)

		if embeddingErr != nil {
			log.Printf(
				"semantic search query embedding failed: %v",
				embeddingErr,
			)
		} else {
			queryVector, vectorErr := vectorLiteral(
				embedding,
			)

			if vectorErr != nil {
				log.Printf(
					"semantic search vector encoding failed: %v",
					vectorErr,
				)
			} else {
				semanticPage, semanticErr := s.store.SearchSemanticProducts(
					ctx,
					queryVector,
					s.embedder.Model(),
					normalized.Page,
					normalized.Limit,
					s.semanticConfig.PrimaryMin,
					s.semanticConfig.SimilarMin,
				)

				if semanticErr != nil {
					log.Printf(
						"semantic product search failed: %v",
						semanticErr,
					)
				} else {
					switch {
					case semanticPage.PrimaryTotal > 0:
						pageResult = semanticPage

					case pageResult.Total == 0 &&
						semanticPage.Total > 0:
						pageResult = semanticPage
					}
				}
			}
		}
	}

	result := buildResult(
		"text",
		normalized.DisplayQuery,
		normalized.Page,
		normalized.Limit,
		pageResult,
	)

	s.recordHistory(
		ctx,
		customerID,
		normalized.Page,
		normalized.DisplayQuery,
		normalized.NormalizedQuery,
		pageResult,
	)

	return result, nil
}

func (s *Service) SearchImage(
	ctx context.Context,
	customerID string,
	input ImageInput,
) (Result, error) {
	page, limit, err := normalizePaging(
		input.Page,
		input.Limit,
	)
	if err != nil {
		return Result{}, err
	}

	if len(input.Data) == 0 {
		return Result{}, ErrInvalidImage
	}

	if int64(len(input.Data)) > maxSearchImageBytes {
		return Result{}, ErrImageTooLarge
	}

	if !isSupportedSearchImageMediaType(
		input.MediaType,
	) {
		return Result{}, ErrUnsupportedImageType
	}

	if s.embedder == nil ||
		!s.semanticConfig.Enabled {
		return Result{}, ErrImageSearchUnavailable
	}

	imageEmbedder, ok :=
		s.embedder.(ImageQueryEmbedder)

	if !ok {
		return Result{}, ErrImageSearchUnavailable
	}

	embedding, err := imageEmbedder.EmbedImageQuery(
		ctx,
		input.MediaType,
		input.Data,
	)
	if err != nil {
		if errors.Is(
			err,
			ErrInvalidImage,
		) ||
			errors.Is(
				err,
				ErrImageTooLarge,
			) ||
			errors.Is(
				err,
				ErrUnsupportedImageType,
			) {
			return Result{}, err
		}

		return Result{}, fmt.Errorf(
			"%w: %v",
			ErrImageSearchUnavailable,
			err,
		)
	}

	queryVector, err := vectorLiteral(
		embedding,
	)
	if err != nil {
		return Result{}, fmt.Errorf(
			"%w: %v",
			ErrImageSearchUnavailable,
			err,
		)
	}

	pageResult, err := s.store.SearchSemanticProducts(
		ctx,
		queryVector,
		imageEmbedder.Model(),
		page,
		limit,
		s.semanticConfig.PrimaryMin,
		s.semanticConfig.SimilarMin,
	)
	if err != nil {
		return Result{}, err
	}

	result := buildResult(
		"image",
		"",
		page,
		limit,
		pageResult,
	)

	s.recordHistory(
		ctx,
		customerID,
		page,
		"image search",
		"image search",
		pageResult,
	)

	return result, nil
}

func buildResult(
	mode string,
	query string,
	page int,
	limit int,
	pageResult SearchPage,
) Result {
	if pageResult.Items == nil {
		pageResult.Items = make(
			[]ProductResult,
			0,
		)
	}

	showingSimilar := false

	for _, item := range pageResult.Items {
		if item.SearchTier >= 2 {
			showingSimilar = true
			break
		}
	}

	return Result{
		Items: pageResult.Items,

		Meta: Meta{
			Mode: mode,

			Query: query,

			Page: page,

			Limit: limit,

			Total: pageResult.Total,

			TotalPages: totalPages(
				pageResult.Total,
				limit,
			),

			PrimaryTotal: pageResult.PrimaryTotal,

			SimilarTotal: pageResult.SimilarTotal,

			NoMatch: pageResult.PrimaryTotal == 0,

			ShowingSimilar: showingSimilar,
		},
	}
}

func (s *Service) recordHistory(
	ctx context.Context,
	customerID string,
	page int,
	displayQuery string,
	normalizedQuery string,
	pageResult SearchPage,
) {
	customerID = strings.TrimSpace(
		customerID,
	)

	if customerID == "" ||
		page != 1 {
		return
	}

	categories := historyCategories(
		pageResult.Items,
	)

	if err := s.store.RecordSearchHistory(
		ctx,
		customerID,
		displayQuery,
		normalizedQuery,
		pageResult.PrimaryTotal,
		categories,
	); err != nil {
		log.Printf(
			"record customer search history: %v",
			err,
		)
	}
}

func totalPages(
	total int64,
	limit int,
) int {
	if total <= 0 ||
		limit <= 0 {
		return 0
	}

	return int(
		(total + int64(limit) - 1) /
			int64(limit),
	)
}

func historyCategories(
	items []ProductResult,
) []HistoryCategorySignal {
	const maxHistoryCategories = 8

	seen := make(
		map[string]struct{},
	)

	result := make(
		[]HistoryCategorySignal,
		0,
		maxHistoryCategories,
	)

	for _, item := range items {
		if item.SearchTier > 1 {
			continue
		}

		categoryID := strings.TrimSpace(
			item.Category.ID,
		)

		if categoryID == "" {
			continue
		}

		if _, exists := seen[categoryID]; exists {
			continue
		}

		weight :=
			100 -
				len(result)*5

		if weight < 60 {
			weight = 60
		}

		result = append(
			result,
			HistoryCategorySignal{
				CategoryID:      categoryID,
				RelevanceWeight: weight,
			},
		)

		seen[categoryID] =
			struct{}{}

		if len(result) >=
			maxHistoryCategories {
			break
		}
	}

	return result
}

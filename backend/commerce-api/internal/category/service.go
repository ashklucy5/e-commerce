package category

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	platformcache "project.local/commerce-api/internal/platform/cache"
)

type Service struct {
	repository *Repository

	cache *platformcache.Store
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) SetCache(
	store *platformcache.Store,
) {
	s.cache =
		store
}

func (s *Service) ListActive(
	ctx context.Context,
) (
	[]Category,
	error,
) {
	if s.cache == nil {
		return s.listActiveFromRepository(
			ctx,
		)
	}

	return platformcache.RememberJSON(
		ctx,
		s.cache,
		platformcache.PublicCategoriesKey(),
		platformcache.PublicCategoriesTTL,
		func(
			loadCtx context.Context,
		) (
			[]Category,
			error,
		) {
			return s.listActiveFromRepository(
				loadCtx,
			)
		},
	)
}

func (s *Service) listActiveFromRepository(
	ctx context.Context,
) (
	[]Category,
	error,
) {
	categories, err :=
		s.repository.ListActive(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list active categories: %w",
				err,
			)
	}

	return categories, nil
}

func (s *Service) GetActiveBySlug(
	ctx context.Context,
	slug string,
) (
	Category,
	error,
) {
	slug =
		strings.TrimSpace(
			slug,
		)

	if slug == "" {
		return Category{},
			ErrInvalidCategorySlug
	}

	// When caching is enabled, the active category list becomes the
	// canonical cached read model.
	//
	// This avoids maintaining separate list/tree/slug cache entries
	// that could expire at different times.
	if s.cache != nil {
		categories, err :=
			s.ListActive(
				ctx,
			)
		if err != nil {
			return Category{},
				err
		}

		for _, item := range categories {

			if item.Slug ==
				slug {

				return item, nil
			}
		}

		return Category{},
			ErrCategoryNotFound
	}

	item, err :=
		s.repository.FindActiveBySlug(
			ctx,
			slug,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Category{},
				ErrCategoryNotFound
		}

		return Category{},
			fmt.Errorf(
				"get active category by slug: %w",
				err,
			)
	}

	return item, nil
}

func (s *Service) Tree(
	ctx context.Context,
) (
	[]CategoryNode,
	error,
) {
	categories, err :=
		s.ListActive(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"build category tree: %w",
				err,
			)
	}

	return BuildTree(
			categories,
		),
		nil
}

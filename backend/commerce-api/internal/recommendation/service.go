package recommendation

import (
	"context"
	"strings"
)

const defaultRecommendationLimit = 200

type RankingStore interface {
	RankProductIDs(
		ctx context.Context,
		customerID string,
		limit int,
	) ([]string, error)
}

type Ranker interface {
	RankProductIDs(
		ctx context.Context,
		customerID string,
	) ([]string, error)
}

type Service struct {
	store RankingStore
}

func NewService(
	store RankingStore,
) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) RankProductIDs(
	ctx context.Context,
	customerID string,
) ([]string, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	if customerID == "" {
		return make(
			[]string,
			0,
		), nil
	}

	productIDs, err :=
		s.store.RankProductIDs(
			ctx,
			customerID,
			defaultRecommendationLimit,
		)
	if err != nil {
		return nil, err
	}

	if productIDs == nil {
		productIDs = make(
			[]string,
			0,
		)
	}

	return productIDs, nil
}

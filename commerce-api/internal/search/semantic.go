package search

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	searchEmbeddingDimensions = 1024

	defaultEmbeddingModel = "voyage-multimodal-3.5"

	defaultSemanticPrimaryMin = 0.62
	defaultSemanticSimilarMin = 0.50

	defaultEmbeddingTimeout = 10 * time.Second

	maxSearchImageBytes = int64(10 * 1024 * 1024)

	maxSearchImageRequestBytes = maxSearchImageBytes +
		int64(1024*1024)
)

type semanticConfig struct {
	Enabled bool

	APIKey string
	Model  string

	PrimaryMin float64
	SimilarMin float64

	Timeout time.Duration
}

func loadSemanticConfigFromEnv() (
	semanticConfig,
	error,
) {
	enabled := false

	enabledRaw := strings.TrimSpace(
		os.Getenv(
			"SEARCH_SEMANTIC_ENABLED",
		),
	)

	if enabledRaw != "" {
		parsed, err := strconv.ParseBool(
			enabledRaw,
		)
		if err != nil {
			return semanticConfig{},
				fmt.Errorf(
					"invalid SEARCH_SEMANTIC_ENABLED: %w",
					err,
				)
		}

		enabled = parsed
	}

	cfg := semanticConfig{
		Enabled: enabled,

		APIKey: strings.TrimSpace(
			os.Getenv(
				"VOYAGE_API_KEY",
			),
		),

		Model: strings.TrimSpace(
			os.Getenv(
				"SEARCH_EMBEDDING_MODEL",
			),
		),

		PrimaryMin: defaultSemanticPrimaryMin,

		SimilarMin: defaultSemanticSimilarMin,

		Timeout: defaultEmbeddingTimeout,
	}

	if cfg.Model == "" {
		cfg.Model =
			defaultEmbeddingModel
	}

	if value := strings.TrimSpace(
		os.Getenv(
			"SEARCH_SEMANTIC_PRIMARY_MIN",
		),
	); value != "" {
		parsed, err := strconv.ParseFloat(
			value,
			64,
		)
		if err != nil {
			return semanticConfig{},
				fmt.Errorf(
					"invalid SEARCH_SEMANTIC_PRIMARY_MIN: %w",
					err,
				)
		}

		cfg.PrimaryMin = parsed
	}

	if value := strings.TrimSpace(
		os.Getenv(
			"SEARCH_SEMANTIC_SIMILAR_MIN",
		),
	); value != "" {
		parsed, err := strconv.ParseFloat(
			value,
			64,
		)
		if err != nil {
			return semanticConfig{},
				fmt.Errorf(
					"invalid SEARCH_SEMANTIC_SIMILAR_MIN: %w",
					err,
				)
		}

		cfg.SimilarMin = parsed
	}

	if value := strings.TrimSpace(
		os.Getenv(
			"SEARCH_EMBEDDING_TIMEOUT",
		),
	); value != "" {
		parsed, err := time.ParseDuration(
			value,
		)
		if err != nil {
			return semanticConfig{},
				fmt.Errorf(
					"invalid SEARCH_EMBEDDING_TIMEOUT: %w",
					err,
				)
		}

		cfg.Timeout = parsed
	}

	if cfg.Timeout <= 0 {
		return semanticConfig{},
			fmt.Errorf(
				"SEARCH_EMBEDDING_TIMEOUT must be greater than zero",
			)
	}

	if cfg.SimilarMin < 0 ||
		cfg.SimilarMin > 1 {
		return semanticConfig{},
			fmt.Errorf(
				"SEARCH_SEMANTIC_SIMILAR_MIN must be between 0 and 1",
			)
	}

	if cfg.PrimaryMin < 0 ||
		cfg.PrimaryMin > 1 {
		return semanticConfig{},
			fmt.Errorf(
				"SEARCH_SEMANTIC_PRIMARY_MIN must be between 0 and 1",
			)
	}

	if cfg.PrimaryMin <=
		cfg.SimilarMin {
		return semanticConfig{},
			fmt.Errorf(
				"SEARCH_SEMANTIC_PRIMARY_MIN must be greater than SEARCH_SEMANTIC_SIMILAR_MIN",
			)
	}

	if cfg.Enabled &&
		cfg.APIKey == "" {
		return semanticConfig{},
			fmt.Errorf(
				"VOYAGE_API_KEY is required when SEARCH_SEMANTIC_ENABLED=true",
			)
	}

	return cfg, nil
}

func shouldTrySemantic(
	query string,
) bool {
	query =
		strings.TrimSpace(
			query,
		)

	if query == "" {
		return false
	}

	for _, character := range query {
		if unicode.IsDigit(
			character,
		) {
			return false
		}
	}

	if strings.Contains(
		query,
		"-",
	) {
		return false
	}

	return true
}

func validateEmbedding(
	embedding []float32,
) error {
	if len(embedding) !=
		searchEmbeddingDimensions {
		return fmt.Errorf(
			"expected %d dimensions, got %d",
			searchEmbeddingDimensions,
			len(embedding),
		)
	}

	for index, value := range embedding {
		floatValue :=
			float64(
				value,
			)

		if math.IsNaN(
			floatValue,
		) ||
			math.IsInf(
				floatValue,
				0,
			) {
			return fmt.Errorf(
				"dimension %d is not finite",
				index,
			)
		}
	}

	return nil
}

func vectorLiteral(
	embedding []float32,
) (string, error) {
	if err :=
		validateEmbedding(
			embedding,
		); err != nil {
		return "", err
	}

	var builder strings.Builder

	builder.Grow(
		len(embedding) * 12,
	)

	builder.WriteByte(
		'[',
	)

	for index, value := range embedding {
		if index > 0 {
			builder.WriteByte(
				',',
			)
		}

		builder.WriteString(
			strconv.FormatFloat(
				float64(
					value,
				),
				'g',
				-1,
				32,
			),
		)
	}

	builder.WriteByte(
		']',
	)

	return builder.String(),
		nil
}

func isSupportedSearchImageMediaType(
	mediaType string,
) bool {
	switch strings.ToLower(
		strings.TrimSpace(
			mediaType,
		),
	) {
	case "image/jpeg",
		"image/png",
		"image/webp",
		"image/gif":
		return true

	default:
		return false
	}
}

func normalizedRemoteImageURL(
	raw string,
) string {
	raw =
		strings.TrimSpace(
			raw,
		)

	if raw == "" {
		return ""
	}

	parsed, err :=
		url.Parse(
			raw,
		)
	if err != nil ||
		parsed.Host == "" {
		return ""
	}

	switch strings.ToLower(
		parsed.Scheme,
	) {
	case "http",
		"https":
		return parsed.String()

	default:
		return ""
	}
}

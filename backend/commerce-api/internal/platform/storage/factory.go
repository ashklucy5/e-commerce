package storage

import (
	"fmt"

	"project.local/commerce-api/internal/platform/config"
)

func NewFromConfig(
	cfg config.StorageConfig,
) (*Gateway, error) {
	switch cfg.Provider {
	case "disabled":
		return NewGateway(
			NewDisabledProvider(),
		), nil

	case "s3":
		provider, err :=
			NewS3Provider(
				cfg,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"create S3 storage provider: %w",
					err,
				)
		}

		return NewGateway(
			newS3HTTPProvider(
				provider,
			),
		), nil

	default:
		return nil,
			fmt.Errorf(
				"unsupported storage provider %q",
				cfg.Provider,
			)
	}
}

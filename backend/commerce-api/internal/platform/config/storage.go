package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type StorageConfig struct {
	Provider string

	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string

	ForcePathStyle bool

	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration
}

func LoadStorage() (StorageConfig, error) {
	cfg := StorageConfig{
		Provider: strings.ToLower(
			envOrDefault(
				"STORAGE_PROVIDER",
				"disabled",
			),
		),

		Endpoint: strings.TrimSpace(
			os.Getenv(
				"STORAGE_ENDPOINT",
			),
		),

		Region: envOrDefault(
			"STORAGE_REGION",
			"auto",
		),

		Bucket: strings.TrimSpace(
			os.Getenv(
				"STORAGE_BUCKET",
			),
		),

		AccessKey: strings.TrimSpace(
			os.Getenv(
				"STORAGE_ACCESS_KEY",
			),
		),

		SecretKey: strings.TrimSpace(
			os.Getenv(
				"STORAGE_SECRET_KEY",
			),
		),

		PublicBaseURL: strings.TrimRight(
			strings.TrimSpace(
				os.Getenv(
					"STORAGE_PUBLIC_BASE_URL",
				),
			),
			"/",
		),

		UploadURLTTL: envDurationOrDefault(
			"STORAGE_UPLOAD_URL_TTL",
			15*time.Minute,
		),

		DownloadURLTTL: envDurationOrDefault(
			"STORAGE_DOWNLOAD_URL_TTL",
			15*time.Minute,
		),
	}

	forcePathStyle, err :=
		envBoolOrDefault(
			"STORAGE_FORCE_PATH_STYLE",
			false,
		)
	if err != nil {
		return StorageConfig{},
			err
	}

	cfg.ForcePathStyle = forcePathStyle

	switch cfg.Provider {
	case "disabled":
		return cfg, nil

	case "s3":
		if err := validateS3Storage(
			cfg,
		); err != nil {
			return StorageConfig{},
				err
		}

		return cfg, nil

	default:
		return StorageConfig{},
			fmt.Errorf(
				"unsupported STORAGE_PROVIDER %q",
				cfg.Provider,
			)
	}
}

func validateS3Storage(
	cfg StorageConfig,
) error {
	required := map[string]string{
		"STORAGE_ENDPOINT":   cfg.Endpoint,
		"STORAGE_BUCKET":     cfg.Bucket,
		"STORAGE_ACCESS_KEY": cfg.AccessKey,
		"STORAGE_SECRET_KEY": cfg.SecretKey,
	}

	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf(
				"%s is required when STORAGE_PROVIDER=s3",
				name,
			)
		}
	}

	if cfg.UploadURLTTL <= 0 {
		return fmt.Errorf(
			"STORAGE_UPLOAD_URL_TTL must be greater than zero",
		)
	}

	if cfg.DownloadURLTTL <= 0 {
		return fmt.Errorf(
			"STORAGE_DOWNLOAD_URL_TTL must be greater than zero",
		)
	}

	return nil
}

func envOrDefault(
	name string,
	fallback string,
) string {
	value := strings.TrimSpace(
		os.Getenv(name),
	)

	if value == "" {
		return fallback
	}

	return value
}

func envDurationOrDefault(
	name string,
	fallback time.Duration,
) time.Duration {
	value := strings.TrimSpace(
		os.Getenv(name),
	)

	if value == "" {
		return fallback
	}

	duration, err :=
		time.ParseDuration(value)

	if err != nil {
		return fallback
	}

	return duration
}

func envBoolOrDefault(
	name string,
	fallback bool,
) (bool, error) {
	value := strings.TrimSpace(
		os.Getenv(name),
	)

	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(
		value,
	)
	if err != nil {
		return false,
			fmt.Errorf(
				"%s must be true or false",
				name,
			)
	}

	return parsed, nil
}

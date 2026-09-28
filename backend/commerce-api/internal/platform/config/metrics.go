package config

import (
	"fmt"
	"os"
	"strings"
)

const DefaultMetricsPath = "/internal/metrics"

type MetricsConfig struct {
	Enabled bool

	Path string

	BearerToken string
}

func LoadMetricsConfig(
	appEnv string,
) (
	MetricsConfig,
	error,
) {
	isProduction :=
		strings.EqualFold(
			strings.TrimSpace(
				appEnv,
			),
			"production",
		)

	// Development gets metrics automatically.
	//
	// Production requires explicit opt-in so the endpoint cannot
	// accidentally become public during a deployment.
	defaultEnabled :=
		!isProduction

	enabled, err :=
		getBoolEnv(
			"METRICS_ENABLED",
			defaultEnabled,
		)
	if err != nil {
		return MetricsConfig{}, err
	}

	metricsPath :=
		strings.TrimSpace(
			getEnv(
				"METRICS_PATH",
				DefaultMetricsPath,
			),
		)

	if metricsPath == "" ||
		!strings.HasPrefix(
			metricsPath,
			"/",
		) ||
		metricsPath == "/" ||
		strings.ContainsAny(
			metricsPath,
			"?#",
		) {

		return MetricsConfig{},
			fmt.Errorf(
				"METRICS_PATH must be a static absolute HTTP path",
			)
	}

	if len(metricsPath) > 128 {
		return MetricsConfig{},
			fmt.Errorf(
				"METRICS_PATH cannot exceed 128 characters",
			)
	}

	if strings.HasSuffix(
		metricsPath,
		"/",
	) {
		metricsPath =
			strings.TrimRight(
				metricsPath,
				"/",
			)
	}

	bearerToken :=
		strings.TrimSpace(
			os.Getenv(
				"METRICS_TOKEN",
			),
		)

	if enabled &&
		isProduction &&
		len(bearerToken) < 32 {

		return MetricsConfig{},
			fmt.Errorf(
				"METRICS_TOKEN must contain at least 32 characters when production metrics are enabled",
			)
	}

	return MetricsConfig{
			Enabled: enabled,

			Path: metricsPath,

			BearerToken: bearerToken,
		},
		nil
}

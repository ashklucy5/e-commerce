package config

import (
	"fmt"
	"os"
	"strings"
)

// LoadAppAllowedOrigins loads browser origins allowed to call the
// customer-facing API. Admin/staff origins are configured separately.
func LoadAppAllowedOrigins(
	appEnv string,
) ([]string, error) {
	origins :=
		splitCSV(
			os.Getenv(
				"APP_ALLOWED_ORIGINS",
			),
		)

	production :=
		strings.EqualFold(
			strings.TrimSpace(
				appEnv,
			),
			"production",
		)

	if len(
		origins,
	) == 0 {
		if production {
			return nil,
				fmt.Errorf(
					"APP_ALLOWED_ORIGINS is required in production",
				)
		}

		origins =
			[]string{
				"http://localhost:3000",
				"http://127.0.0.1:3000",
			}
	}

	for _, origin := range origins {

		if origin == "*" {
			return nil,
				fmt.Errorf(
					"APP_ALLOWED_ORIGINS cannot contain wildcard origin",
				)
		}
	}

	return origins, nil
}

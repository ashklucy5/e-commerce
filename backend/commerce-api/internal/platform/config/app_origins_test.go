package config

import "testing"

func TestLoadAppAllowedOrigins(
	t *testing.T,
) {
	t.Run(
		"development defaults",
		func(
			t *testing.T,
		) {
			t.Setenv(
				"APP_ALLOWED_ORIGINS",
				"",
			)

			origins, err :=
				LoadAppAllowedOrigins(
					"development",
				)
			if err != nil {
				t.Fatal(
					err,
				)
			}

			if len(
				origins,
			) != 2 {
				t.Fatalf(
					"expected 2 origins, got %d",
					len(
						origins,
					),
				)
			}
		},
	)

	t.Run(
		"production requires explicit origins",
		func(
			t *testing.T,
		) {
			t.Setenv(
				"APP_ALLOWED_ORIGINS",
				"",
			)

			if _, err :=
				LoadAppAllowedOrigins(
					"production",
				); err == nil {

				t.Fatal(
					"expected production configuration error",
				)
			}
		},
	)

	t.Run(
		"wildcard rejected",
		func(
			t *testing.T,
		) {
			t.Setenv(
				"APP_ALLOWED_ORIGINS",
				"*",
			)

			if _, err :=
				LoadAppAllowedOrigins(
					"development",
				); err == nil {

				t.Fatal(
					"expected wildcard configuration error",
				)
			}
		},
	)
}

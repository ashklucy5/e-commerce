package cache

import (
	"strings"

	"project.local/commerce-api/internal/platform/security"
)

const namespace = "commerce:v1"

func Key(
	parts ...string,
) string {
	clean :=
		make(
			[]string,
			0,
			len(parts)+1,
		)

	clean =
		append(
			clean,
			namespace,
		)

	for _, part := range parts {

		part =
			strings.TrimSpace(
				part,
			)

		if part == "" {
			continue
		}

		clean =
			append(
				clean,
				part,
			)
	}

	return strings.Join(
		clean,
		":",
	)
}

func HashedKey(
	scope string,
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	return Key(
		scope,
		security.SHA256String(
			value,
		),
	)
}

func PublicCategoriesKey() string {
	return Key(
		"categories",
		"active",
	)
}

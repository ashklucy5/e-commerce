package catalogwrite

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
)

func ensureCategoryPath(
	ctx context.Context,
	tx pgx.Tx,
	path string,
) (string, error) {
	path = normalizeCategoryPath(path)

	parts := strings.Split(
		path,
		" > ",
	)

	var parentID *string
	var currentID string

	for index, name := range parts {
		name = strings.TrimSpace(name)

		if name == "" {
			continue
		}

		id, found, err := findCategoryChild(
			ctx,
			tx,
			parentID,
			name,
		)
		if err != nil {
			return "", err
		}

		if found {
			currentID = id
			parentID = &currentID

			continue
		}

		fullPath := strings.Join(
			parts[:index+1],
			" > ",
		)

		slugBase := slugify(
			strings.Join(
				parts[:index+1],
				"-",
			),
		)

		if slugBase == "" {
			slugBase = "category"
		}

		categorySlug, err :=
			uniqueCategorySlug(
				ctx,
				tx,
				slugBase,
			)
		if err != nil {
			return "", err
		}

		err = tx.QueryRow(
			ctx,
			`
				INSERT INTO categories (
					parent_id,
					name,
					slug,
					sort_order,
					is_active,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					0,
					true,
					now(),
					now()
				)
				RETURNING id::text
			`,
			parentID,
			name,
			categorySlug,
		).Scan(
			&currentID,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"create category %q: %w",
					fullPath,
					err,
				)
		}

		parentID = &currentID
	}

	if currentID == "" {
		return "",
			fmt.Errorf(
				"category path is empty",
			)
	}

	return currentID, nil
}

func findCategoryChild(
	ctx context.Context,
	tx pgx.Tx,
	parentID *string,
	name string,
) (string, bool, error) {
	var id string
	var err error

	if parentID == nil {
		err = tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM categories
				WHERE
					parent_id IS NULL
					AND lower(trim(name)) =
						lower(trim($1::text))
				LIMIT 1
			`,
			name,
		).Scan(
			&id,
		)
	} else {
		err = tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM categories
				WHERE
					parent_id = $1
					AND lower(trim(name)) =
						lower(trim($2::text))
				LIMIT 1
			`,
			*parentID,
			name,
		).Scan(
			&id,
		)
	}

	if err == pgx.ErrNoRows {
		return "", false, nil
	}

	if err != nil {
		return "",
			false,
			fmt.Errorf(
				"find category %q: %w",
				name,
				err,
			)
	}

	return id, true, nil
}

func uniqueCategorySlug(
	ctx context.Context,
	tx pgx.Tx,
	base string,
) (string, error) {
	base = strings.Trim(
		base,
		"-",
	)

	if base == "" {
		base = "category"
	}

	for suffix := 1; suffix <= 10000; suffix++ {
		candidate := base

		if suffix > 1 {
			candidate = fmt.Sprintf(
				"%s-%d",
				base,
				suffix,
			)
		}

		var exists bool

		err := tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM categories
					WHERE slug = $1
				)
			`,
			candidate,
		).Scan(
			&exists,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"check category slug %q: %w",
					candidate,
					err,
				)
		}

		if !exists {
			return candidate, nil
		}
	}

	return "",
		fmt.Errorf(
			"could not generate a unique category slug for %q",
			base,
		)
}

func uniqueProductSlug(
	ctx context.Context,
	tx pgx.Tx,
	base string,
) (string, error) {
	base = strings.Trim(
		base,
		"-",
	)

	if base == "" {
		base = "product"
	}

	for suffix := 1; suffix <= 10000; suffix++ {
		candidate := base

		if suffix > 1 {
			candidate = fmt.Sprintf(
				"%s-%d",
				base,
				suffix,
			)
		}

		var exists bool

		err := tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM products
					WHERE slug = $1
				)
			`,
			candidate,
		).Scan(
			&exists,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"check product slug %q: %w",
					candidate,
					err,
				)
		}

		if !exists {
			return candidate, nil
		}
	}

	return "",
		fmt.Errorf(
			"could not generate a unique product slug for %q",
			base,
		)
}

func slugify(
	value string,
) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	var builder strings.Builder
	needsDash := false

	for _, r := range value {
		if unicode.IsLetter(r) ||
			unicode.IsDigit(r) {

			if needsDash &&
				builder.Len() > 0 {

				builder.WriteByte('-')
			}

			builder.WriteRune(r)
			needsDash = false

			continue
		}

		if builder.Len() > 0 {
			needsDash = true
		}
	}

	return strings.Trim(
		builder.String(),
		"-",
	)
}

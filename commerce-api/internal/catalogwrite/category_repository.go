package catalogwrite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) ListAdminCategories(
	ctx context.Context,
) ([]AdminCategory, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				c.id::text,
				c.parent_id::text,
				c.name,
				c.slug,
				c.description,
				c.sort_order,
				c.is_active,
				n.prefix
			FROM categories c
			LEFT JOIN product_code_namespaces n
				ON n.category_id = c.id
			ORDER BY
				c.sort_order ASC,
				c.name ASC,
				c.id ASC
		`,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"query Admin categories: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]AdminCategory,
		0,
	)

	for rows.Next() {
		var item AdminCategory

		if err := rows.Scan(
			&item.ID,
			&item.ParentID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.SortOrder,
			&item.IsActive,
			&item.ProductCodePrefix,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan Admin category: %w",
					err,
				)
		}

		item.ProductCodeReady =
			item.ProductCodePrefix != nil

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate Admin categories: %w",
				err,
			)
	}

	return items, nil
}

func (r *Repository) CreateCategory(
	ctx context.Context,
	input CreateCategoryInput,
) (AdminCategory, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return AdminCategory{},
			fmt.Errorf(
				"begin category transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if input.ParentID != nil {
		exists, err := categoryExistsByID(
			ctx,
			tx,
			*input.ParentID,
		)
		if err != nil {
			return AdminCategory{}, err
		}

		if !exists {
			return AdminCategory{},
				ErrCategoryParentNotFound
		}
	}

	lockParent := "root"

	if input.ParentID != nil {
		lockParent = *input.ParentID
	}

	_, err = tx.Exec(
		ctx,
		`
			SELECT pg_advisory_xact_lock(
				hashtextextended($1, 0)
			)
		`,
		strings.ToLower(
			strings.TrimSpace(
				lockParent+
					"|"+
					input.Name,
			),
		),
	)
	if err != nil {
		return AdminCategory{},
			fmt.Errorf(
				"lock category name: %w",
				err,
			)
	}

	duplicate, err := categoryNameExists(
		ctx,
		tx,
		input.ParentID,
		input.Name,
	)
	if err != nil {
		return AdminCategory{}, err
	}

	if duplicate {
		return AdminCategory{},
			ErrCategoryAlreadyExists
	}

	slugBase := input.Slug

	if slugBase == "" {
		slugBase = input.Name
	}

	slug, err := uniqueCategorySlug(
		ctx,
		tx,
		slugify(slugBase),
	)
	if err != nil {
		return AdminCategory{}, err
	}

	var created AdminCategory

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO categories (
				parent_id,
				name,
				slug,
				description,
				sort_order,
				is_active,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				NULLIF($4, ''),
				$5,
				$6,
				now(),
				now()
			)
			RETURNING
				id::text,
				parent_id::text,
				name,
				slug,
				description,
				sort_order,
				is_active
		`,
		input.ParentID,
		input.Name,
		slug,
		input.Description,
		input.SortOrder,
		input.IsActive,
	).Scan(
		&created.ID,
		&created.ParentID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SortOrder,
		&created.IsActive,
	)
	if err != nil {
		return AdminCategory{},
			fmt.Errorf(
				"create category: %w",
				err,
			)
	}

	if input.ProductCodePrefix != "" {
		_, err = tx.Exec(
			ctx,
			`
				INSERT INTO product_code_namespaces (
					category_id,
					prefix,
					next_number,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					1,
					now(),
					now()
				)
			`,
			created.ID,
			input.ProductCodePrefix,
		)
		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(
				err,
				&pgErr,
			) &&
				pgErr.Code == "23505" {
				return AdminCategory{},
					ErrProductCodePrefixInUse
			}

			return AdminCategory{},
				fmt.Errorf(
					"create product-code namespace: %w",
					err,
				)
		}

		prefix := input.ProductCodePrefix

		created.ProductCodePrefix =
			&prefix

		created.ProductCodeReady =
			true
	}

	if err := tx.Commit(ctx); err != nil {
		return AdminCategory{},
			fmt.Errorf(
				"commit category transaction: %w",
				err,
			)
	}

	return created, nil
}

func categoryExistsByID(
	ctx context.Context,
	tx pgx.Tx,
	categoryID string,
) (bool, error) {
	var exists bool

	err := tx.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM categories
				WHERE id::text = $1
			)
		`,
		categoryID,
	).Scan(
		&exists,
	)
	if err != nil {
		return false,
			fmt.Errorf(
				"check category parent: %w",
				err,
			)
	}

	return exists, nil
}

func categoryNameExists(
	ctx context.Context,
	tx pgx.Tx,
	parentID *string,
	name string,
) (bool, error) {
	var exists bool
	var err error

	if parentID == nil {
		err = tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM categories
					WHERE
						parent_id IS NULL
						AND lower(trim(name)) =
							lower(trim($1::text))
				)
			`,
			name,
		).Scan(
			&exists,
		)
	} else {
		err = tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM categories
					WHERE
						parent_id::text = $1
						AND lower(trim(name)) =
							lower(trim($2::text))
				)
			`,
			*parentID,
			name,
		).Scan(
			&exists,
		)
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"check duplicate category: %w",
				err,
			)
	}

	return exists, nil
}

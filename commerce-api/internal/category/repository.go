package category

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for categories.
//
// Higher layers should not need to know SQL.
// They ask the repository for category data instead.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a category repository using
// the application's PostgreSQL connection pool.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

// ListActive returns all active categories.
//
// Parent categories are listed before child categories where
// possible by sorting on parent_id and sort_order.
func (r *Repository) ListActive(
	ctx context.Context,
) ([]Category, error) {
	const query = `
		SELECT
			id,
			parent_id,
			name,
			slug,
			description,
			image_url,
			icon_url,
			sort_order,
			is_active,
			created_at,
			updated_at
		FROM categories
		WHERE is_active = true
		ORDER BY
			sort_order ASC,
			name ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query active categories: %w",
			err,
		)
	}
	defer rows.Close()

	categories := make(
		[]Category,
		0,
	)

	for rows.Next() {
		var item Category

		err := rows.Scan(
			&item.ID,
			&item.ParentID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.ImageURL,
			&item.IconURL,
			&item.SortOrder,
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan category: %w",
				err,
			)
		}

		categories = append(
			categories,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate categories: %w",
			err,
		)
	}

	return categories, nil
}

// FindActiveBySlug returns one active category by its slug.
//
// pgx.ErrNoRows is intentionally preserved so the service layer
// can later translate it into a category-not-found error.
func (r *Repository) FindActiveBySlug(
	ctx context.Context,
	slug string,
) (Category, error) {
	const query = `
		SELECT
			id,
			parent_id,
			name,
			slug,
			description,
			image_url,
			icon_url,
			sort_order,
			is_active,
			created_at,
			updated_at
		FROM categories
		WHERE
			slug = $1
			AND is_active = true
		LIMIT 1
	`

	var item Category

	err := r.db.QueryRow(
		ctx,
		query,
		slug,
	).Scan(
		&item.ID,
		&item.ParentID,
		&item.Name,
		&item.Slug,
		&item.Description,
		&item.ImageURL,
		&item.IconURL,
		&item.SortOrder,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Category{}, pgx.ErrNoRows
		}

		return Category{}, fmt.Errorf(
			"find active category by slug: %w",
			err,
		)
	}

	return item, nil
}

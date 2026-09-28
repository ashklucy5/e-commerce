package catalogmediawrite

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateImage(
	ctx context.Context,
	input CreateImageInput,
	publicURL string,
) (ImageResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ImageResult{},
			fmt.Errorf(
				"begin product image transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if input.IsPrimary {
		_, err = tx.Exec(
			ctx,
			`
				UPDATE product_images
				SET is_primary = false
				WHERE
					product_id = $1
					AND is_primary = true
			`,
			input.ProductID,
		)
		if err != nil {
			return ImageResult{},
				fmt.Errorf(
					"clear existing primary product image: %w",
					err,
				)
		}
	}

	var result ImageResult
	var variantID pgtype.Text
	var altText pgtype.Text

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO product_images (
				product_id,
				variant_id,
				url,
				alt_text,
				sort_order,
				is_primary,
				created_at
			)
			VALUES (
				$1,
				NULLIF($2, '')::uuid,
				$3,
				NULLIF($4, ''),
				$5,
				$6,
				now()
			)
			RETURNING
				id::text,
				product_id::text,
				variant_id::text,
				url,
				alt_text,
				sort_order,
				is_primary
		`,
		input.ProductID,
		input.VariantID,
		publicURL,
		input.AltText,
		input.SortOrder,
		input.IsPrimary,
	).Scan(
		&result.ID,
		&result.ProductID,
		&variantID,
		&result.URL,
		&altText,
		&result.SortOrder,
		&result.IsPrimary,
	)
	if err != nil {
		return ImageResult{},
			fmt.Errorf(
				"create product image: %w",
				err,
			)
	}

	if variantID.Valid {
		value := variantID.String
		result.VariantID = &value
	}

	if altText.Valid {
		value := altText.String
		result.AltText = &value
	}

	if err := tx.Commit(ctx); err != nil {
		return ImageResult{},
			fmt.Errorf(
				"commit product image transaction: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) Upsert360Frame(
	ctx context.Context,
	input Create360FrameInput,
	publicURL string,
) (Frame360Result, error) {
	var result Frame360Result
	var variantID pgtype.Text

	var err error

	if input.VariantID == "" {
		err = r.db.QueryRow(
			ctx,
			`
				INSERT INTO product_360_frames (
					product_id,
					variant_id,
					storage_key,
					url,
					frame_index
				)
				VALUES (
					$1,
					NULL,
					$2,
					$3,
					$4
				)
				ON CONFLICT (
					product_id,
					frame_index
				)
				WHERE variant_id IS NULL
				DO UPDATE SET
					storage_key = EXCLUDED.storage_key,
					url = EXCLUDED.url
				RETURNING
					id::text,
					product_id::text,
					variant_id::text,
					storage_key,
					url,
					frame_index
			`,
			input.ProductID,
			input.StorageKey,
			publicURL,
			input.FrameIndex,
		).Scan(
			&result.ID,
			&result.ProductID,
			&variantID,
			&result.StorageKey,
			&result.URL,
			&result.FrameIndex,
		)
	} else {
		err = r.db.QueryRow(
			ctx,
			`
				INSERT INTO product_360_frames (
					product_id,
					variant_id,
					storage_key,
					url,
					frame_index
				)
				VALUES (
					$1,
					$2::uuid,
					$3,
					$4,
					$5
				)
				ON CONFLICT (
					product_id,
					variant_id,
					frame_index
				)
				WHERE variant_id IS NOT NULL
				DO UPDATE SET
					storage_key = EXCLUDED.storage_key,
					url = EXCLUDED.url
				RETURNING
					id::text,
					product_id::text,
					variant_id::text,
					storage_key,
					url,
					frame_index
			`,
			input.ProductID,
			input.VariantID,
			input.StorageKey,
			publicURL,
			input.FrameIndex,
		).Scan(
			&result.ID,
			&result.ProductID,
			&variantID,
			&result.StorageKey,
			&result.URL,
			&result.FrameIndex,
		)
	}

	if err != nil {
		return Frame360Result{},
			fmt.Errorf(
				"upsert product 360 frame: %w",
				err,
			)
	}

	if variantID.Valid {
		value := variantID.String
		result.VariantID = &value
	}

	return result, nil
}

func (r *Repository) Upsert3DModel(
	ctx context.Context,
	input Upsert3DModelInput,
	publicURL string,
) (Model3DResult, error) {
	var result Model3DResult
	var variantID pgtype.Text
	var posterURL pgtype.Text

	var err error

	if input.VariantID == "" {
		err = r.db.QueryRow(
			ctx,
			`
				INSERT INTO product_3d_models (
					product_id,
					variant_id,
					storage_key,
					url,
					poster_url
				)
				VALUES (
					$1,
					NULL,
					$2,
					$3,
					NULLIF($4, '')
				)
				ON CONFLICT (product_id)
				WHERE variant_id IS NULL
				DO UPDATE SET
					storage_key = EXCLUDED.storage_key,
					url = EXCLUDED.url,
					poster_url = EXCLUDED.poster_url,
					updated_at = now()
				RETURNING
					id::text,
					product_id::text,
					variant_id::text,
					storage_key,
					url,
					poster_url
			`,
			input.ProductID,
			input.StorageKey,
			publicURL,
			input.PosterURL,
		).Scan(
			&result.ID,
			&result.ProductID,
			&variantID,
			&result.StorageKey,
			&result.URL,
			&posterURL,
		)
	} else {
		err = r.db.QueryRow(
			ctx,
			`
				INSERT INTO product_3d_models (
					product_id,
					variant_id,
					storage_key,
					url,
					poster_url
				)
				VALUES (
					$1,
					$2::uuid,
					$3,
					$4,
					NULLIF($5, '')
				)
				ON CONFLICT (
					product_id,
					variant_id
				)
				WHERE variant_id IS NOT NULL
				DO UPDATE SET
					storage_key = EXCLUDED.storage_key,
					url = EXCLUDED.url,
					poster_url = EXCLUDED.poster_url,
					updated_at = now()
				RETURNING
					id::text,
					product_id::text,
					variant_id::text,
					storage_key,
					url,
					poster_url
			`,
			input.ProductID,
			input.VariantID,
			input.StorageKey,
			publicURL,
			input.PosterURL,
		).Scan(
			&result.ID,
			&result.ProductID,
			&variantID,
			&result.StorageKey,
			&result.URL,
			&posterURL,
		)
	}

	if err != nil {
		return Model3DResult{},
			fmt.Errorf(
				"upsert product 3D model: %w",
				err,
			)
	}

	if variantID.Valid {
		value := variantID.String
		result.VariantID = &value
	}

	if posterURL.Valid {
		value := posterURL.String
		result.PosterURL = &value
	}

	return result, nil
}

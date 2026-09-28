package cart

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) Begin(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err := r.db.Begin(
		ctx,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin cart transaction: %w",
				err,
			)
	}

	return tx, nil
}

func (r *Repository) CreateCart(
	ctx context.Context,
	cartKey string,
	currency string,
	expiresAt time.Time,
) (Cart, error) {
	row := r.db.QueryRow(
		ctx,
		`
			INSERT INTO carts (
				cart_key,
				status,
				currency,
				expires_at,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				'active',
				$2,
				$3,
				now(),
				now()
			)
			RETURNING
				id::text,
				cart_key,
				status,
				currency,
				expires_at,
				created_at,
				updated_at
		`,
		cartKey,
		currency,
		expiresAt,
	)

	result, err :=
		scanCart(
			row,
		)
	if err != nil {
		return Cart{},
			fmt.Errorf(
				"create cart: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) GetCart(
	ctx context.Context,
	cartKey string,
) (Cart, error) {
	row := r.db.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				cart_key,
				status,
				currency,
				expires_at,
				created_at,
				updated_at
			FROM carts
			WHERE cart_key = $1
		`,
		cartKey,
	)

	result, err :=
		scanCart(
			row,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Cart{},
				ErrCartNotFound
		}

		return Cart{},
			fmt.Errorf(
				"get cart: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) LockCart(
	ctx context.Context,
	tx pgx.Tx,
	cartKey string,
) (Cart, error) {
	row := tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				cart_key,
				status,
				currency,
				expires_at,
				created_at,
				updated_at
			FROM carts
			WHERE cart_key = $1
			FOR UPDATE
		`,
		cartKey,
	)

	result, err :=
		scanCart(
			row,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Cart{},
				ErrCartNotFound
		}

		return Cart{},
			fmt.Errorf(
				"lock cart: %w",
				err,
			)
	}

	return result, nil
}

func scanCart(
	row pgx.Row,
) (Cart, error) {
	var result Cart
	var expiresAt pgtype.Timestamptz

	err := row.Scan(
		&result.ID,
		&result.CartKey,
		&result.Status,
		&result.Currency,
		&expiresAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return Cart{},
			err
	}

	if expiresAt.Valid {
		value := expiresAt.Time
		result.ExpiresAt = &value
	}

	return result, nil
}

func (r *Repository) GetItems(
	ctx context.Context,
	cartID string,
) ([]Item, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				ci.id::text,
				v.id::text,
				p.id::text,
				v.sku,
				p.name,
				p.slug,
				COALESCE(v.color_name, ''),
				COALESCE(v.size, ''),
				COALESCE(image.url, ''),
				ci.quantity,
				v.minimum_order_quantity,
				v.order_increment,
				GREATEST(
					COALESCE(
						i.quantity_on_hand -
						i.quantity_reserved,
						0
					),
					0
				),
				COALESCE(
					price_tier.unit_price_amount,
					v.price_amount
				),
				v.currency,
				v.is_active,
				p.status = 'active',
				ci.created_at,
				ci.updated_at
			FROM cart_items ci
			JOIN product_variants v
				ON v.id = ci.variant_id
			JOIN products p
				ON p.id = v.product_id
			LEFT JOIN inventory i
				ON i.variant_id = v.id

			LEFT JOIN LATERAL (
				SELECT
					pt.unit_price_amount
				FROM product_variant_price_tiers pt
				WHERE
					pt.variant_id = v.id
					AND pt.min_quantity <= ci.quantity
				ORDER BY
					pt.min_quantity DESC
				LIMIT 1
			) price_tier
				ON true

			LEFT JOIN LATERAL (
				SELECT
					pi.url
				FROM product_images pi
				WHERE
					pi.product_id = p.id
					AND (
						pi.variant_id = v.id
						OR pi.variant_id IS NULL
					)
				ORDER BY
					CASE
						WHEN pi.variant_id = v.id
							THEN 0
						ELSE 1
					END,
					pi.is_primary DESC,
					pi.sort_order ASC,
					pi.created_at ASC
				LIMIT 1
			) image
				ON true

			WHERE ci.cart_id = $1::uuid
			ORDER BY ci.created_at ASC
		`,
		cartID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"get cart items: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]Item,
		0,
	)

	for rows.Next() {
		var item Item

		if err := rows.Scan(
			&item.ID,
			&item.VariantID,
			&item.ProductID,
			&item.SKU,
			&item.ProductName,
			&item.ProductSlug,
			&item.ColorName,
			&item.Size,
			&item.ImageURL,
			&item.Quantity,
			&item.MinimumOrderQuantity,
			&item.OrderIncrement,
			&item.AvailableQuantity,
			&item.UnitPriceAmount,
			&item.Currency,
			&item.variantActive,
			&item.productActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan cart item: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate cart items: %w",
				err,
			)
	}

	return items, nil
}

func (r *Repository) GetPurchasableVariant(
	ctx context.Context,
	tx pgx.Tx,
	variantID string,
) (
	purchasableVariant,
	error,
) {
	var result purchasableVariant

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				v.id::text,
				v.minimum_order_quantity,
				v.order_increment,
				GREATEST(
					COALESCE(
						i.quantity_on_hand -
						i.quantity_reserved,
						0
					),
					0
				),
				v.currency
			FROM product_variants v
			JOIN products p
				ON p.id = v.product_id
			LEFT JOIN inventory i
				ON i.variant_id = v.id
			WHERE
				v.id = $1::uuid
				AND v.is_active = true
				AND p.status = 'active'
		`,
		variantID,
	).Scan(
		&result.ID,
		&result.MinimumOrderQuantity,
		&result.OrderIncrement,
		&result.AvailableQuantity,
		&result.Currency,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return purchasableVariant{},
				ErrVariantUnavailable
		}

		return purchasableVariant{},
			fmt.Errorf(
				"get purchasable variant: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) UpsertItem(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
	variantID string,
	quantity int,
) error {
	_, err := tx.Exec(
		ctx,
		`
			INSERT INTO cart_items (
				cart_id,
				variant_id,
				quantity,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3,
				now(),
				now()
			)
			ON CONFLICT (
				cart_id,
				variant_id
			)
			DO UPDATE SET
				quantity =
					EXCLUDED.quantity,
				updated_at =
					now()
		`,
		cartID,
		variantID,
		quantity,
	)
	if err != nil {
		return fmt.Errorf(
			"upsert cart item: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) GetItemVariantID(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
	itemID string,
) (string, error) {
	var variantID string

	err := tx.QueryRow(
		ctx,
		`
			SELECT variant_id::text
			FROM cart_items
			WHERE
				id = $1::uuid
				AND cart_id = $2::uuid
		`,
		itemID,
		cartID,
	).Scan(
		&variantID,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return "",
				ErrItemNotFound
		}

		return "",
			fmt.Errorf(
				"get cart item: %w",
				err,
			)
	}

	return variantID, nil
}

func (r *Repository) UpdateItem(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
	itemID string,
	quantity int,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			UPDATE cart_items
			SET
				quantity = $3,
				updated_at = now()
			WHERE
				id = $1::uuid
				AND cart_id = $2::uuid
		`,
		itemID,
		cartID,
		quantity,
	)
	if err != nil {
		return fmt.Errorf(
			"update cart item: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}

	return nil
}

func (r *Repository) DeleteItem(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
	itemID string,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			DELETE FROM cart_items
			WHERE
				id = $1::uuid
				AND cart_id = $2::uuid
		`,
		itemID,
		cartID,
	)
	if err != nil {
		return fmt.Errorf(
			"delete cart item: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}

	return nil
}

func (r *Repository) TouchCart(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
) error {
	_, err := tx.Exec(
		ctx,
		`
			UPDATE carts
			SET updated_at = now()
			WHERE id = $1::uuid
		`,
		cartID,
	)
	if err != nil {
		return fmt.Errorf(
			"touch cart: %w",
			err,
		)
	}

	return nil
}

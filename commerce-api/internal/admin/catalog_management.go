package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

const adminEventCatalogProductUpdated = "catalog_product_updated"

var (
	ErrAdminCatalogProductNotFound  = errors.New("Admin catalog product not found")
	ErrAdminCatalogConflict         = errors.New("Admin catalog conflict")
	ErrInvalidAdminCatalogProduct   = errors.New("invalid Admin catalog product")
	ErrAdminCatalogInactiveCategory = errors.New("active products require an active category")
	ErrAdminCatalogNoActiveVariant  = errors.New("active products require at least one active variant")
)

type CatalogProductReadFilter struct {
	Status  string
	Query   string
	QueryID string
}

type CatalogProductListItem struct {
	ID string `json:"id"`

	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`

	ProductCode string `json:"product_code"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Brand       string `json:"brand,omitempty"`

	Status     string `json:"status"`
	IsFeatured bool   `json:"is_featured"`

	VariantCount       int64 `json:"variant_count"`
	ActiveVariantCount int64 `json:"active_variant_count"`
	AvailableStock     int64 `json:"available_stock"`

	MinimumActivePriceAmount *int64 `json:"minimum_active_price_amount,omitempty"`
	Currency                 string `json:"currency,omitempty"`

	PrimaryImageURL string `json:"primary_image_url,omitempty"`

	PublishedAt *time.Time `json:"published_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CatalogProductDetail struct {
	CatalogProductListItem

	ShortDescription string `json:"short_description,omitempty"`
	Description      string `json:"description,omitempty"`

	Variants json.RawMessage `json:"variants"`
	Images   json.RawMessage `json:"images"`
}

type CatalogProductListResult struct {
	Items []CatalogProductListItem
	Meta  platformpagination.Meta
}

type UpdateCatalogProductInput struct {
	CategoryID *string

	Name *string
	Slug *string

	Brand            *string
	ShortDescription *string
	Description      *string

	Status     *string
	IsFeatured *bool
}

func (s *Service) ListCatalogProducts(
	ctx context.Context,
	params platformpagination.Params,
	filter CatalogProductReadFilter,
) (
	CatalogProductListResult,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(
			ctx,
		)
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				SELECT
					p.id::text,
					p.category_id::text,
					c.name,
					p.product_code,
					p.name,
					p.slug,
					COALESCE(p.brand, ''),
					p.status,
					p.is_featured,

					(
						SELECT COUNT(*)::bigint
						FROM product_variants v
						WHERE v.product_id = p.id
					),

					(
						SELECT COUNT(*)::bigint
						FROM product_variants v
						WHERE
							v.product_id = p.id
							AND v.is_active = true
					),

					COALESCE(
						(
							SELECT
								SUM(
									GREATEST(
										i.quantity_on_hand -
											i.quantity_reserved,
										0
									)
								)::bigint
							FROM product_variants v
							JOIN inventory i
								ON i.variant_id = v.id
							WHERE v.product_id = p.id
						),
						0
					),

					price.price_amount,

					COALESCE(
						price.currency,
						''
					),

					COALESCE(
						image.url,
						''
					),

					p.published_at,
					p.created_at,
					p.updated_at,

					COUNT(*) OVER()::bigint

				FROM products p

				JOIN categories c
					ON c.id = p.category_id

				LEFT JOIN LATERAL (
					SELECT
						v.price_amount,
						v.currency
					FROM product_variants v
					WHERE
						v.product_id = p.id
						AND v.is_active = true
					ORDER BY
						v.price_amount,
						v.id
					LIMIT 1
				) price
					ON true

				LEFT JOIN LATERAL (
					SELECT
						pi.url
					FROM product_images pi
					WHERE
						pi.product_id = p.id
						AND pi.is_primary = true
					LIMIT 1
				) image
					ON true

				WHERE
					(
						$1 = ''
						OR p.status = $1
					)
					AND (
						$2 = ''
						OR p.id =
							NULLIF(
								$3,
								''
							)::uuid
						OR p.product_code = $2
						OR p.slug = $2
						OR EXISTS (
							SELECT 1
							FROM product_variants v
							WHERE
								v.product_id = p.id
								AND v.sku = $2
						)
					)

				ORDER BY
					p.updated_at DESC,
					p.id DESC

				LIMIT $4
				OFFSET $5
			`,
			filter.Status,
			filter.Query,
			filter.QueryID,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return CatalogProductListResult{},
			fmt.Errorf(
				"list Admin catalog products: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]CatalogProductListItem,
			0,
			params.Limit,
		)

	var total int64

	for rows.Next() {
		var item CatalogProductListItem

		var minimumPrice pgtype.Int8
		var publishedAt pgtype.Timestamptz

		if err :=
			rows.Scan(
				&item.ID,
				&item.CategoryID,
				&item.CategoryName,
				&item.ProductCode,
				&item.Name,
				&item.Slug,
				&item.Brand,
				&item.Status,
				&item.IsFeatured,
				&item.VariantCount,
				&item.ActiveVariantCount,
				&item.AvailableStock,
				&minimumPrice,
				&item.Currency,
				&item.PrimaryImageURL,
				&publishedAt,
				&item.CreatedAt,
				&item.UpdatedAt,
				&total,
			); err != nil {

			return CatalogProductListResult{},
				fmt.Errorf(
					"scan Admin catalog product: %w",
					err,
				)
		}

		if minimumPrice.Valid {
			value :=
				minimumPrice.Int64

			item.MinimumActivePriceAmount =
				&value
		}

		item.PublishedAt =
			adminTimePointer(
				publishedAt,
			)

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return CatalogProductListResult{},
			fmt.Errorf(
				"iterate Admin catalog products: %w",
				err,
			)
	}

	return CatalogProductListResult{
		Items: items,

		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetCatalogProduct(
	ctx context.Context,
	productID string,
) (
	CatalogProductDetail,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(
			ctx,
		)
	defer cancel()

	var result CatalogProductDetail

	var minimumPrice pgtype.Int8
	var publishedAt pgtype.Timestamptz

	var variants []byte
	var images []byte

	err :=
		s.db.QueryRow(
			queryCtx,
			`
				SELECT
					p.id::text,
					p.category_id::text,
					c.name,
					p.product_code,
					p.name,
					p.slug,
					COALESCE(p.brand, ''),
					p.status,
					p.is_featured,

					(
						SELECT COUNT(*)::bigint
						FROM product_variants v
						WHERE v.product_id = p.id
					),

					(
						SELECT COUNT(*)::bigint
						FROM product_variants v
						WHERE
							v.product_id = p.id
							AND v.is_active = true
					),

					COALESCE(
						(
							SELECT
								SUM(
									GREATEST(
										i.quantity_on_hand -
											i.quantity_reserved,
										0
									)
								)::bigint
							FROM product_variants v
							JOIN inventory i
								ON i.variant_id = v.id
							WHERE v.product_id = p.id
						),
						0
					),

					(
						SELECT MIN(
							v.price_amount
						)
						FROM product_variants v
						WHERE
							v.product_id = p.id
							AND v.is_active = true
					),

					COALESCE(
						(
							SELECT v.currency
							FROM product_variants v
							WHERE
								v.product_id = p.id
								AND v.is_active = true
							ORDER BY
								v.price_amount,
								v.id
							LIMIT 1
						),
						''
					),

					COALESCE(
						(
							SELECT pi.url
							FROM product_images pi
							WHERE
								pi.product_id = p.id
								AND pi.is_primary = true
							LIMIT 1
						),
						''
					),

					p.published_at,
					p.created_at,
					p.updated_at,

					COALESCE(
						p.short_description,
						''
					),

					COALESCE(
						p.description,
						''
					),

					COALESCE(
						(
							SELECT
								jsonb_agg(
									jsonb_build_object(
										'id',
										v.id,

										'sku',
										v.sku,

										'color_name',
										v.color_name,

										'color_hex',
										v.color_hex,

										'size',
										v.size,

										'minimum_order_quantity',
										v.minimum_order_quantity,

										'order_increment',
										v.order_increment,

										'price_amount',
										v.price_amount,

										'compare_at_price_amount',
										v.compare_at_price_amount,

										'cost_amount',
										v.cost_amount,

										'currency',
										v.currency,

										'barcode',
										v.barcode,

										'weight_grams',
										v.weight_grams,

										'is_active',
										v.is_active,

										'quantity_on_hand',
										COALESCE(
											i.quantity_on_hand,
											0
										),

										'quantity_reserved',
										COALESCE(
											i.quantity_reserved,
											0
										),

										'available_quantity',
										GREATEST(
											COALESCE(
												i.quantity_on_hand,
												0
											) -
											COALESCE(
												i.quantity_reserved,
												0
											),
											0
										),

										'reorder_level',
										COALESCE(
											i.reorder_level,
											0
										),

										'price_tiers',
										COALESCE(
											(
												SELECT
													jsonb_agg(
														jsonb_build_object(
															'id',
															t.id,

															'min_quantity',
															t.min_quantity,

															'unit_price_amount',
															t.unit_price_amount
														)
														ORDER BY
															t.min_quantity,
															t.id
													)
												FROM product_variant_price_tiers t
												WHERE t.variant_id = v.id
											),
											'[]'::jsonb
										)
									)
									ORDER BY
										v.created_at,
										v.id
								)
							FROM product_variants v
							LEFT JOIN inventory i
								ON i.variant_id = v.id
							WHERE v.product_id = p.id
						),
						'[]'::jsonb
					),

					COALESCE(
						(
							SELECT
								jsonb_agg(
									jsonb_build_object(
										'id',
										pi.id,

										'variant_id',
										pi.variant_id,

										'url',
										pi.url,

										'alt_text',
										pi.alt_text,

										'sort_order',
										pi.sort_order,

										'is_primary',
										pi.is_primary
									)
									ORDER BY
										pi.is_primary DESC,
										pi.sort_order,
										pi.id
								)
							FROM product_images pi
							WHERE pi.product_id = p.id
						),
						'[]'::jsonb
					)

				FROM products p

				JOIN categories c
					ON c.id = p.category_id

				WHERE p.id = $1::uuid
			`,
			productID,
		).Scan(
			&result.ID,
			&result.CategoryID,
			&result.CategoryName,
			&result.ProductCode,
			&result.Name,
			&result.Slug,
			&result.Brand,
			&result.Status,
			&result.IsFeatured,
			&result.VariantCount,
			&result.ActiveVariantCount,
			&result.AvailableStock,
			&minimumPrice,
			&result.Currency,
			&result.PrimaryImageURL,
			&publishedAt,
			&result.CreatedAt,
			&result.UpdatedAt,
			&result.ShortDescription,
			&result.Description,
			&variants,
			&images,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return CatalogProductDetail{},
			ErrAdminCatalogProductNotFound
	}

	if err != nil {

		return CatalogProductDetail{},
			fmt.Errorf(
				"get Admin catalog product: %w",
				err,
			)
	}

	if minimumPrice.Valid {
		value :=
			minimumPrice.Int64

		result.MinimumActivePriceAmount =
			&value
	}

	result.PublishedAt =
		adminTimePointer(
			publishedAt,
		)

	result.Variants =
		json.RawMessage(
			variants,
		)

	result.Images =
		json.RawMessage(
			images,
		)

	return result, nil
}

func (s *Service) UpdateCatalogProduct(
	ctx context.Context,
	productID string,
	input UpdateCatalogProductInput,
	metadata AdminActionMetadata,
) (
	CatalogProductDetail,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(
			ctx,
		)
	defer cancel()

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var current CatalogProductDetail

				err :=
					tx.QueryRow(
						ctx,
						`
							SELECT
								category_id::text,
								product_code,
								name,
								slug,
								COALESCE(brand, ''),
								COALESCE(short_description, ''),
								COALESCE(description, ''),
								status,
								is_featured
							FROM products
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						productID,
					).Scan(
						&current.CategoryID,
						&current.ProductCode,
						&current.Name,
						&current.Slug,
						&current.Brand,
						&current.ShortDescription,
						&current.Description,
						&current.Status,
						&current.IsFeatured,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminCatalogProductNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin catalog product: %w",
						err,
					)
				}

				next :=
					current

				applyAdminCatalogProductUpdate(
					&next,
					input,
				)

				if err :=
					validateAdminCatalogProduct(
						next,
					); err != nil {

					return err
				}

				var categoryActive bool

				err =
					tx.QueryRow(
						ctx,
						`
							SELECT is_active
							FROM categories
							WHERE id = $1::uuid
						`,
						next.CategoryID,
					).Scan(
						&categoryActive,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return fmt.Errorf(
						"%w: category does not exist",
						ErrInvalidAdminCatalogProduct,
					)
				}

				if err != nil {
					return fmt.Errorf(
						"load catalog category: %w",
						err,
					)
				}

				if next.Status == "active" &&
					!categoryActive {

					return ErrAdminCatalogInactiveCategory
				}

				if next.Status ==
					"active" {

					var hasActiveVariant bool

					if err :=
						tx.QueryRow(
							ctx,
							`
								SELECT EXISTS (
									SELECT 1
									FROM product_variants
									WHERE
										product_id = $1::uuid
										AND is_active = true
								)
							`,
							productID,
						).Scan(
							&hasActiveVariant,
						); err != nil {

						return fmt.Errorf(
							"check active catalog variants: %w",
							err,
						)
					}

					if !hasActiveVariant {
						return ErrAdminCatalogNoActiveVariant
					}
				}

				_, err =
					tx.Exec(
						ctx,
						`
							UPDATE products
							SET
								category_id = $2::uuid,
								name = $3,
								slug = $4,
								brand = NULLIF($5, ''),
								short_description = NULLIF($6, ''),
								description = NULLIF($7, ''),
								status = $8::varchar(20),
								is_featured = $9,

								published_at =
									CASE
										WHEN
											$8::varchar(20) = 'active'
											AND published_at IS NULL
										THEN now()
										ELSE published_at
									END,

								updated_at = now()

							WHERE id = $1::uuid
						`,
						productID,
						next.CategoryID,
						next.Name,
						next.Slug,
						next.Brand,
						next.ShortDescription,
						next.Description,
						next.Status,
						next.IsFeatured,
					)

				if err != nil {
					if isAdminUniqueViolation(
						err,
					) {
						return ErrAdminCatalogConflict
					}

					return fmt.Errorf(
						"update Admin catalog product: %w",
						err,
					)
				}

				return insertAdminActionAuditTx(
					ctx,
					tx,
					metadata,
					adminEventCatalogProductUpdated,
					map[string]any{
						"product_id": productID,

						"product_code": current.ProductCode,

						"previous_status": current.Status,

						"new_status": next.Status,
					},
				)
			},
		)

	if err != nil {
		return CatalogProductDetail{},
			err
	}

	return s.GetCatalogProduct(
		ctx,
		productID,
	)
}

func applyAdminCatalogProductUpdate(
	target *CatalogProductDetail,
	input UpdateCatalogProductInput,
) {
	if input.CategoryID != nil {
		target.CategoryID =
			strings.TrimSpace(
				*input.CategoryID,
			)
	}

	if input.Name != nil {
		target.Name =
			strings.TrimSpace(
				*input.Name,
			)
	}

	if input.Slug != nil {
		target.Slug =
			strings.ToLower(
				strings.TrimSpace(
					*input.Slug,
				),
			)
	}

	if input.Brand != nil {
		target.Brand =
			strings.TrimSpace(
				*input.Brand,
			)
	}

	if input.ShortDescription != nil {
		target.ShortDescription =
			strings.TrimSpace(
				*input.ShortDescription,
			)
	}

	if input.Description != nil {
		target.Description =
			strings.TrimSpace(
				*input.Description,
			)
	}

	if input.Status != nil {
		target.Status =
			strings.ToLower(
				strings.TrimSpace(
					*input.Status,
				),
			)
	}

	if input.IsFeatured != nil {
		target.IsFeatured =
			*input.IsFeatured
	}
}

func validateAdminCatalogProduct(
	item CatalogProductDetail,
) error {
	if item.CategoryID == "" {
		return fmt.Errorf(
			"%w: category_id is required",
			ErrInvalidAdminCatalogProduct,
		)
	}

	if item.Name == "" ||
		len(
			[]rune(
				item.Name,
			),
		) > 180 {

		return fmt.Errorf(
			"%w: name is required and cannot exceed 180 characters",
			ErrInvalidAdminCatalogProduct,
		)
	}

	if !validAdminCatalogSlug(
		item.Slug,
	) {
		return fmt.Errorf(
			"%w: slug is invalid",
			ErrInvalidAdminCatalogProduct,
		)
	}

	if len(
		[]rune(
			item.Brand,
		),
	) > 120 {

		return fmt.Errorf(
			"%w: brand cannot exceed 120 characters",
			ErrInvalidAdminCatalogProduct,
		)
	}

	if len(
		[]rune(
			item.ShortDescription,
		),
	) > 500 {

		return fmt.Errorf(
			"%w: short_description cannot exceed 500 characters",
			ErrInvalidAdminCatalogProduct,
		)
	}

	switch item.Status {
	case "draft",
		"active",
		"archived":

	default:
		return fmt.Errorf(
			"%w: status must be draft, active, or archived",
			ErrInvalidAdminCatalogProduct,
		)
	}

	return nil
}

func validAdminCatalogSlug(
	value string,
) bool {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" ||
		len(
			[]rune(
				value,
			),
		) > 200 ||
		strings.HasPrefix(
			value,
			"-",
		) ||
		strings.HasSuffix(
			value,
			"-",
		) {

		return false
	}

	for _, character := range value {

		if character == '-' ||
			unicode.IsLetter(
				character,
			) ||
			unicode.IsDigit(
				character,
			) {

			continue
		}

		return false
	}

	return true
}

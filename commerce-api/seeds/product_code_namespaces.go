package seeds

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productCodeNamespaceSeed struct {
	CategorySlug string
	Prefix       string
}

func SeedProductCodeNamespaces(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin product-code namespace seed transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	items := []productCodeNamespaceSeed{
		{
			CategorySlug: "bags",
			Prefix:       "ACC-BAG",
		},
		{
			CategorySlug: "sunglasses",
			Prefix:       "ACC-SUN",
		},
		{
			CategorySlug: "watches",
			Prefix:       "ACC-WAT",
		},
		{
			CategorySlug: "mens-shoes",
			Prefix:       "FWT-MSH",
		},
		{
			CategorySlug: "womens-shoes",
			Prefix:       "FWT-WSH",
		},
		{
			CategorySlug: "kids-boys",
			Prefix:       "KID-BOY",
		},
		{
			CategorySlug: "kids-girls",
			Prefix:       "KID-GRL",
		},
		{
			CategorySlug: "mens-panjabi",
			Prefix:       "MEN-PAN",
		},
		{
			CategorySlug: "mens-pants",
			Prefix:       "MEN-PNT",
		},
		{
			CategorySlug: "mens-shirts",
			Prefix:       "MEN-SHT",
		},
		{
			CategorySlug: "men-shirts-formal-shirts",
			Prefix:       "MEN-FSH",
		},
		{
			CategorySlug: "mens-t-shirts",
			Prefix:       "MEN-TSH",
		},
		{
			CategorySlug: "men-traditional-wear",
			Prefix:       "MEN-TRD",
		},
		{
			CategorySlug: "womens-dresses",
			Prefix:       "WOM-DRS",
		},
		{
			CategorySlug: "womens-kurti-kameez",
			Prefix:       "WOM-KRK",
		},
		{
			CategorySlug: "womens-saree",
			Prefix:       "WOM-SAR",
		},
		{
			CategorySlug: "womens-tops",
			Prefix:       "WOM-TOP",
		},
	}

	for _, item := range items {
		_, err := tx.Exec(
			ctx,
			`
				INSERT INTO product_code_namespaces (
					category_id,
					prefix,
					next_number,
					created_at,
					updated_at
				)
				SELECT
					id,
					$2,
					1,
					now(),
					now()
				FROM categories
				WHERE slug = $1
				ON CONFLICT (category_id)
				DO NOTHING
			`,
			item.CategorySlug,
			item.Prefix,
		)
		if err != nil {
			return fmt.Errorf(
				"seed namespace %q for category %q: %w",
				item.Prefix,
				item.CategorySlug,
				err,
			)
		}

		var actualPrefix string

		err = tx.QueryRow(
			ctx,
			`
				SELECT n.prefix
				FROM product_code_namespaces n
				JOIN categories c
					ON c.id = n.category_id
				WHERE c.slug = $1
			`,
			item.CategorySlug,
		).Scan(
			&actualPrefix,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf(
					"namespace category %q is missing or has no namespace",
					item.CategorySlug,
				)
			}

			return fmt.Errorf(
				"verify namespace for category %q: %w",
				item.CategorySlug,
				err,
			)
		}

		if actualPrefix != item.Prefix {
			return fmt.Errorf(
				"namespace mismatch for category %q: database has %q, seed expects %q",
				item.CategorySlug,
				actualPrefix,
				item.Prefix,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit product-code namespace seed transaction: %w",
			err,
		)
	}

	return nil
}

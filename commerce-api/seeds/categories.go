package seeds

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type rootCategory struct {
	Name      string
	Slug      string
	SortOrder int
}

type childCategory struct {
	ParentSlug string
	Name       string
	Slug       string
	SortOrder  int
}

func SeedCategories(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin category seed transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rootCategories := []rootCategory{
		{
			Name:      "Men",
			Slug:      "men",
			SortOrder: 10,
		},
		{
			Name:      "Women",
			Slug:      "women",
			SortOrder: 20,
		},
		{
			Name:      "Kids",
			Slug:      "kids",
			SortOrder: 30,
		},
		{
			Name:      "Footwear",
			Slug:      "footwear",
			SortOrder: 40,
		},
		{
			Name:      "Accessories",
			Slug:      "accessories",
			SortOrder: 50,
		},
	}

	for _, item := range rootCategories {
		_, err := tx.Exec(
			ctx,
			`
				INSERT INTO categories (
					name,
					slug,
					sort_order,
					is_active
				)
				VALUES ($1, $2, $3, true)
				ON CONFLICT (slug)
				DO UPDATE SET
					name = EXCLUDED.name,
					sort_order = EXCLUDED.sort_order,
					is_active = EXCLUDED.is_active,
					updated_at = now()
			`,
			item.Name,
			item.Slug,
			item.SortOrder,
		)
		if err != nil {
			return fmt.Errorf(
				"seed root category %q: %w",
				item.Slug,
				err,
			)
		}
	}

	childCategories := []childCategory{
		{
			ParentSlug: "men",
			Name:       "Shirts",
			Slug:       "mens-shirts",
			SortOrder:  10,
		},
		{
			ParentSlug: "men",
			Name:       "T-Shirts",
			Slug:       "mens-t-shirts",
			SortOrder:  20,
		},
		{
			ParentSlug: "men",
			Name:       "Panjabi",
			Slug:       "mens-panjabi",
			SortOrder:  30,
		},
		{
			ParentSlug: "men",
			Name:       "Pants",
			Slug:       "mens-pants",
			SortOrder:  40,
		},
		{
			ParentSlug: "men",
			Name:       "Traditional Wear",
			Slug:       "men-traditional-wear",
			SortOrder:  50,
		},

		{
			ParentSlug: "women",
			Name:       "Kurti & Kameez",
			Slug:       "womens-kurti-kameez",
			SortOrder:  10,
		},
		{
			ParentSlug: "women",
			Name:       "Saree",
			Slug:       "womens-saree",
			SortOrder:  20,
		},
		{
			ParentSlug: "women",
			Name:       "Tops",
			Slug:       "womens-tops",
			SortOrder:  30,
		},
		{
			ParentSlug: "women",
			Name:       "Dresses",
			Slug:       "womens-dresses",
			SortOrder:  40,
		},

		{
			ParentSlug: "kids",
			Name:       "Boys",
			Slug:       "kids-boys",
			SortOrder:  10,
		},
		{
			ParentSlug: "kids",
			Name:       "Girls",
			Slug:       "kids-girls",
			SortOrder:  20,
		},

		{
			ParentSlug: "footwear",
			Name:       "Men's Shoes",
			Slug:       "mens-shoes",
			SortOrder:  10,
		},
		{
			ParentSlug: "footwear",
			Name:       "Women's Shoes",
			Slug:       "womens-shoes",
			SortOrder:  20,
		},

		{
			ParentSlug: "accessories",
			Name:       "Bags",
			Slug:       "bags",
			SortOrder:  10,
		},
		{
			ParentSlug: "accessories",
			Name:       "Watches",
			Slug:       "watches",
			SortOrder:  20,
		},
		{
			ParentSlug: "accessories",
			Name:       "Sunglasses",
			Slug:       "sunglasses",
			SortOrder:  30,
		},

		{
			ParentSlug: "mens-shirts",
			Name:       "Formal Shirts",
			Slug:       "men-shirts-formal-shirts",
			SortOrder:  10,
		},
	}
	// Replace the placeholder above with the remaining categories.
	childCategories = []childCategory{
		{
			ParentSlug: "men",
			Name:       "Shirts",
			Slug:       "mens-shirts",
			SortOrder:  10,
		},
		{
			ParentSlug: "men",
			Name:       "T-Shirts",
			Slug:       "mens-t-shirts",
			SortOrder:  20,
		},
		{
			ParentSlug: "men",
			Name:       "Panjabi",
			Slug:       "mens-panjabi",
			SortOrder:  30,
		},
		{
			ParentSlug: "men",
			Name:       "Pants",
			Slug:       "mens-pants",
			SortOrder:  40,
		},
		{
			ParentSlug: "men",
			Name:       "Traditional Wear",
			Slug:       "men-traditional-wear",
			SortOrder:  50,
		},

		{
			ParentSlug: "women",
			Name:       "Kurti & Kameez",
			Slug:       "womens-kurti-kameez",
			SortOrder:  10,
		},
		{
			ParentSlug: "women",
			Name:       "Saree",
			Slug:       "womens-saree",
			SortOrder:  20,
		},
		{
			ParentSlug: "women",
			Name:       "Tops",
			Slug:       "womens-tops",
			SortOrder:  30,
		},
		{
			ParentSlug: "women",
			Name:       "Dresses",
			Slug:       "womens-dresses",
			SortOrder:  40,
		},

		{
			ParentSlug: "kids",
			Name:       "Boys",
			Slug:       "kids-boys",
			SortOrder:  10,
		},
		{
			ParentSlug: "kids",
			Name:       "Girls",
			Slug:       "kids-girls",
			SortOrder:  20,
		},

		{
			ParentSlug: "footwear",
			Name:       "Men's Shoes",
			Slug:       "mens-shoes",
			SortOrder:  10,
		},
		{
			ParentSlug: "footwear",
			Name:       "Women's Shoes",
			Slug:       "womens-shoes",
			SortOrder:  20,
		},

		{
			ParentSlug: "accessories",
			Name:       "Bags",
			Slug:       "bags",
			SortOrder:  10,
		},
		{
			ParentSlug: "accessories",
			Name:       "Watches",
			Slug:       "watches",
			SortOrder:  20,
		},
		{
			ParentSlug: "accessories",
			Name:       "Sunglasses",
			Slug:       "sunglasses",
			SortOrder:  30,
		},

		// Third-level product category.
		// mens-shirts is created earlier in this slice.
		{
			ParentSlug: "mens-shirts",
			Name:       "Formal Shirts",
			Slug:       "men-shirts-formal-shirts",
			SortOrder:  10,
		},
	}

	for _, item := range childCategories {
		_, err := tx.Exec(
			ctx,
			`
				INSERT INTO categories (
					parent_id,
					name,
					slug,
					sort_order,
					is_active
				)
				SELECT
					id,
					$1,
					$2,
					$3,
					true
				FROM categories
				WHERE slug = $4
				ON CONFLICT (slug)
				DO UPDATE SET
					parent_id = EXCLUDED.parent_id,
					name = EXCLUDED.name,
					sort_order = EXCLUDED.sort_order,
					is_active = EXCLUDED.is_active,
					updated_at = now()
			`,
			item.Name,
			item.Slug,
			item.SortOrder,
			item.ParentSlug,
		)
		if err != nil {
			return fmt.Errorf(
				"seed child category %q: %w",
				item.Slug,
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit category seed transaction: %w",
			err,
		)
	}

	return nil
}

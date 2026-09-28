package catalogwrite

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func resolveProductCategoryID(
	ctx context.Context,
	tx pgx.Tx,
	input ProductInput,
) (string, error) {
	if input.CategoryID == "" {
		return ensureCategoryPath(
			ctx,
			tx,
			input.CategoryPath,
		)
	}

	var categoryID string
	var isActive bool

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				is_active
			FROM categories
			WHERE id::text = $1
			LIMIT 1
		`,
		input.CategoryID,
	).Scan(
		&categoryID,
		&isActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "",
				fmt.Errorf(
					"%w: %s",
					ErrProductCategoryNotFound,
					input.CategoryID,
				)
		}

		return "",
			fmt.Errorf(
				"resolve product category: %w",
				err,
			)
	}

	if input.Status == "active" &&
		!isActive {
		return "",
			ErrProductCategoryInactive
	}

	return categoryID, nil
}

func allocateVariantSKU(
	ctx context.Context,
	tx pgx.Tx,
	productCode string,
	productID string,
	start int,
) (string, error) {
	base := automaticSKUBase(
		productCode,
		productID,
	)

	if start < 1 {
		start = 1
	}

	for number := start; number <= 9999; number++ {

		candidate := fmt.Sprintf(
			"%s-V%03d",
			base,
			number,
		)

		if len(candidate) > 100 {
			return "",
				fmt.Errorf(
					"generated SKU exceeds supported length",
				)
		}

		var exists bool

		err := tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM product_variants
					WHERE upper(sku) =
						upper($1)
				)
			`,
			candidate,
		).Scan(
			&exists,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"check generated SKU %q: %w",
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
			"could not generate SKU for product %s",
			productCode,
		)
}

func automaticSKUBase(
	productCode string,
	productID string,
) string {
	productCode = strings.ToUpper(
		strings.TrimSpace(
			productCode,
		),
	)

	// "-V0001" still leaves plenty of room
	// under the varchar(100) SKU constraint.
	if productCode != "" &&
		len(productCode) <= 94 {
		return productCode
	}

	compactID := strings.ToUpper(
		strings.ReplaceAll(
			productID,
			"-",
			"",
		),
	)

	if len(compactID) > 16 {
		compactID =
			compactID[:16]
	}

	return "PV-" + compactID
}

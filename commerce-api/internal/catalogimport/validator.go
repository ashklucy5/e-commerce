package catalogimport

import (
	"fmt"
	"regexp"
	"strings"
)

var colorHexPattern = regexp.MustCompile(
	`^#[0-9A-Fa-f]{6}$`,
)

func Validate(
	result *ParseResult,
) {
	validateProducts(result)
	validateVariants(result)
	validateImages(result)
	validatePriceTiers(result)
	validateCategories(result)
}

func validateProducts(
	result *ParseResult,
) {
	seen := make(map[string]RowSource)

	for _, row := range result.Workbook.Products {
		code := strings.TrimSpace(row.ProductCode)

		if code == "" {
			addError(
				result,
				row.Source,
				"product_code",
				"REQUIRED",
				"product_code is required",
			)
		}

		if row.ProductName == "" {
			addError(
				result,
				row.Source,
				"product_name",
				"REQUIRED",
				"product_name is required",
			)
		}

		if row.CategoryPath == "" {
			addError(
				result,
				row.Source,
				"category_path",
				"REQUIRED",
				"category_path is required",
			)
		} else if !validCategoryPath(
			row.CategoryPath,
		) {
			addError(
				result,
				row.Source,
				"category_path",
				"INVALID_CATEGORY_PATH",
				"category_path must look like Men > Shirts > Formal Shirts",
			)
		}

		switch strings.ToLower(row.Status) {
		case "draft", "active", "archived":
		default:
			addError(
				result,
				row.Source,
				"status",
				"INVALID_STATUS",
				"status must be draft, active, or archived",
			)
		}

		key := strings.ToUpper(code)

		if previous, exists := seen[key]; exists &&
			key != "" {
			addError(
				result,
				row.Source,
				"product_code",
				"DUPLICATE_PRODUCT_CODE",
				fmt.Sprintf(
					"product_code already appears on row %d",
					previous.Row,
				),
			)
		} else {
			seen[key] = row.Source
		}
	}
}

func validateVariants(
	result *ParseResult,
) {
	productCodes := make(map[string]struct{})

	for _, product := range result.Workbook.Products {
		productCodes[strings.ToUpper(
			strings.TrimSpace(product.ProductCode),
		)] = struct{}{}
	}

	seenSKU := make(map[string]RowSource)

	for _, row := range result.Workbook.Variants {
		productCode := strings.ToUpper(
			strings.TrimSpace(row.ProductCode),
		)

		sku := strings.ToUpper(
			strings.TrimSpace(row.SKU),
		)

		if productCode == "" {
			addError(
				result,
				row.Source,
				"product_code",
				"REQUIRED",
				"product_code is required",
			)
		} else if _, exists :=
			productCodes[productCode]; !exists {
			addError(
				result,
				row.Source,
				"product_code",
				"UNKNOWN_PRODUCT_CODE",
				"variant product_code does not exist in Products sheet",
			)
		}

		if sku == "" {
			addError(
				result,
				row.Source,
				"sku",
				"REQUIRED",
				"sku is required",
			)
		}

		if previous, exists := seenSKU[sku]; exists &&
			sku != "" {
			addError(
				result,
				row.Source,
				"sku",
				"DUPLICATE_SKU",
				fmt.Sprintf(
					"sku already appears on row %d",
					previous.Row,
				),
			)
		} else {
			seenSKU[sku] = row.Source
		}

		if row.PriceAmount < 0 {
			addError(
				result,
				row.Source,
				"price",
				"NEGATIVE_PRICE",
				"price cannot be negative",
			)
		}

		if row.CompareAtPriceAmount != nil &&
			*row.CompareAtPriceAmount < row.PriceAmount {
			addError(
				result,
				row.Source,
				"compare_at_price",
				"INVALID_COMPARE_AT_PRICE",
				"compare_at_price cannot be lower than price",
			)
		}

		if row.CostAmount != nil &&
			*row.CostAmount < 0 {
			addError(
				result,
				row.Source,
				"cost",
				"NEGATIVE_COST",
				"cost cannot be negative",
			)
		}

		if row.MinimumOrderQuantity <= 0 {
			addError(
				result,
				row.Source,
				"moq",
				"INVALID_MOQ",
				"moq must be greater than zero",
			)
		}

		if row.OrderIncrement != 1 {
			addError(
				result,
				row.Source,
				"order_increment",
				"INVALID_ORDER_INCREMENT",
				"order_increment must be 1",
			)
		}

		if row.Stock < 0 {
			addError(
				result,
				row.Source,
				"stock",
				"NEGATIVE_STOCK",
				"stock cannot be negative",
			)
		}

		if row.ReorderLevel < 0 {
			addError(
				result,
				row.Source,
				"reorder_level",
				"NEGATIVE_REORDER_LEVEL",
				"reorder_level cannot be negative",
			)
		}

		if len(row.Currency) != 3 {
			addError(
				result,
				row.Source,
				"currency",
				"INVALID_CURRENCY",
				"currency must contain exactly 3 letters",
			)
		}

		if row.ColorHex != "" &&
			!colorHexPattern.MatchString(
				row.ColorHex,
			) {
			addError(
				result,
				row.Source,
				"color_hex",
				"INVALID_COLOR_HEX",
				"color_hex must look like #000000",
			)
		}

		if row.WeightGrams != nil &&
			*row.WeightGrams < 0 {
			addError(
				result,
				row.Source,
				"weight_grams",
				"NEGATIVE_WEIGHT",
				"weight_grams cannot be negative",
			)
		}
	}
}

func validateImages(
	result *ParseResult,
) {
	products := make(map[string]struct{})
	variants := make(map[string]string)

	for _, product := range result.Workbook.Products {
		products[strings.ToUpper(
			product.ProductCode,
		)] = struct{}{}
	}

	for _, variant := range result.Workbook.Variants {
		variants[strings.ToUpper(
			variant.SKU,
		)] = strings.ToUpper(
			variant.ProductCode,
		)
	}

	primaryByProduct := make(
		map[string]RowSource,
	)

	for _, row := range result.Workbook.Images {
		productCode := strings.ToUpper(
			strings.TrimSpace(row.ProductCode),
		)

		sku := strings.ToUpper(
			strings.TrimSpace(row.SKU),
		)

		if _, exists := products[productCode]; !exists {
			addError(
				result,
				row.Source,
				"product_code",
				"UNKNOWN_PRODUCT_CODE",
				"image product_code does not exist in Products sheet",
			)
		}

		if sku != "" {
			variantProduct, exists := variants[sku]

			if !exists {
				addError(
					result,
					row.Source,
					"sku",
					"UNKNOWN_SKU",
					"image SKU does not exist in Variants sheet",
				)
			} else if variantProduct != productCode {
				addError(
					result,
					row.Source,
					"sku",
					"SKU_PRODUCT_MISMATCH",
					"image SKU belongs to a different product_code",
				)
			}
		}

		hasURL := strings.TrimSpace(
			row.ImageURL,
		) != ""

		hasFile := strings.TrimSpace(
			row.ImageFile,
		) != ""

		if hasURL == hasFile {
			addError(
				result,
				row.Source,
				"image_url",
				"INVALID_IMAGE_SOURCE",
				"provide exactly one of image_url or image_file",
			)
		}

		if row.SortOrder < 0 {
			addError(
				result,
				row.Source,
				"sort_order",
				"NEGATIVE_SORT_ORDER",
				"sort_order cannot be negative",
			)
		}

		if row.IsPrimary {
			if sku != "" {
				addError(
					result,
					row.Source,
					"is_primary",
					"VARIANT_IMAGE_CANNOT_BE_PRIMARY",
					"only a product-level image may be the primary image",
				)
			}

			if previous, exists :=
				primaryByProduct[productCode]; exists {
				addError(
					result,
					row.Source,
					"is_primary",
					"DUPLICATE_PRIMARY_IMAGE",
					fmt.Sprintf(
						"product already has a primary image on row %d",
						previous.Row,
					),
				)
			} else {
				primaryByProduct[productCode] =
					row.Source
			}
		}
	}
}

func validatePriceTiers(
	result *ParseResult,
) {
	variants := make(
		map[string]VariantRow,
	)

	for _, variant := range result.Workbook.Variants {
		variants[strings.ToUpper(
			strings.TrimSpace(variant.SKU),
		)] = variant
	}

	seen := make(map[string]RowSource)

	for _, row := range result.Workbook.PriceTiers {
		sku := strings.ToUpper(
			strings.TrimSpace(row.SKU),
		)

		variant, exists := variants[sku]

		if !exists {
			addError(
				result,
				row.Source,
				"sku",
				"UNKNOWN_SKU",
				"price tier SKU does not exist in Variants sheet",
			)
		}

		if row.MinQuantity <= 0 {
			addError(
				result,
				row.Source,
				"min_quantity",
				"INVALID_MIN_QUANTITY",
				"min_quantity must be greater than zero",
			)
		}

		if exists &&
			row.MinQuantity <
				variant.MinimumOrderQuantity {
			addError(
				result,
				row.Source,
				"min_quantity",
				"TIER_BELOW_MOQ",
				"price tier min_quantity cannot be below the variant MOQ",
			)
		}

		if row.UnitPriceAmount < 0 {
			addError(
				result,
				row.Source,
				"unit_price",
				"NEGATIVE_PRICE",
				"unit_price cannot be negative",
			)
		}

		key := fmt.Sprintf(
			"%s:%d",
			sku,
			row.MinQuantity,
		)

		if previous, exists := seen[key]; exists {
			addError(
				result,
				row.Source,
				"min_quantity",
				"DUPLICATE_PRICE_TIER",
				fmt.Sprintf(
					"same SKU and quantity tier already appears on row %d",
					previous.Row,
				),
			)
		} else {
			seen[key] = row.Source
		}
	}
}

func validateCategories(
	result *ParseResult,
) {
	seen := make(map[string]RowSource)

	for _, row := range result.Workbook.Categories {
		if !validCategoryPath(row.CategoryPath) {
			addError(
				result,
				row.Source,
				"category_path",
				"INVALID_CATEGORY_PATH",
				"category_path must look like Men > Shirts > Formal Shirts",
			)
		}

		if row.SortOrder < 0 {
			addError(
				result,
				row.Source,
				"sort_order",
				"NEGATIVE_SORT_ORDER",
				"sort_order cannot be negative",
			)
		}

		key := strings.ToLower(
			normalizeCategoryPath(
				row.CategoryPath,
			),
		)

		if previous, exists := seen[key]; exists &&
			key != "" {
			addError(
				result,
				row.Source,
				"category_path",
				"DUPLICATE_CATEGORY",
				fmt.Sprintf(
					"category path already appears on row %d",
					previous.Row,
				),
			)
		} else {
			seen[key] = row.Source
		}
	}
}

func validCategoryPath(
	path string,
) bool {
	parts := strings.Split(path, ">")

	if len(parts) == 0 {
		return false
	}

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}

	return true
}

func normalizeCategoryPath(
	path string,
) string {
	parts := strings.Split(path, ">")

	for index := range parts {
		parts[index] = strings.TrimSpace(
			parts[index],
		)
	}

	return strings.Join(parts, " > ")
}

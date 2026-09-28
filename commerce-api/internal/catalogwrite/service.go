package catalogwrite

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) CreateProduct(
	ctx context.Context,
	input ProductInput,
) (ProductResult, error) {
	normalizeProductInput(
		&input,
	)

	if err := validateProductInput(
		input,
	); err != nil {
		return ProductResult{},
			newProductValidationError(
				err,
			)
	}

	return s.repository.CreateProduct(
		ctx,
		input,
	)
}

func normalizeProductInput(
	input *ProductInput,
) {
	input.ProductCode = strings.TrimSpace(
		input.ProductCode,
	)

	input.Name = strings.TrimSpace(
		input.Name,
	)

	input.CategoryID = strings.TrimSpace(
		input.CategoryID,
	)

	input.CategoryPath =
		normalizeCategoryPath(
			input.CategoryPath,
		)

	input.Slug = strings.TrimSpace(
		input.Slug,
	)

	input.Brand = strings.TrimSpace(
		input.Brand,
	)

	input.ShortDescription =
		strings.TrimSpace(
			input.ShortDescription,
		)

	input.Description =
		strings.TrimSpace(
			input.Description,
		)

	input.Status = strings.ToLower(
		strings.TrimSpace(
			input.Status,
		),
	)

	if input.Status == "" {
		input.Status = "draft"
	}

	for index := range input.Variants {
		variant := &input.Variants[index]

		variant.SKU = strings.TrimSpace(
			variant.SKU,
		)

		variant.ColorName = strings.TrimSpace(
			variant.ColorName,
		)

		variant.ColorHex = strings.TrimSpace(
			variant.ColorHex,
		)

		variant.Size = strings.TrimSpace(
			variant.Size,
		)

		variant.Currency = strings.ToUpper(
			strings.TrimSpace(
				variant.Currency,
			),
		)

		if variant.Currency == "" {
			variant.Currency = "BDT"
		}

		variant.Barcode = strings.TrimSpace(
			variant.Barcode,
		)

		if variant.MinimumOrderQuantity == 0 {
			variant.MinimumOrderQuantity = 1
		}

		if variant.OrderIncrement == 0 {
			variant.OrderIncrement = 1
		}
	}
}

func validateProductInput(
	input ProductInput,
) error {
	if input.Name == "" {
		return fmt.Errorf(
			"product name is required",
		)
	}

	if input.CategoryID == "" &&
		input.CategoryPath == "" {
		return fmt.Errorf(
			"category_id or category_path is required",
		)
	}

	switch input.Status {
	case "draft", "active", "archived":
	default:
		return fmt.Errorf(
			"invalid product status %q",
			input.Status,
		)
	}

	if len(input.Variants) == 0 {
		return fmt.Errorf(
			"at least one product variant is required",
		)
	}

	seenSKUs := make(
		map[string]struct{},
		len(input.Variants),
	)

	for index, variant := range input.Variants {
		label := variantValidationLabel(
			index,
			variant.SKU,
		)

		// SKU is optional for normal product creation.
		// When omitted, the repository generates one
		// after the immutable product code is known.
		if variant.SKU != "" {
			skuKey := strings.ToUpper(
				variant.SKU,
			)

			if _, exists :=
				seenSKUs[skuKey]; exists {

				return fmt.Errorf(
					"duplicate SKU %q",
					variant.SKU,
				)
			}

			seenSKUs[skuKey] =
				struct{}{}
		}

		if variant.MinimumOrderQuantity <= 0 {
			return fmt.Errorf(
				"%s: MOQ must be greater than zero",
				label,
			)
		}

		if variant.OrderIncrement != 1 {
			return fmt.Errorf(
				"%s: order increment must be 1",
				label,
			)
		}

		if variant.PriceAmount < 0 {
			return fmt.Errorf(
				"%s: price cannot be negative",
				label,
			)
		}

		if variant.CompareAtPriceAmount != nil &&
			*variant.CompareAtPriceAmount <
				variant.PriceAmount {

			return fmt.Errorf(
				"%s: compare-at price cannot be below price",
				label,
			)
		}

		if variant.CostAmount != nil &&
			*variant.CostAmount < 0 {

			return fmt.Errorf(
				"%s: cost cannot be negative",
				label,
			)
		}

		if len(variant.Currency) != 3 {
			return fmt.Errorf(
				"%s: currency must contain exactly 3 letters",
				label,
			)
		}

		if variant.Stock < 0 {
			return fmt.Errorf(
				"%s: stock cannot be negative",
				label,
			)
		}

		if variant.ReorderLevel < 0 {
			return fmt.Errorf(
				"%s: reorder level cannot be negative",
				label,
			)
		}

		if variant.WeightGrams != nil &&
			*variant.WeightGrams < 0 {

			return fmt.Errorf(
				"%s: weight cannot be negative",
				label,
			)
		}

		seenTiers := make(
			map[int]struct{},
			len(variant.PriceTiers),
		)

		for _, tier := range variant.PriceTiers {
			if tier.MinQuantity <= 0 {
				return fmt.Errorf(
					"%s: price-tier quantity must be greater than zero",
					label,
				)
			}

			if tier.MinQuantity <
				variant.MinimumOrderQuantity {

				return fmt.Errorf(
					"%s: price-tier quantity %d cannot be below MOQ %d",
					label,
					tier.MinQuantity,
					variant.MinimumOrderQuantity,
				)
			}

			if tier.UnitPriceAmount < 0 {
				return fmt.Errorf(
					"%s: price-tier amount cannot be negative",
					label,
				)
			}

			if _, exists :=
				seenTiers[tier.MinQuantity]; exists {

				return fmt.Errorf(
					"%s: duplicate price tier for quantity %d",
					label,
					tier.MinQuantity,
				)
			}

			seenTiers[tier.MinQuantity] =
				struct{}{}
		}
	}

	return nil
}

func variantValidationLabel(
	index int,
	sku string,
) string {
	sku = strings.TrimSpace(
		sku,
	)

	if sku != "" {
		return fmt.Sprintf(
			"SKU %q",
			sku,
		)
	}

	return fmt.Sprintf(
		"variant %d",
		index+1,
	)
}

func normalizeCategoryPath(
	path string,
) string {
	raw := strings.Split(
		path,
		">",
	)

	parts := make(
		[]string,
		0,
		len(raw),
	)

	for _, value := range raw {
		value = strings.TrimSpace(
			value,
		)

		if value == "" {
			continue
		}

		parts = append(
			parts,
			value,
		)
	}

	return strings.Join(
		parts,
		" > ",
	)
}

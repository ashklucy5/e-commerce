package catalogimport

import "testing"

func TestValidateRejectsOrderIncrementOtherThanOne(t *testing.T) {
	result := validOrderIncrementFixture(5)

	Validate(result)

	for _, validationError := range result.Errors {
		if validationError.Sheet == "Variants" &&
			validationError.Row == 2 &&
			validationError.Field == "order_increment" &&
			validationError.Code == "INVALID_ORDER_INCREMENT" &&
			validationError.Message == "order_increment must be 1" {
			return
		}
	}

	t.Fatalf(
		"expected INVALID_ORDER_INCREMENT error, got %#v",
		result.Errors,
	)
}

func TestValidateAcceptsOrderIncrementOne(t *testing.T) {
	result := validOrderIncrementFixture(1)

	Validate(result)

	if len(result.Errors) != 0 {
		t.Fatalf(
			"expected no validation errors, got %#v",
			result.Errors,
		)
	}
}

func validOrderIncrementFixture(
	orderIncrement int,
) *ParseResult {
	return &ParseResult{
		Workbook: Workbook{
			Products: []ProductRow{
				{
					Source: RowSource{
						Sheet: "Products",
						Row:   2,
					},
					ProductCode:  "TEST-PRODUCT",
					ProductName:  "Test Product",
					CategoryPath: "Men > Shirts",
					Status:       "active",
				},
			},
			Variants: []VariantRow{
				{
					Source: RowSource{
						Sheet: "Variants",
						Row:   2,
					},
					ProductCode:          "TEST-PRODUCT",
					SKU:                  "TEST-PRODUCT-BLK-M",
					PriceAmount:          150000,
					Currency:             "BDT",
					MinimumOrderQuantity: 1,
					OrderIncrement:       orderIncrement,
					Stock:                20,
					ReorderLevel:         5,
					IsActive:             true,
				},
			},
		},
	}
}

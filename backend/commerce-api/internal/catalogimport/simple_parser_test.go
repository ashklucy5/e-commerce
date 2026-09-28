package catalogimport

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseAutoAdvancedWorkbookStillUsesAdvancedContract(
	t *testing.T,
) {
	payload :=
		advancedWorkbookFixture(t)

	result, err :=
		ParseAuto(
			bytes.NewReader(
				payload,
			),
		)

	if err != nil {
		t.Fatalf(
			"parse advanced workbook: %v",
			err,
		)
	}

	if len(result.Errors) != 0 {
		t.Fatalf(
			"expected no validation errors, got %#v",
			result.Errors,
		)
	}

	if len(
		result.Workbook.Products,
	) != 1 {
		t.Fatalf(
			"expected 1 product, got %d",
			len(
				result.Workbook.Products,
			),
		)
	}

	if len(
		result.Workbook.Variants,
	) != 1 {
		t.Fatalf(
			"expected 1 variant, got %d",
			len(
				result.Workbook.Variants,
			),
		)
	}

	if result.Workbook.
		Products[0].
		ProductCode !=
		"ADV-001" {

		t.Fatalf(
			"advanced product code changed: %q",
			result.Workbook.
				Products[0].
				ProductCode,
		)
	}

	if result.Workbook.
		Variants[0].
		SKU !=
		"ADV-001-BLK-M" {

		t.Fatalf(
			"advanced SKU changed: %q",
			result.Workbook.
				Variants[0].
				SKU,
		)
	}
}

func TestParseAutoSimpleWorkbookExpandsVariantsAndImages(
	t *testing.T,
) {
	payload :=
		simpleWorkbookFixture(
			t,
			"Black | Navy | White",
			"M | L | XL",
			"12 | 12 | 12",
			"1490 | 1590 | 1690",
			"58 | 69 | 80",
		)

	result, err :=
		ParseAuto(
			bytes.NewReader(
				payload,
			),
		)

	if err != nil {
		t.Fatalf(
			"parse simple workbook: %v",
			err,
		)
	}

	if len(result.Errors) != 0 {
		t.Fatalf(
			"expected no validation errors, got %#v",
			result.Errors,
		)
	}

	if got :=
		len(
			result.Workbook.Products,
		); got != 1 {

		t.Fatalf(
			"expected 1 product, got %d",
			got,
		)
	}

	if got :=
		len(
			result.Workbook.Variants,
		); got != 3 {

		t.Fatalf(
			"expected 3 variants, got %d",
			got,
		)
	}

	if got :=
		len(
			result.Workbook.Images,
		); got != 2 {

		t.Fatalf(
			"expected 2 images, got %d",
			got,
		)
	}

	product :=
		result.Workbook.
			Products[0]

	if product.ProductCode == "" {
		t.Fatal(
			"expected deterministic product code",
		)
	}

	if product.CategoryPath !=
		"Men > Shirts > Formal Shirts" {

		t.Fatalf(
			"unexpected category path %q",
			product.CategoryPath,
		)
	}

	expectedPrices :=
		[]int64{
			149000,
			159000,
			169000,
		}

	expectedStock :=
		[]int{
			58,
			69,
			80,
		}

	expectedColors :=
		[]string{
			"Black",
			"Navy",
			"White",
		}

	expectedSizes :=
		[]string{
			"M",
			"L",
			"XL",
		}

	seenSKU :=
		make(
			map[string]struct{},
		)

	for index, variant := range result.Workbook.Variants {

		if variant.Source.Sheet !=
			sheetVariants {

			t.Fatalf(
				"variant %d has source sheet %q",
				index,
				variant.Source.Sheet,
			)
		}

		if variant.Source.Row !=
			index+2 {

			t.Fatalf(
				"variant %d has source row %d",
				index,
				variant.Source.Row,
			)
		}

		if variant.ProductCode !=
			product.ProductCode {

			t.Fatalf(
				"variant %d product code mismatch",
				index,
			)
		}

		if variant.OrderIncrement != 1 {
			t.Fatalf(
				"variant %d order increment = %d, want 1",
				index,
				variant.OrderIncrement,
			)
		}

		if variant.
			MinimumOrderQuantity != 12 {

			t.Fatalf(
				"variant %d MOQ = %d, want 12",
				index,
				variant.
					MinimumOrderQuantity,
			)
		}

		if variant.PriceAmount !=
			expectedPrices[index] {

			t.Fatalf(
				"variant %d price = %d, want %d",
				index,
				variant.PriceAmount,
				expectedPrices[index],
			)
		}

		if variant.Stock !=
			expectedStock[index] {

			t.Fatalf(
				"variant %d stock = %d, want %d",
				index,
				variant.Stock,
				expectedStock[index],
			)
		}

		if variant.ColorName !=
			expectedColors[index] {

			t.Fatalf(
				"variant %d color = %q, want %q",
				index,
				variant.ColorName,
				expectedColors[index],
			)
		}

		if variant.Size !=
			expectedSizes[index] {

			t.Fatalf(
				"variant %d size = %q, want %q",
				index,
				variant.Size,
				expectedSizes[index],
			)
		}

		if _, exists :=
			seenSKU[variant.SKU]; exists {

			t.Fatalf(
				"duplicate generated SKU %q",
				variant.SKU,
			)
		}

		seenSKU[variant.SKU] = struct{}{}
	}

	if !result.Workbook.
		Images[0].
		IsPrimary {

		t.Fatal(
			"expected Image 1 to become primary",
		)
	}

	if result.Workbook.
		Images[1].
		IsPrimary {

		t.Fatal(
			"expected Image 2 to be non-primary",
		)
	}
}

func TestParseAutoSimpleWorkbookBroadcastsSingleValues(
	t *testing.T,
) {
	payload :=
		simpleWorkbookFixture(
			t,
			"Black | Navy | White",
			"M | L | XL",
			"12",
			"1490",
			"50",
		)

	result, err :=
		ParseAuto(
			bytes.NewReader(
				payload,
			),
		)

	if err != nil {
		t.Fatalf(
			"parse simple workbook: %v",
			err,
		)
	}

	if len(result.Errors) != 0 {
		t.Fatalf(
			"expected no validation errors, got %#v",
			result.Errors,
		)
	}

	if len(
		result.Workbook.Variants,
	) != 3 {
		t.Fatalf(
			"expected 3 variants, got %d",
			len(
				result.Workbook.Variants,
			),
		)
	}

	for index, variant := range result.Workbook.Variants {

		if variant.
			MinimumOrderQuantity != 12 {

			t.Fatalf(
				"variant %d MOQ = %d, want 12",
				index,
				variant.
					MinimumOrderQuantity,
			)
		}

		if variant.PriceAmount !=
			149000 {

			t.Fatalf(
				"variant %d price = %d, want 149000",
				index,
				variant.PriceAmount,
			)
		}

		if variant.Stock != 50 {
			t.Fatalf(
				"variant %d stock = %d, want 50",
				index,
				variant.Stock,
			)
		}
	}
}

func TestParseAutoSimpleWorkbookRejectsMismatchedVariantLists(
	t *testing.T,
) {
	payload :=
		simpleWorkbookFixture(
			t,
			"Black | Navy | White",
			"M | L",
			"12 | 12 | 12",
			"1490 | 1590 | 1690",
			"58 | 69 | 80",
		)

	result, err :=
		ParseAuto(
			bytes.NewReader(
				payload,
			),
		)

	if err != nil {
		t.Fatalf(
			"parse simple workbook: %v",
			err,
		)
	}

	for _, validationError := range result.Errors {

		if validationError.Sheet ==
			sheetProducts &&
			validationError.Row == 2 &&
			validationError.Field ==
				"sizes" &&
			validationError.Code ==
				"SIMPLE_VARIANT_LENGTH_MISMATCH" {

			return
		}
	}

	t.Fatalf(
		"expected simple list length validation error, got %#v",
		result.Errors,
	)
}

func TestParseAutoSimpleIdentityStableWhenPriceAndStockChange(
	t *testing.T,
) {
	firstPayload :=
		simpleWorkbookFixture(
			t,
			"Black | Navy | White",
			"M | L | XL",
			"12 | 12 | 12",
			"1490 | 1590 | 1690",
			"58 | 69 | 80",
		)

	secondPayload :=
		simpleWorkbookFixture(
			t,
			"Black | Navy | White",
			"M | L | XL",
			"12 | 12 | 12",
			"1790 | 1890 | 1990",
			"108 | 109 | 110",
		)

	first, err :=
		ParseAuto(
			bytes.NewReader(
				firstPayload,
			),
		)

	if err != nil {
		t.Fatalf(
			"parse first workbook: %v",
			err,
		)
	}

	second, err :=
		ParseAuto(
			bytes.NewReader(
				secondPayload,
			),
		)

	if err != nil {
		t.Fatalf(
			"parse second workbook: %v",
			err,
		)
	}

	if first.Workbook.
		Products[0].
		ProductCode !=
		second.Workbook.
			Products[0].
			ProductCode {

		t.Fatalf(
			"product identity changed after price/stock edit: %q != %q",
			first.Workbook.
				Products[0].
				ProductCode,
			second.Workbook.
				Products[0].
				ProductCode,
		)
	}

	for index := range first.Workbook.Variants {

		if first.Workbook.
			Variants[index].
			SKU !=
			second.Workbook.
				Variants[index].
				SKU {

			t.Fatalf(
				"variant %d identity changed: %q != %q",
				index,
				first.Workbook.
					Variants[index].
					SKU,
				second.Workbook.
					Variants[index].
					SKU,
			)
		}
	}
}

func advancedWorkbookFixture(
	t *testing.T,
) []byte {
	t.Helper()

	file :=
		excelize.NewFile()

	file.SetSheetName(
		"Sheet1",
		sheetProducts,
	)

	writeTestRow(
		t,
		file,
		sheetProducts,
		1,
		[]string{
			"product_code",
			"product_name",
			"category_path",
			"status",
		},
	)

	writeTestRow(
		t,
		file,
		sheetProducts,
		2,
		[]string{
			"ADV-001",
			"Advanced Product",
			"Men > Shirts",
			"active",
		},
	)

	if _, err :=
		file.NewSheet(
			sheetVariants,
		); err != nil {

		t.Fatalf(
			"create Variants sheet: %v",
			err,
		)
	}

	writeTestRow(
		t,
		file,
		sheetVariants,
		1,
		[]string{
			"product_code",
			"sku",
			"price",
			"moq",
			"order_increment",
			"stock",
		},
	)

	writeTestRow(
		t,
		file,
		sheetVariants,
		2,
		[]string{
			"ADV-001",
			"ADV-001-BLK-M",
			"1490",
			"1",
			"1",
			"10",
		},
	)

	return writeWorkbookBytes(
		t,
		file,
	)
}

func simpleWorkbookFixture(
	t *testing.T,
	colors string,
	sizes string,
	moq string,
	price string,
	stock string,
) []byte {
	t.Helper()

	file :=
		excelize.NewFile()

	file.SetSheetName(
		"Sheet1",
		sheetProducts,
	)

	writeTestRow(
		t,
		file,
		sheetProducts,
		1,
		[]string{
			"Product Name",
			"Category",
			"Subcategory",
			"Child Category (Optional)",
			"Brand",
			"Short Description",
			"Long Description",
			"Status",
			"Colors",
			"Sizes",
			"Minimum Order Quantity (MOQ)",
			"Price (BDT)",
			"Opening Stock",
			"Image 1",
			"Image 2",
			"Image 3",
			"Image 4",
			"Image 5",
			"Image 6",
		},
	)

	writeTestRow(
		t,
		file,
		sheetProducts,
		2,
		[]string{
			"Premium Oxford Formal Shirt",
			"Men",
			"Shirts",
			"Formal Shirts",
			"Ene dei",
			"Demo shirt",
			"Long demo description",
			"active",
			colors,
			sizes,
			moq,
			price,
			stock,
			"https://example.com/shirt-front.jpg",
			"https://example.com/shirt-detail.jpg",
			"",
			"",
			"",
			"",
		},
	)

	return writeWorkbookBytes(
		t,
		file,
	)
}

func writeTestRow(
	t *testing.T,
	file *excelize.File,
	sheet string,
	row int,
	values []string,
) {
	t.Helper()

	for index, value := range values {

		cell, err :=
			excelize.CoordinatesToCellName(
				index+1,
				row,
			)

		if err != nil {
			t.Fatalf(
				"cell coordinate: %v",
				err,
			)
		}

		if err :=
			file.SetCellValue(
				sheet,
				cell,
				value,
			); err != nil {

			t.Fatalf(
				"set %s!%s: %v",
				sheet,
				cell,
				err,
			)
		}
	}
}

func writeWorkbookBytes(
	t *testing.T,
	file *excelize.File,
) []byte {
	t.Helper()

	defer file.Close()

	var buffer bytes.Buffer

	if err :=
		file.Write(
			&buffer,
		); err != nil {

		t.Fatalf(
			"write workbook: %v",
			err,
		)
	}

	if !strings.HasPrefix(
		buffer.String(),
		"PK",
	) {
		t.Fatal(
			"generated workbook is not an XLSX ZIP container",
		)
	}

	return buffer.Bytes()
}

package catalogimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	simpleHeaderProductName      = "product_name"
	simpleHeaderCategory         = "category"
	simpleHeaderSubcategory      = "subcategory"
	simpleHeaderChildCategory    = "child_category_(optional)"
	simpleHeaderBrand            = "brand"
	simpleHeaderShortDescription = "short_description"
	simpleHeaderLongDescription  = "long_description"
	simpleHeaderStatus           = "status"
	simpleHeaderColors           = "colors"
	simpleHeaderSizes            = "sizes"
	simpleHeaderMOQ              = "minimum_order_quantity_(moq)"
	simpleHeaderPrice            = "price_(bdt)"
	simpleHeaderOpeningStock     = "opening_stock"
)

var simpleRequiredHeaders = []string{
	simpleHeaderProductName,
	simpleHeaderCategory,
	simpleHeaderSubcategory,
	simpleHeaderColors,
	simpleHeaderSizes,
	simpleHeaderMOQ,
	simpleHeaderPrice,
	simpleHeaderOpeningStock,
}

// ParseAuto supports both catalog-import formats:
//
// Advanced:
//
//	Products + Variants
//	Optional: Images, PriceTiers, Categories
//
// Simple:
//
//	Products only, using the human-friendly admin template.
func ParseAuto(
	reader io.Reader,
) (*ParseResult, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"read XLSX workbook: %w",
			err,
		)
	}

	file, err := excelize.OpenReader(
		bytes.NewReader(payload),
		excelize.Options{
			RawCellValue:      true,
			UnzipXMLSizeLimit: 128 * 1024 * 1024,
			UnzipSizeLimit:    1024 * 1024 * 1024,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open XLSX workbook: %w",
			err,
		)
	}
	defer file.Close()

	productsSheet, ok := findSheet(
		file,
		sheetProducts,
	)
	if !ok {
		return nil, fmt.Errorf(
			"required worksheet %q not found",
			sheetProducts,
		)
	}

	// If Variants exists, this is the existing advanced
	// workbook contract. Delegate entirely to the old parser
	// so we do not alter the already-working behavior.
	if _, hasVariants := findSheet(
		file,
		sheetVariants,
	); hasVariants {
		return Parse(
			bytes.NewReader(payload),
		)
	}

	headers, err := readNormalizedHeaders(
		file,
		productsSheet,
	)
	if err != nil {
		return nil, err
	}

	if !looksLikeSimpleProductsSheet(
		headers,
	) {
		// Preserve the existing advanced-format error
		// for a malformed advanced workbook.
		return nil, fmt.Errorf(
			"required worksheet %q not found",
			sheetVariants,
		)
	}

	return parseSimpleWorkbook(
		file,
		productsSheet,
	)
}

func readNormalizedHeaders(
	file *excelize.File,
	sheet string,
) (map[string]struct{}, error) {
	rows, err := file.Rows(sheet)
	if err != nil {
		return nil, fmt.Errorf(
			"open worksheet %q: %w",
			sheet,
			err,
		)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf(
			"worksheet %q is empty",
			sheet,
		)
	}

	cells, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf(
			"read headers from %q: %w",
			sheet,
			err,
		)
	}

	headers := make(
		map[string]struct{},
		len(cells),
	)

	for _, cell := range cells {
		header := normalizeHeader(
			cell,
		)

		if header == "" {
			continue
		}

		headers[header] = struct{}{}
	}

	return headers, nil
}

func looksLikeSimpleProductsSheet(
	headers map[string]struct{},
) bool {
	for _, header := range []string{
		simpleHeaderProductName,
		simpleHeaderCategory,
		simpleHeaderPrice,
		simpleHeaderOpeningStock,
	} {
		if _, exists :=
			headers[header]; !exists {
			return false
		}
	}

	return true
}

func parseSimpleWorkbook(
	file *excelize.File,
	productsSheet string,
) (*ParseResult, error) {
	result := &ParseResult{
		Workbook: Workbook{
			Products: make(
				[]ProductRow,
				0,
			),
			Variants: make(
				[]VariantRow,
				0,
			),
			Images: make(
				[]ImageRow,
				0,
			),
			PriceTiers: make(
				[]PriceTierRow,
				0,
			),
			Categories: make(
				[]CategoryRow,
				0,
			),
		},
		Errors: make(
			[]ValidationError,
			0,
		),
	}

	/*
		The staging table requires each
		(sheet, row) pair to be unique.

		A simple Products row may produce:
		  1 Product
		  N Variants
		  N Images

		So generated records receive virtual
		canonical Variants / Images row numbers.

		This also means the existing staging,
		action-plan and Apply code can remain
		completely unchanged.
	*/
	variantRow := 2
	imageRow := 2

	err := readSheet(
		file,
		productsSheet,
		simpleRequiredHeaders,
		func(
			source RowSource,
			values rowValues,
		) {
			parseSimpleProductRow(
				result,
				source,
				values,
				&variantRow,
				&imageRow,
			)
		},
	)
	if err != nil {
		return nil, err
	}

	Validate(result)

	return result, nil
}

type simpleVariantValues struct {
	colors []string
	sizes  []string
	moqs   []string
	prices []string
	stocks []string
}

func parseSimpleProductRow(
	result *ParseResult,
	source RowSource,
	values rowValues,
	variantRow *int,
	imageRow *int,
) {
	productName := value(
		values,
		simpleHeaderProductName,
	)

	brand := value(
		values,
		simpleHeaderBrand,
	)

	categoryPath :=
		simpleCategoryPath(
			value(
				values,
				simpleHeaderCategory,
			),
			value(
				values,
				simpleHeaderSubcategory,
			),
			value(
				values,
				simpleHeaderChildCategory,
			),
		)

	productCode :=
		simpleProductCode(
			productName,
			brand,
		)

	result.Workbook.Products =
		append(
			result.Workbook.Products,
			ProductRow{
				Source:       source,
				ProductCode:  productCode,
				ProductName:  productName,
				CategoryPath: categoryPath,
				Brand:        brand,

				ShortDescription: value(
					values,
					simpleHeaderShortDescription,
				),

				Description: value(
					values,
					simpleHeaderLongDescription,
				),

				Status: valueOr(
					values,
					simpleHeaderStatus,
					"draft",
				),

				IsFeatured: false,
			},
		)

	lists := simpleVariantValues{
		colors: splitSimpleList(
			value(
				values,
				simpleHeaderColors,
			),
		),

		sizes: splitSimpleList(
			value(
				values,
				simpleHeaderSizes,
			),
		),

		moqs: splitSimpleList(
			value(
				values,
				simpleHeaderMOQ,
			),
		),

		prices: splitSimpleList(
			value(
				values,
				simpleHeaderPrice,
			),
		),

		stocks: splitSimpleList(
			value(
				values,
				simpleHeaderOpeningStock,
			),
		),
	}

	count := maxSimpleListLength(
		lists.colors,
		lists.sizes,
		lists.moqs,
		lists.prices,
		lists.stocks,
	)

	if count == 0 {
		count = 1
	}

	validateSimpleListLength(
		result,
		source,
		"colors",
		lists.colors,
		count,
	)

	validateSimpleListLength(
		result,
		source,
		"sizes",
		lists.sizes,
		count,
	)

	validateSimpleListLength(
		result,
		source,
		"minimum_order_quantity_(moq)",
		lists.moqs,
		count,
	)

	validateSimpleListLength(
		result,
		source,
		"price_(bdt)",
		lists.prices,
		count,
	)

	validateSimpleListLength(
		result,
		source,
		"opening_stock",
		lists.stocks,
		count,
	)

	if len(lists.prices) == 0 {
		addError(
			result,
			source,
			"price_(bdt)",
			"REQUIRED",
			"Price (BDT) is required",
		)
	}

	for index := 0; index < count; index++ {

		variantSource := RowSource{
			Sheet: sheetVariants,
			Row:   *variantRow,
		}

		*variantRow =
			*variantRow + 1

		result.Workbook.Variants =
			append(
				result.Workbook.Variants,
				VariantRow{
					Source: variantSource,

					ProductCode: productCode,

					SKU: simpleVariantSKU(
						productCode,
						index,
					),

					ColorName: simpleListValue(
						lists.colors,
						index,
					),

					Size: simpleListValue(
						lists.sizes,
						index,
					),

					PriceAmount: parseMoneyField(
						result,
						variantSource,
						"price",
						simpleListValue(
							lists.prices,
							index,
						),
						true,
					),

					Currency: "BDT",

					MinimumOrderQuantity: parseIntField(
						result,
						variantSource,
						"moq",
						simpleListValue(
							lists.moqs,
							index,
						),
						1,
					),

					/*
						Final commerce rule:
						MOQ is the minimum.
						After MOQ, quantity moves
						one whole unit at a time.
					*/
					OrderIncrement: 1,

					Stock: parseIntField(
						result,
						variantSource,
						"stock",
						simpleListValue(
							lists.stocks,
							index,
						),
						0,
					),

					ReorderLevel: 0,

					IsActive: true,
				},
			)
	}

	/*
		Image 1 becomes primary.
		Image 2-6 remain ordinary
		product-level images.
	*/
	for index := 1; index <= 6; index++ {

		imageURL := value(
			values,
			fmt.Sprintf(
				"image_%d",
				index,
			),
		)

		if imageURL == "" {
			continue
		}

		imageSource := RowSource{
			Sheet: sheetImages,
			Row:   *imageRow,
		}

		*imageRow =
			*imageRow + 1

		result.Workbook.Images =
			append(
				result.Workbook.Images,
				ImageRow{
					Source: imageSource,

					ProductCode: productCode,

					ImageURL: imageURL,

					AltText: fmt.Sprintf(
						"%s image %d",
						productName,
						index,
					),

					SortOrder: index - 1,

					IsPrimary: index == 1,
				},
			)
	}
}

func simpleCategoryPath(
	category string,
	subcategory string,
	child string,
) string {
	parts := make(
		[]string,
		0,
		3,
	)

	for _, part := range []string{
		category,
		subcategory,
		child,
	} {
		part =
			strings.TrimSpace(
				part,
			)

		if part == "" {
			continue
		}

		parts = append(
			parts,
			part,
		)
	}

	return strings.Join(
		parts,
		" > ",
	)
}

func splitSimpleList(
	raw string,
) []string {
	raw = strings.TrimSpace(
		raw,
	)

	if raw == "" {
		return nil
	}

	parts :=
		strings.Split(
			raw,
			"|",
		)

	result := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {

		result = append(
			result,
			strings.TrimSpace(
				part,
			),
		)
	}

	return result
}

func maxSimpleListLength(
	lists ...[]string,
) int {
	maximum := 0

	for _, list := range lists {

		if len(list) >
			maximum {

			maximum =
				len(list)
		}
	}

	return maximum
}

func validateSimpleListLength(
	result *ParseResult,
	source RowSource,
	field string,
	values []string,
	variantCount int,
) {
	/*
		Accepted:
		  0 values
		  1 value → broadcast
		  N values → one per variant
	*/
	if len(values) == 0 ||
		len(values) == 1 ||
		len(values) ==
			variantCount {

		return
	}

	addError(
		result,
		source,
		field,
		"SIMPLE_VARIANT_LENGTH_MISMATCH",
		fmt.Sprintf(
			"%s has %d value(s); use either 1 value or %d pipe-separated values to match this product's variants",
			field,
			len(values),
			variantCount,
		),
	)
}

func simpleListValue(
	values []string,
	index int,
) string {
	switch len(values) {
	case 0:
		return ""

	case 1:
		return values[0]

	default:
		if index >= 0 &&
			index < len(values) {

			return values[index]
		}

		return ""
	}
}

/*
The simple sheet has no product_code field.

Generate a deterministic identifier from
Product Name + Brand.

This deliberately excludes price, stock,
category and variant data so changing those
values later still targets the same product.
*/
func simpleProductCode(
	productName string,
	brand string,
) string {
	identity :=
		strings.ToLower(
			strings.TrimSpace(
				productName,
			),
		) +
			"|" +
			strings.ToLower(
				strings.TrimSpace(
					brand,
				),
			)

	sum := sha256.Sum256(
		[]byte(identity),
	)

	return "SIMPLE-" +
		strings.ToUpper(
			hex.EncodeToString(
				sum[:8],
			),
		)
}

func simpleVariantSKU(
	productCode string,
	index int,
) string {
	return fmt.Sprintf(
		"%s-V%02d",
		productCode,
		index+1,
	)
}

package catalogimport

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	sheetProducts   = "Products"
	sheetVariants   = "Variants"
	sheetImages     = "Images"
	sheetPriceTiers = "PriceTiers"
	sheetCategories = "Categories"
)

type rowValues map[string]string

func Parse(
	reader io.Reader,
) (*ParseResult, error) {
	file, err := excelize.OpenReader(
		reader,
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

	result := &ParseResult{
		Workbook: Workbook{
			Products:   make([]ProductRow, 0),
			Variants:   make([]VariantRow, 0),
			Images:     make([]ImageRow, 0),
			PriceTiers: make([]PriceTierRow, 0),
			Categories: make([]CategoryRow, 0),
		},
		Errors: make([]ValidationError, 0),
	}

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

	variantsSheet, ok := findSheet(
		file,
		sheetVariants,
	)
	if !ok {
		return nil, fmt.Errorf(
			"required worksheet %q not found",
			sheetVariants,
		)
	}

	err = readSheet(
		file,
		productsSheet,
		[]string{
			"product_code",
			"product_name",
			"category_path",
		},
		func(source RowSource, values rowValues) {
			parseProductRow(
				result,
				source,
				values,
			)
		},
	)
	if err != nil {
		return nil, err
	}

	err = readSheet(
		file,
		variantsSheet,
		[]string{
			"product_code",
			"sku",
			"price",
		},
		func(source RowSource, values rowValues) {
			parseVariantRow(
				result,
				source,
				values,
			)
		},
	)
	if err != nil {
		return nil, err
	}

	if sheet, exists := findSheet(
		file,
		sheetImages,
	); exists {
		err = readSheet(
			file,
			sheet,
			[]string{
				"product_code",
			},
			func(source RowSource, values rowValues) {
				parseImageRow(
					result,
					source,
					values,
				)
			},
		)
		if err != nil {
			return nil, err
		}
	}

	if sheet, exists := findSheet(
		file,
		sheetPriceTiers,
	); exists {
		err = readSheet(
			file,
			sheet,
			[]string{
				"sku",
				"min_quantity",
				"unit_price",
			},
			func(source RowSource, values rowValues) {
				parsePriceTierRow(
					result,
					source,
					values,
				)
			},
		)
		if err != nil {
			return nil, err
		}
	}

	if sheet, exists := findSheet(
		file,
		sheetCategories,
	); exists {
		err = readSheet(
			file,
			sheet,
			[]string{
				"category_path",
			},
			func(source RowSource, values rowValues) {
				parseCategoryRow(
					result,
					source,
					values,
				)
			},
		)
		if err != nil {
			return nil, err
		}
	}

	Validate(result)

	return result, nil
}

func findSheet(
	file *excelize.File,
	expected string,
) (string, bool) {
	for _, sheet := range file.GetSheetList() {
		if strings.EqualFold(
			strings.TrimSpace(sheet),
			expected,
		) {
			return sheet, true
		}
	}

	return "", false
}

func readSheet(
	file *excelize.File,
	sheet string,
	requiredHeaders []string,
	handler func(RowSource, rowValues),
) error {
	rows, err := file.Rows(sheet)
	if err != nil {
		return fmt.Errorf(
			"open worksheet %q: %w",
			sheet,
			err,
		)
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf(
			"worksheet %q is empty",
			sheet,
		)
	}

	headerCells, err := rows.Columns()
	if err != nil {
		return fmt.Errorf(
			"read headers from %q: %w",
			sheet,
			err,
		)
	}

	headerIndexes := make(
		map[string]int,
		len(headerCells),
	)

	for index, value := range headerCells {
		header := normalizeHeader(value)

		if header == "" {
			continue
		}

		if _, exists := headerIndexes[header]; exists {
			return fmt.Errorf(
				"worksheet %q has duplicate header %q",
				sheet,
				header,
			)
		}

		headerIndexes[header] = index
	}

	for _, required := range requiredHeaders {
		if _, exists := headerIndexes[required]; !exists {
			return fmt.Errorf(
				"worksheet %q is missing required column %q",
				sheet,
				required,
			)
		}
	}

	rowNumber := 1

	for rows.Next() {
		rowNumber++

		cells, err := rows.Columns()
		if err != nil {
			return fmt.Errorf(
				"read %s row %d: %w",
				sheet,
				rowNumber,
				err,
			)
		}

		if blankRow(cells) {
			continue
		}

		values := make(
			rowValues,
			len(headerIndexes),
		)

		for header, index := range headerIndexes {
			if index >= len(cells) {
				values[header] = ""
				continue
			}

			values[header] = strings.TrimSpace(
				cells[index],
			)
		}

		handler(
			RowSource{
				Sheet: sheet,
				Row:   rowNumber,
			},
			values,
		)
	}

	if err := rows.Error(); err != nil {
		return fmt.Errorf(
			"iterate worksheet %q: %w",
			sheet,
			err,
		)
	}

	return nil
}

func parseProductRow(
	result *ParseResult,
	source RowSource,
	values rowValues,
) {
	row := ProductRow{
		Source:           source,
		ProductCode:      value(values, "product_code"),
		ProductName:      value(values, "product_name"),
		CategoryPath:     value(values, "category_path"),
		Slug:             value(values, "slug"),
		Brand:            value(values, "brand"),
		ShortDescription: value(values, "short_description"),
		Description:      value(values, "description"),
		Status:           valueOr(values, "status", "draft"),
		IsFeatured: parseBoolField(
			result,
			source,
			"is_featured",
			value(values, "is_featured"),
			false,
		),
	}

	result.Workbook.Products = append(
		result.Workbook.Products,
		row,
	)
}

func parseVariantRow(
	result *ParseResult,
	source RowSource,
	values rowValues,
) {
	price := parseMoneyField(
		result,
		source,
		"price",
		value(values, "price"),
		true,
	)

	compareAt := parseOptionalMoneyField(
		result,
		source,
		"compare_at_price",
		value(values, "compare_at_price"),
	)

	cost := parseOptionalMoneyField(
		result,
		source,
		"cost",
		value(values, "cost"),
	)

	weight := parseOptionalIntField(
		result,
		source,
		"weight_grams",
		value(values, "weight_grams"),
	)

	row := VariantRow{
		Source:      source,
		ProductCode: value(values, "product_code"),
		SKU:         value(values, "sku"),
		ColorName:   value(values, "color_name"),
		ColorHex:    value(values, "color_hex"),
		Size:        value(values, "size"),
		PriceAmount: price,
		Currency: strings.ToUpper(
			valueOr(
				values,
				"currency",
				"BDT",
			),
		),
		MinimumOrderQuantity: parseIntField(
			result,
			source,
			"moq",
			value(values, "moq"),
			1,
		),
		OrderIncrement: parseIntField(
			result,
			source,
			"order_increment",
			value(values, "order_increment"),
			1,
		),
		Stock: parseIntField(
			result,
			source,
			"stock",
			value(values, "stock"),
			0,
		),
		ReorderLevel: parseIntField(
			result,
			source,
			"reorder_level",
			value(values, "reorder_level"),
			0,
		),
		Barcode:     value(values, "barcode"),
		WeightGrams: weight,
		IsActive: parseBoolField(
			result,
			source,
			"is_active",
			value(values, "is_active"),
			true,
		),
		CompareAtPriceAmount: compareAt,
		CostAmount:           cost,
	}

	result.Workbook.Variants = append(
		result.Workbook.Variants,
		row,
	)
}

func parseImageRow(
	result *ParseResult,
	source RowSource,
	values rowValues,
) {
	row := ImageRow{
		Source:      source,
		ProductCode: value(values, "product_code"),
		SKU:         value(values, "sku"),
		ImageURL:    value(values, "image_url"),
		ImageFile:   value(values, "image_file"),
		AltText:     value(values, "alt_text"),
		SortOrder: parseIntField(
			result,
			source,
			"sort_order",
			value(values, "sort_order"),
			0,
		),
		IsPrimary: parseBoolField(
			result,
			source,
			"is_primary",
			value(values, "is_primary"),
			false,
		),
	}

	result.Workbook.Images = append(
		result.Workbook.Images,
		row,
	)
}

func parsePriceTierRow(
	result *ParseResult,
	source RowSource,
	values rowValues,
) {
	row := PriceTierRow{
		Source: source,
		SKU:    value(values, "sku"),
		MinQuantity: parseIntField(
			result,
			source,
			"min_quantity",
			value(values, "min_quantity"),
			0,
		),
		UnitPriceAmount: parseMoneyField(
			result,
			source,
			"unit_price",
			value(values, "unit_price"),
			true,
		),
	}

	result.Workbook.PriceTiers = append(
		result.Workbook.PriceTiers,
		row,
	)
}

func parseCategoryRow(
	result *ParseResult,
	source RowSource,
	values rowValues,
) {
	row := CategoryRow{
		Source:       source,
		CategoryPath: value(values, "category_path"),
		Description:  value(values, "description"),
		ImageURL:     value(values, "image_url"),
		IconURL:      value(values, "icon_url"),
		SortOrder: parseIntField(
			result,
			source,
			"sort_order",
			value(values, "sort_order"),
			0,
		),
		IsActive: parseBoolField(
			result,
			source,
			"is_active",
			value(values, "is_active"),
			true,
		),
	}

	result.Workbook.Categories = append(
		result.Workbook.Categories,
		row,
	)
}

func value(
	values rowValues,
	key string,
) string {
	return strings.TrimSpace(values[key])
}

func valueOr(
	values rowValues,
	key string,
	fallback string,
) string {
	v := value(values, key)

	if v == "" {
		return fallback
	}

	return v
}

func normalizeHeader(value string) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	value = strings.ReplaceAll(
		value,
		" ",
		"_",
	)

	value = strings.ReplaceAll(
		value,
		"-",
		"_",
	)

	return value
}

func blankRow(cells []string) bool {
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}

	return true
}

func parseMoneyField(
	result *ParseResult,
	source RowSource,
	field string,
	raw string,
	required bool,
) int64 {
	if strings.TrimSpace(raw) == "" {
		if required {
			addError(
				result,
				source,
				field,
				"REQUIRED",
				field+" is required",
			)
		}

		return 0
	}

	amount, err := parseMoneyToPoisha(raw)
	if err != nil {
		addError(
			result,
			source,
			field,
			"INVALID_MONEY",
			err.Error(),
		)

		return 0
	}

	return amount
}

func parseOptionalMoneyField(
	result *ParseResult,
	source RowSource,
	field string,
	raw string,
) *int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	amount, err := parseMoneyToPoisha(raw)
	if err != nil {
		addError(
			result,
			source,
			field,
			"INVALID_MONEY",
			err.Error(),
		)

		return nil
	}

	return &amount
}

func parseIntField(
	result *ParseResult,
	source RowSource,
	field string,
	raw string,
	fallback int,
) int {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return fallback
	}

	raw = strings.ReplaceAll(raw, ",", "")

	number, err := strconv.Atoi(raw)
	if err != nil {
		addError(
			result,
			source,
			field,
			"INVALID_INTEGER",
			field+" must be a whole number",
		)

		return fallback
	}

	return number
}

func parseOptionalIntField(
	result *ParseResult,
	source RowSource,
	field string,
	raw string,
) *int {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return nil
	}

	number := parseIntField(
		result,
		source,
		field,
		raw,
		0,
	)

	return &number
}

func parseBoolField(
	result *ParseResult,
	source RowSource,
	field string,
	raw string,
	fallback bool,
) bool {
	raw = strings.ToLower(
		strings.TrimSpace(raw),
	)

	if raw == "" {
		return fallback
	}

	switch raw {
	case "true", "1", "yes", "y":
		return true

	case "false", "0", "no", "n":
		return false

	default:
		addError(
			result,
			source,
			field,
			"INVALID_BOOLEAN",
			field+" must be true/false, yes/no, or 1/0",
		)

		return fallback
	}
}

func addError(
	result *ParseResult,
	source RowSource,
	field string,
	code string,
	message string,
) {
	result.Errors = append(
		result.Errors,
		ValidationError{
			Sheet:   source.Sheet,
			Row:     source.Row,
			Field:   field,
			Code:    code,
			Message: message,
		},
	)
}

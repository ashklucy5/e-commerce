package admin

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func BuildFinanceImportTemplate() (
	[]byte,
	error,
) {
	book :=
		excelize.NewFile()

	defer func() {
		_ = book.Close()
	}()

	defaultSheet :=
		book.GetSheetName(
			0,
		)

	if defaultSheet == "" {
		return nil,
			fmt.Errorf(
				"create finance import workbook: default sheet is missing",
			)
	}

	if err :=
		book.SetSheetName(
			defaultSheet,
			FinanceImportSheetName,
		); err != nil {

		return nil,
			fmt.Errorf(
				"name finance import sheet: %w",
				err,
			)
	}

	for columnIndex, header := range financeImportHeaders {

		cell, err :=
			excelize.CoordinatesToCellName(
				columnIndex+1,
				1,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"finance import header cell: %w",
					err,
				)
		}

		if err :=
			book.SetCellValue(
				FinanceImportSheetName,
				cell,
				header,
			); err != nil {

			return nil,
				fmt.Errorf(
					"write finance import header: %w",
					err,
				)
		}
	}

	examples :=
		[][]any{
			{
				"expense",
				"2026-09-30",
				"salary",
				"staff",
				45000,
				"",
				"BDT",
				"",
				"",
				"",
				"staff@example.com",
				"",
				"September salary",
				"PAYROLL-2026-09",
				"",
			},
			{
				"expense",
				"2026-09-30",
				"marketing",
				"business",
				80000,
				"",
				"BDT",
				"",
				"",
				"",
				"",
				"",
				"September campaign spend",
				"META-2026-09",
				"",
			},
			{
				"expense",
				"2026-09-30",
				"packaging",
				"product",
				12000,
				"",
				"BDT",
				"PRD-100",
				"",
				"",
				"",
				"",
				"Packaging expense",
				"PKG-100",
				"",
			},
			{
				"expense",
				"2026-09-30",
				"warehouse",
				"warehouse",
				25000,
				"",
				"BDT",
				"",
				"",
				"",
				"",
				"DHAKA-MAIN",
				"Warehouse operating expense",
				"WH-SEP-2026",
				"",
			},
			{
				"variant_cost",
				"2026-09-30",
				"",
				"",
				"",
				420,
				"BDT",
				"",
				"SKU-A-RED",
				"",
				"",
				"",
				"Supplier buying cost",
				"PO-771",
				"TRUE",
			},
		}

	for rowIndex, row := range examples {

		for columnIndex, value := range row {

			cell, err :=
				excelize.CoordinatesToCellName(
					columnIndex+1,
					rowIndex+2,
				)
			if err != nil {
				return nil,
					fmt.Errorf(
						"finance import example cell: %w",
						err,
					)
			}

			if err :=
				book.SetCellValue(
					FinanceImportSheetName,
					cell,
					value,
				); err != nil {

				return nil,
					fmt.Errorf(
						"write finance import example: %w",
						err,
					)
			}
		}
	}

	headerStyle, err :=
		book.NewStyle(
			&excelize.Style{
				Font: &excelize.Font{
					Bold: true,
				},

				Alignment: &excelize.Alignment{
					Vertical: "center",
				},

				Fill: excelize.Fill{
					Type: "pattern",

					Pattern: 1,

					Color: []string{
						"#E8EEF8",
					},
				},
			},
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create finance import header style: %w",
				err,
			)
	}

	if err :=
		book.SetCellStyle(
			FinanceImportSheetName,
			"A1",
			"O1",
			headerStyle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"style finance import header: %w",
				err,
			)
	}

	if err :=
		book.SetPanes(
			FinanceImportSheetName,
			&excelize.Panes{
				Freeze: true,

				Split: false,

				XSplit: 0,

				YSplit: 1,

				TopLeftCell: "A2",

				ActivePane: "bottomLeft",
			},
		); err != nil {

		return nil,
			fmt.Errorf(
				"freeze finance import header: %w",
				err,
			)
	}

	widths :=
		map[string]float64{
			"A": 16,
			"B": 15,
			"C": 24,
			"D": 14,
			"E": 16,
			"F": 20,
			"G": 12,
			"H": 20,
			"I": 20,
			"J": 38,
			"K": 32,
			"L": 20,
			"M": 34,
			"N": 24,
			"O": 14,
		}

	for column, width := range widths {

		if err :=
			book.SetColWidth(
				FinanceImportSheetName,
				column,
				column,
				width,
			); err != nil {

			return nil,
				fmt.Errorf(
					"size finance import column %s: %w",
					column,
					err,
				)
		}
	}

	if err :=
		book.AutoFilter(
			FinanceImportSheetName,
			"A1:O1",
			nil,
		); err != nil {

		return nil,
			fmt.Errorf(
				"add finance import filter: %w",
				err,
			)
	}

	if _, err :=
		book.NewSheet(
			"Instructions",
		); err != nil {

		return nil,
			fmt.Errorf(
				"create finance import instructions sheet: %w",
				err,
			)
	}

	instructions :=
		[][]any{
			{
				"Finance import template",
				"Version",
				FinanceImportTemplateVersion,
			},
			{
				"Rule",
				"Meaning",
			},
			{
				"record_type",
				"Use expense or variant_cost.",
			},
			{
				"date",
				"Business effective date. Recommended format: YYYY-MM-DD.",
			},
			{
				"expense / scope",
				"Use business, order, product, variant, staff, or warehouse.",
			},
			{
				"expense / category",
				"delivery, payment_fee, china_freight, customs, packaging, marketing, warehouse, salary, staff_benefit, rent, utilities, software, professional_service, bank_fee, insurance, tax_fee, office, travel, other.",
			},
			{
				"expense / amount",
				"Positive whole-number amount using the same monetary-unit convention as the commerce backend.",
			},
			{
				"product scope",
				"Fill product_code only.",
			},
			{
				"variant scope",
				"Fill sku only.",
			},
			{
				"order scope",
				"Fill order_id only.",
			},
			{
				"staff scope",
				"Fill staff_email only.",
			},
			{
				"warehouse scope",
				"Fill warehouse_code only.",
			},
			{
				"business scope",
				"Leave all target identifier columns blank.",
			},
			{
				"variant_cost",
				"Fill date, unit_cost_amount, sku and optionally currency, description, reference, set_current.",
			},
			{
				"set_current",
				"TRUE updates the variant's current cost_amount. FALSE or blank records historical cost only.",
			},
			{
				"currency",
				"Three-letter currency code. Expense rows default to BDT. Variant-cost rows default to the variant currency.",
			},
			{
				"all-or-nothing",
				"Rows are validated first. A workbook with any invalid row cannot be applied.",
			},
			{
				"duplicate protection",
				"Uploading the exact same workbook again returns the existing import batch instead of duplicating costs.",
			},
			{
				"blank rows",
				"Blank rows are ignored.",
			},
			{
				"maximum rows",
				FinanceImportMaxRows,
			},
		}

	for rowIndex, row := range instructions {

		for columnIndex, value := range row {

			cell, err :=
				excelize.CoordinatesToCellName(
					columnIndex+1,
					rowIndex+1,
				)
			if err != nil {
				return nil,
					fmt.Errorf(
						"finance import instruction cell: %w",
						err,
					)
			}

			if err :=
				book.SetCellValue(
					"Instructions",
					cell,
					value,
				); err != nil {

				return nil,
					fmt.Errorf(
						"write finance import instruction: %w",
						err,
					)
			}
		}
	}

	if err :=
		book.SetCellStyle(
			"Instructions",
			"A1",
			"C1",
			headerStyle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"style finance import instructions title: %w",
				err,
			)
	}

	if err :=
		book.SetCellStyle(
			"Instructions",
			"A2",
			"B2",
			headerStyle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"style finance import instructions header: %w",
				err,
			)
	}

	if err :=
		book.SetColWidth(
			"Instructions",
			"A",
			"A",
			24,
		); err != nil {

		return nil,
			fmt.Errorf(
				"size finance import instructions column A: %w",
				err,
			)
	}

	if err :=
		book.SetColWidth(
			"Instructions",
			"B",
			"B",
			110,
		); err != nil {

		return nil,
			fmt.Errorf(
				"size finance import instructions column B: %w",
				err,
			)
	}

	if err :=
		book.SetColWidth(
			"Instructions",
			"C",
			"C",
			18,
		); err != nil {

		return nil,
			fmt.Errorf(
				"size finance import instructions column C: %w",
				err,
			)
	}

	book.SetActiveSheet(
		0,
	)

	buffer, err :=
		book.WriteToBuffer()
	if err != nil {
		return nil,
			fmt.Errorf(
				"write finance import template: %w",
				err,
			)
	}

	return buffer.Bytes(),
		nil
}

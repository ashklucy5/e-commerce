package main

import (
	"fmt"
	"os"

	"project.local/commerce-api/internal/catalogimport"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(
			os.Stderr,
			"usage: go run ./cmd/catalog-import-check <catalog.xlsx>",
		)
		os.Exit(2)
	}

	filePath := os.Args[1]

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"open workbook: %v\n",
			err,
		)
		os.Exit(1)
	}
	defer file.Close()

	result, err := catalogimport.Parse(file)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"parse workbook: %v\n",
			err,
		)
		os.Exit(1)
	}

	fmt.Println("Catalog Import Validation")
	fmt.Println("-------------------------")

	fmt.Printf(
		"Products:    %d\n",
		len(result.Workbook.Products),
	)

	fmt.Printf(
		"Variants:    %d\n",
		len(result.Workbook.Variants),
	)

	fmt.Printf(
		"Images:      %d\n",
		len(result.Workbook.Images),
	)

	fmt.Printf(
		"PriceTiers:  %d\n",
		len(result.Workbook.PriceTiers),
	)

	fmt.Printf(
		"Categories:  %d\n",
		len(result.Workbook.Categories),
	)

	fmt.Printf(
		"Errors:      %d\n",
		len(result.Errors),
	)

	if !result.Valid() {
		fmt.Println()
		fmt.Println("Validation errors:")
		fmt.Println()

		for _, validationError := range result.Errors {
			if validationError.Field != "" {
				fmt.Printf(
					"%s row %d [%s] %s: %s\n",
					validationError.Sheet,
					validationError.Row,
					validationError.Code,
					validationError.Field,
					validationError.Message,
				)

				continue
			}

			fmt.Printf(
				"%s row %d [%s]: %s\n",
				validationError.Sheet,
				validationError.Row,
				validationError.Code,
				validationError.Message,
			)
		}

		fmt.Println()
		fmt.Println(
			"Validation failed. No catalog data was written to the database.",
		)

		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Validation: OK")

	printPreview(result)

	fmt.Println()
	fmt.Println(
		"No catalog data was written to the database.",
	)
}

func printPreview(
	result *catalogimport.ParseResult,
) {
	fmt.Println()
	fmt.Println("Import Preview")
	fmt.Println("--------------")

	for _, product := range result.Workbook.Products {
		fmt.Printf(
			"Product: %s | %s | %s\n",
			product.ProductCode,
			product.ProductName,
			product.CategoryPath,
		)
	}

	fmt.Println()

	for _, variant := range result.Workbook.Variants {
		fmt.Printf(
			"Variant: %s | %s %s | price=%d poisha | MOQ=%d | increment=%d | stock=%d\n",
			variant.SKU,
			variant.ColorName,
			variant.Size,
			variant.PriceAmount,
			variant.MinimumOrderQuantity,
			variant.OrderIncrement,
			variant.Stock,
		)

		for _, tier := range result.Workbook.PriceTiers {
			if tier.SKU != variant.SKU {
				continue
			}

			fmt.Printf(
				"  Tier: %d+ units -> %d poisha\n",
				tier.MinQuantity,
				tier.UnitPriceAmount,
			)
		}
	}

	fmt.Println()

	for _, image := range result.Workbook.Images {
		target := "product"

		if image.SKU != "" {
			target = "variant " + image.SKU
		}

		source := image.ImageURL
		if source == "" {
			source = image.ImageFile
		}

		fmt.Printf(
			"Image: %s | %s\n",
			target,
			source,
		)
	}
}

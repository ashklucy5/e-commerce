package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"project.local/commerce-api/internal/catalogimport"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(
			os.Stderr,
			"usage: go run ./cmd/catalog-import-stage <catalog.xlsx>",
		)

		os.Exit(2)
	}

	filePath := os.Args[1]

	absolutePath, err := filepath.Abs(
		filePath,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"resolve workbook path: %v\n",
			err,
		)

		os.Exit(1)
	}

	checksum, err := checksumFile(
		absolutePath,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"calculate workbook checksum: %v\n",
			err,
		)

		os.Exit(1)
	}

	file, err := os.Open(
		absolutePath,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"open workbook: %v\n",
			err,
		)

		os.Exit(1)
	}
	defer file.Close()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"configuration error: %v\n",
			err,
		)

		os.Exit(1)
	}

	ctx := context.Background()

	db, err := database.NewPostgres(
		ctx,
		cfg,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"postgres startup failed: %v\n",
			err,
		)

		os.Exit(1)
	}
	defer db.Close()

	repository := catalogimport.NewRepository(
		db,
	)

	service := catalogimport.NewService(
		repository,
	)

	storageKey := "local://" +
		filepath.ToSlash(
			absolutePath,
		)

	result, err := service.Stage(
		ctx,
		filepath.Base(absolutePath),
		storageKey,
		checksum,
		file,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"catalog import staging failed: %v\n",
			err,
		)

		os.Exit(1)
	}

	fmt.Println("Catalog Import Staging")
	fmt.Println("----------------------")

	fmt.Printf(
		"Batch ID:     %s\n",
		result.Batch.ID,
	)
	fmt.Println()

	fmt.Printf(
		"Status:       %s\n",
		result.Batch.Status,
	)

	fmt.Printf(
		"Total rows:   %d\n",
		result.Batch.TotalRows,
	)

	fmt.Printf(
		"Valid rows:   %d\n",
		result.Batch.ValidRows,
	)

	fmt.Printf(
		"Failed rows:  %d\n",
		result.Batch.FailedRows,
	)
	fmt.Println()

	fmt.Printf(
		"Products:     create=%d update=%d\n",
		result.Batch.CreatedProducts,
		result.Batch.UpdatedProducts,
	)

	fmt.Printf(
		"Variants:     create=%d update=%d\n",
		result.Batch.CreatedVariants,
		result.Batch.UpdatedVariants,
	)

	fmt.Printf(
		"Categories:   create=%d\n",
		result.Batch.CreatedCategories,
	)

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("Validation errors:")

		for _, validationError := range result.Errors {

			fmt.Printf(
				"%s row %d [%s] %s: %s\n",
				validationError.Sheet,
				validationError.Row,
				validationError.Code,
				validationError.Field,
				validationError.Message,
			)
		}

		fmt.Println()
		fmt.Println(
			"Batch was recorded, but validation failed.",
		)

		fmt.Println(
			"Catalog data was NOT modified.",
		)

		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(
		"Import batch staged successfully.",
	)

	fmt.Println(
		"Catalog data was NOT modified.",
	)
}

func checksumFile(
	path string,
) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "",
			fmt.Errorf(
				"open file: %w",
				err,
			)
	}
	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(
		hash,
		file,
	); err != nil {
		return "",
			fmt.Errorf(
				"hash file: %w",
				err,
			)
	}

	return fmt.Sprintf(
		"%x",
		hash.Sum(nil),
	), nil
}

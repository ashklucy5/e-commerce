package main

import (
	"context"
	"fmt"
	"os"

	"project.local/commerce-api/internal/catalogimport"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(
			os.Stderr,
			"usage: go run ./cmd/catalog-import-apply <batch-id>",
		)

		os.Exit(2)
	}

	batchID := os.Args[1]

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

	repository :=
		catalogimport.NewRepository(db)

	service :=
		catalogimport.NewService(
			repository,
		)

	result, err := service.Apply(
		ctx,
		batchID,
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"catalog import apply failed: %v\n",
			err,
		)

		os.Exit(1)
	}

	fmt.Println("Catalog Import Apply")
	fmt.Println("--------------------")

	fmt.Printf(
		"Batch ID:     %s\n",
		result.BatchID,
	)

	fmt.Printf(
		"Status:       %s\n",
		result.Status,
	)

	fmt.Printf(
		"Applied rows: %d\n",
		result.AppliedRows,
	)

	fmt.Printf(
		"Skipped rows: %d\n",
		result.SkippedRows,
	)

	fmt.Println()
	fmt.Println(
		"Catalog import applied successfully.",
	)
}

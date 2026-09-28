package main

import (
	"context"
	"log"

	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/seeds"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"configuration error: %v",
			err,
		)
	}

	ctx := context.Background()

	db, err := database.NewPostgres(
		ctx,
		cfg,
	)
	if err != nil {
		log.Fatalf(
			"postgres startup failed: %v",
			err,
		)
	}
	defer db.Close()

	if err := seeds.SeedCategories(
		ctx,
		db,
	); err != nil {
		log.Fatalf(
			"category seed failed: %v",
			err,
		)
	}

	log.Println(
		"development categories seeded successfully",
	)

	if err := seeds.SeedProductCodeNamespaces(
		ctx,
		db,
	); err != nil {
		log.Fatalf(
			"product-code namespace seed failed: %v",
			err,
		)
	}

	log.Println(
		"product-code namespaces seeded successfully",
	)

	if err := seeds.SeedProducts(
		ctx,
		db,
	); err != nil {
		log.Fatalf(
			"product seed failed: %v",
			err,
		)
	}

	log.Println(
		"development products seeded successfully",
	)

	log.Println(
		"development catalog seed completed successfully",
	)
}

package main

import (
	"context"
	"log"
	"time"

	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/search"
)

func main() {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			20*time.Minute,
		)
	defer cancel()

	cfg, err :=
		config.Load()
	if err != nil {
		log.Fatalf(
			"load configuration: %v",
			err,
		)
	}

	db, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		log.Fatalf(
			"connect PostgreSQL: %v",
			err,
		)
	}
	defer db.Close()

	embedder, err :=
		search.NewVoyageEmbedderFromEnv()
	if err != nil {
		log.Fatalf(
			"configure semantic embeddings: %v",
			err,
		)
	}

	repository :=
		search.NewRepository(
			db,
		)

	indexer :=
		search.NewIndexer(
			repository,
			embedder,
		)

	stats, err :=
		indexer.Run(
			ctx,
			32,
		)
	if err != nil {
		log.Fatalf(
			"reindex semantic search: %v",
			err,
		)
	}

	log.Printf(
		"semantic search indexing complete: scanned=%d indexed=%d unchanged=%d deleted=%d model=%s",
		stats.Scanned,
		stats.Indexed,
		stats.Unchanged,
		stats.Deleted,
		embedder.Model(),
	)
}

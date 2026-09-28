package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type EmbeddingIndexStore interface {
	ListEmbeddingDocuments(
		ctx context.Context,
	) ([]EmbeddingDocument, error)

	UpsertProductEmbedding(
		ctx context.Context,
		productID string,
		documentText string,
		documentHash string,
		embedding []float32,
		model string,
	) error

	DeleteInactiveProductEmbeddings(
		ctx context.Context,
	) (int64, error)
}

type IndexStats struct {
	Scanned   int
	Indexed   int
	Unchanged int
	Deleted   int64
}

type Indexer struct {
	store    EmbeddingIndexStore
	embedder Embedder
}

type pendingEmbeddingDocument struct {
	EmbeddingDocument
	Hash string
}

func NewIndexer(
	store EmbeddingIndexStore,
	embedder Embedder,
) *Indexer {
	return &Indexer{
		store:    store,
		embedder: embedder,
	}
}

func (i *Indexer) Run(
	ctx context.Context,
	batchSize int,
) (IndexStats, error) {
	if i.embedder == nil {
		return IndexStats{}, fmt.Errorf(
			"embedding provider is not configured",
		)
	}

	if batchSize <= 0 {
		batchSize = 32
	}

	if batchSize > 1000 {
		batchSize = 1000
	}

	documents, err :=
		i.store.ListEmbeddingDocuments(
			ctx,
		)
	if err != nil {
		return IndexStats{}, err
	}

	stats :=
		IndexStats{
			Scanned: len(
				documents,
			),
		}

	pending :=
		make(
			[]pendingEmbeddingDocument,
			0,
			len(documents),
		)

	model :=
		i.embedder.Model()

	for _, document := range documents {
		hash :=
			hashEmbeddingDocument(
				document,
			)

		if document.StoredHash ==
			hash &&
			document.StoredModel ==
				model {
			stats.Unchanged++
			continue
		}

		pending =
			append(
				pending,
				pendingEmbeddingDocument{
					EmbeddingDocument: document,

					Hash: hash,
				},
			)
	}

	multimodalEmbedder, hasMultimodal :=
		i.embedder.(MultimodalDocumentEmbedder)

	for start := 0; start < len(pending); start += batchSize {
		end :=
			start +
				batchSize

		if end >
			len(pending) {
			end =
				len(pending)
		}

		batch :=
			pending[start:end]

		var embeddings [][]float32

		if hasMultimodal {
			inputs :=
				make(
					[]MultimodalDocument,
					len(batch),
				)

			for index := range batch {
				inputs[index] =
					MultimodalDocument{
						Text: batch[index].Text,

						ImageURL: batch[index].ImageURL,
					}
			}

			embeddings, err =
				multimodalEmbedder.
					EmbedMultimodalDocuments(
						ctx,
						inputs,
					)
		} else {
			texts :=
				make(
					[]string,
					len(batch),
				)

			for index := range batch {
				texts[index] =
					batch[index].Text
			}

			embeddings, err =
				i.embedder.
					EmbedDocuments(
						ctx,
						texts,
					)
		}

		if err != nil {
			return stats,
				fmt.Errorf(
					"embed product batch starting at %d: %w",
					start,
					err,
				)
		}

		if len(embeddings) !=
			len(batch) {
			return stats,
				fmt.Errorf(
					"embedding batch size mismatch: got %d vectors for %d products",
					len(embeddings),
					len(batch),
				)
		}

		for index := range batch {
			if err :=
				i.store.UpsertProductEmbedding(
					ctx,
					batch[index].ProductID,
					batch[index].Text,
					batch[index].Hash,
					embeddings[index],
					model,
				); err != nil {
				return stats,
					fmt.Errorf(
						"index product %s: %w",
						batch[index].ProductID,
						err,
					)
			}

			stats.Indexed++
		}
	}

	deleted, err :=
		i.store.DeleteInactiveProductEmbeddings(
			ctx,
		)
	if err != nil {
		return stats, err
	}

	stats.Deleted =
		deleted

	return stats, nil
}

func hashEmbeddingDocument(
	document EmbeddingDocument,
) string {
	imageURL :=
		normalizedRemoteImageURL(
			document.ImageURL,
		)

	payload :=
		document.Text +
			"\nPRIMARY_IMAGE_URL:" +
			imageURL

	sum :=
		sha256.Sum256(
			[]byte(
				payload,
			),
		)

	return hex.EncodeToString(
		sum[:],
	)
}

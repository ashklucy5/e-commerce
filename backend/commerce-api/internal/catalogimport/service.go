package catalogimport

import (
	"context"
	"fmt"
	"io"
	"strings"

	"project.local/commerce-api/internal/platform/storage"
)

type Service struct {
	repository *Repository
	storage    *storage.Gateway
}

/*
NewService remains backward-compatible.

Existing callers can still use:

	catalogimport.NewService(repository)

The Admin ZIP-aware path can use:

	catalogimport.NewService(repository, storageGateway)
*/
func NewService(
	repository *Repository,
	storageGateways ...*storage.Gateway,
) *Service {
	var storageGateway *storage.Gateway

	if len(storageGateways) > 0 {
		storageGateway =
			storageGateways[0]
	}

	if storageGateway == nil {
		storageGateway =
			storage.NewGateway(nil)
	}

	return &Service{
		repository: repository,
		storage:    storageGateway,
	}
}

/*
Stage remains backward-compatible.

Existing callers can still use:

	service.Stage(
		ctx,
		filename,
		storageKey,
		checksum,
		reader,
	)

The Admin ZIP-aware path can additionally use:

	service.Stage(
		ctx,
		filename,
		storageKey,
		checksum,
		reader,
		imageAssets,
	)
*/
func (s *Service) Stage(
	ctx context.Context,
	sourceFilename string,
	storageKey string,
	checksum string,
	reader io.Reader,
	imageAssetMaps ...map[string]string,
) (*StageResult, error) {
	sourceFilename =
		strings.TrimSpace(
			sourceFilename,
		)

	storageKey =
		strings.TrimSpace(
			storageKey,
		)

	if sourceFilename == "" {
		return nil,
			fmt.Errorf(
				"source filename is required",
			)
	}

	if storageKey == "" {
		return nil,
			fmt.Errorf(
				"file storage key is required",
			)
	}

	batch, err :=
		s.repository.CreateBatch(
			ctx,
			sourceFilename,
			storageKey,
			checksum,
		)
	if err != nil {
		return nil, err
	}

	/*
		ParseAuto detects:

		1. Advanced workbook:
		   Products + Variants

		2. Simple workbook:
		   Products only

		Both formats become the same internal
		Workbook structure before validation,
		planning and Apply.
	*/
	parseResult, err :=
		ParseAuto(
			reader,
		)

	if err != nil {
		message :=
			fmt.Sprintf(
				"parse workbook: %v",
				err,
			)

		_ =
			s.repository.FailBatch(
				ctx,
				batch.ID,
				message,
			)

		return nil,
			fmt.Errorf(
				"batch %s: %w",
				batch.ID,
				err,
			)
	}

	/*
		Optional filename -> object-storage-key map.

		Old XLSX-only callers pass no map and
		continue working exactly as before.

		The Admin ZIP workflow uploads image
		files first, then supplies the resulting
		storage keys in this map.
	*/
	var imageAssets map[string]string

	if len(imageAssetMaps) > 0 {
		imageAssets =
			imageAssetMaps[0]
	}

	if len(imageAssets) > 0 {
		s.resolveImageFiles(
			parseResult,
			imageAssets,
		)
	}

	savedBatch, err :=
		s.repository.SaveValidation(
			ctx,
			batch.ID,
			parseResult,
		)

	if err != nil {
		message :=
			fmt.Sprintf(
				"save validation: %v",
				err,
			)

		_ =
			s.repository.FailBatch(
				ctx,
				batch.ID,
				message,
			)

		return nil,
			fmt.Errorf(
				"batch %s: %w",
				batch.ID,
				err,
			)
	}

	return &StageResult{
		Batch: savedBatch,

		Errors: parseResult.Errors,
	}, nil
}

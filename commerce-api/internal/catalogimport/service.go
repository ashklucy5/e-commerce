package catalogimport

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Stage(
	ctx context.Context,
	sourceFilename string,
	storageKey string,
	checksum string,
	reader io.Reader,
) (*StageResult, error) {
	sourceFilename = strings.TrimSpace(
		sourceFilename,
	)

	storageKey = strings.TrimSpace(
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
		ParseAuto(reader)

	if err != nil {
		message := fmt.Sprintf(
			"parse workbook: %v",
			err,
		)

		_ = s.repository.FailBatch(
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

	savedBatch, err :=
		s.repository.SaveValidation(
			ctx,
			batch.ID,
			parseResult,
		)

	if err != nil {
		message := fmt.Sprintf(
			"save validation: %v",
			err,
		)

		_ = s.repository.FailBatch(
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
		Batch:  savedBatch,
		Errors: parseResult.Errors,
	}, nil
}

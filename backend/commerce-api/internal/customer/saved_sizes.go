package customer

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidSavedSize = errors.New(
		"invalid saved size",
	)

	ErrInvalidSavedSizeID = errors.New(
		"invalid saved size id",
	)

	ErrSavedSizeNotFound = errors.New(
		"saved size not found",
	)

	ErrSavedSizeExists = errors.New(
		"saved size already exists for this category and brand",
	)

	savedSizeUUIDPattern = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	)

	categoryKeyPattern = regexp.MustCompile(
		`^[a-z0-9][a-z0-9_-]*$`,
	)
)

type SavedSize struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`

	CategoryKey string `json:"category_key"`

	Brand *string `json:"brand,omitempty"`

	SizeSystem *string `json:"size_system,omitempty"`

	SizeLabel string `json:"size_label"`

	FitPreference *string `json:"fit_preference,omitempty"`

	Notes *string `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

type SaveSizeRequest struct {
	CategoryKey string `json:"category_key"`

	Brand *string `json:"brand"`

	SizeSystem *string `json:"size_system"`

	SizeLabel string `json:"size_label"`

	FitPreference *string `json:"fit_preference"`

	Notes *string `json:"notes"`
}

func (s *Service) ListSavedSizes(
	ctx context.Context,
	customerID string,
) ([]SavedSize, error) {
	return s.repository.ListSavedSizes(
		ctx,
		customerID,
	)
}

func (s *Service) CreateSavedSize(
	ctx context.Context,
	customerID string,
	request SaveSizeRequest,
) (SavedSize, error) {
	size, err :=
		normalizeSavedSizeRequest(
			customerID,
			"",
			request,
		)
	if err != nil {
		return SavedSize{},
			err
	}

	return s.repository.CreateSavedSize(
		ctx,
		size,
	)
}

func (s *Service) ReplaceSavedSize(
	ctx context.Context,
	customerID string,
	sizeID string,
	request SaveSizeRequest,
) (SavedSize, error) {
	sizeID =
		strings.TrimSpace(
			sizeID,
		)

	if !savedSizeUUIDPattern.MatchString(
		sizeID,
	) {
		return SavedSize{},
			ErrInvalidSavedSizeID
	}

	size, err :=
		normalizeSavedSizeRequest(
			customerID,
			strings.ToLower(
				sizeID,
			),
			request,
		)
	if err != nil {
		return SavedSize{},
			err
	}

	return s.repository.ReplaceSavedSize(
		ctx,
		size,
	)
}

func (s *Service) DeleteSavedSize(
	ctx context.Context,
	customerID string,
	sizeID string,
) error {
	sizeID =
		strings.TrimSpace(
			sizeID,
		)

	if !savedSizeUUIDPattern.MatchString(
		sizeID,
	) {
		return ErrInvalidSavedSizeID
	}

	return s.repository.DeleteSavedSize(
		ctx,
		customerID,
		strings.ToLower(
			sizeID,
		),
	)
}

func (r *Repository) ListSavedSizes(
	ctx context.Context,
	customerID string,
) ([]SavedSize, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					customer_id::text,
					category_key,
					brand,
					size_system,
					size_label,
					fit_preference,
					notes,
					created_at,
					updated_at
				FROM customer_saved_sizes
				WHERE customer_id = $1::uuid
				ORDER BY
					category_key ASC,
					brand ASC NULLS FIRST,
					created_at ASC
			`,
			customerID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer saved sizes: %w",
				err,
			)
	}

	defer rows.Close()

	results :=
		make(
			[]SavedSize,
			0,
		)

	for rows.Next() {
		result, err :=
			scanSavedSize(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan customer saved size: %w",
					err,
				)
		}

		results =
			append(
				results,
				result,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate customer saved sizes: %w",
				err,
			)
	}

	return results, nil
}

func (r *Repository) CreateSavedSize(
	ctx context.Context,
	size SavedSize,
) (SavedSize, error) {
	result, err :=
		scanSavedSize(
			r.db.QueryRow(
				ctx,
				`
					INSERT INTO customer_saved_sizes (
						customer_id,
						category_key,
						brand,
						size_system,
						size_label,
						fit_preference,
						notes
					)
					VALUES (
						$1::uuid,
						$2::varchar,
						$3::varchar,
						$4::varchar,
						$5::varchar,
						$6::varchar,
						$7::varchar
					)
					RETURNING
						id::text,
						customer_id::text,
						category_key,
						brand,
						size_system,
						size_label,
						fit_preference,
						notes,
						created_at,
						updated_at
				`,
				size.CustomerID,
				size.CategoryKey,
				optionalStringValue(
					size.Brand,
				),
				optionalStringValue(
					size.SizeSystem,
				),
				size.SizeLabel,
				optionalStringValue(
					size.FitPreference,
				),
				optionalStringValue(
					size.Notes,
				),
			),
		)
	if err != nil {
		if isSavedSizeUniqueViolation(
			err,
		) {
			return SavedSize{},
				ErrSavedSizeExists
		}

		return SavedSize{},
			fmt.Errorf(
				"create customer saved size: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ReplaceSavedSize(
	ctx context.Context,
	size SavedSize,
) (SavedSize, error) {
	result, err :=
		scanSavedSize(
			r.db.QueryRow(
				ctx,
				`
					UPDATE customer_saved_sizes
					SET
						category_key =
							$3::varchar,
						brand =
							$4::varchar,
						size_system =
							$5::varchar,
						size_label =
							$6::varchar,
						fit_preference =
							$7::varchar,
						notes =
							$8::varchar,
						updated_at =
							now()
					WHERE
						customer_id = $1::uuid
						AND id = $2::uuid
					RETURNING
						id::text,
						customer_id::text,
						category_key,
						brand,
						size_system,
						size_label,
						fit_preference,
						notes,
						created_at,
						updated_at
				`,
				size.CustomerID,
				size.ID,
				size.CategoryKey,
				optionalStringValue(
					size.Brand,
				),
				optionalStringValue(
					size.SizeSystem,
				),
				size.SizeLabel,
				optionalStringValue(
					size.FitPreference,
				),
				optionalStringValue(
					size.Notes,
				),
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return SavedSize{},
				ErrSavedSizeNotFound
		}

		if isSavedSizeUniqueViolation(
			err,
		) {
			return SavedSize{},
				ErrSavedSizeExists
		}

		return SavedSize{},
			fmt.Errorf(
				"replace customer saved size: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) DeleteSavedSize(
	ctx context.Context,
	customerID string,
	sizeID string,
) error {
	commandTag, err :=
		r.db.Exec(
			ctx,
			`
				DELETE FROM customer_saved_sizes
				WHERE
					customer_id = $1::uuid
					AND id = $2::uuid
			`,
			customerID,
			sizeID,
		)
	if err != nil {
		return fmt.Errorf(
			"delete customer saved size: %w",
			err,
		)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrSavedSizeNotFound
	}

	return nil
}

type savedSizeScanner interface {
	Scan(dest ...any) error
}

func scanSavedSize(
	row savedSizeScanner,
) (SavedSize, error) {
	var result SavedSize

	var brand pgtype.Text

	var sizeSystem pgtype.Text

	var fitPreference pgtype.Text

	var notes pgtype.Text

	if err :=
		row.Scan(
			&result.ID,
			&result.CustomerID,
			&result.CategoryKey,
			&brand,
			&sizeSystem,
			&result.SizeLabel,
			&fitPreference,
			&notes,
			&result.CreatedAt,
			&result.UpdatedAt,
		); err != nil {
		return SavedSize{},
			err
	}

	if brand.Valid {
		value :=
			brand.String

		result.Brand =
			&value
	}

	if sizeSystem.Valid {
		value :=
			sizeSystem.String

		result.SizeSystem =
			&value
	}

	if fitPreference.Valid {
		value :=
			fitPreference.String

		result.FitPreference =
			&value
	}

	if notes.Valid {
		value :=
			notes.String

		result.Notes =
			&value
	}

	return result, nil
}

func normalizeSavedSizeRequest(
	customerID string,
	sizeID string,
	request SaveSizeRequest,
) (SavedSize, error) {
	categoryKey :=
		strings.ToLower(
			strings.TrimSpace(
				request.CategoryKey,
			),
		)

	if categoryKey == "" ||
		len(
			categoryKey,
		) > 80 ||
		!categoryKeyPattern.MatchString(
			categoryKey,
		) {
		return SavedSize{},
			ErrInvalidSavedSize
	}

	sizeLabel :=
		strings.Join(
			strings.Fields(
				request.SizeLabel,
			),
			" ",
		)

	if sizeLabel == "" ||
		len(
			sizeLabel,
		) > 40 {
		return SavedSize{},
			ErrInvalidSavedSize
	}

	brand, err :=
		normalizeOptionalSavedSizeText(
			request.Brand,
			160,
			true,
		)
	if err != nil {
		return SavedSize{},
			err
	}

	sizeSystem, err :=
		normalizeOptionalSavedSizeText(
			request.SizeSystem,
			20,
			false,
		)
	if err != nil {
		return SavedSize{},
			err
	}

	if sizeSystem != nil {
		value :=
			strings.ToUpper(
				*sizeSystem,
			)

		sizeSystem =
			&value
	}

	fitPreference, err :=
		normalizeOptionalSavedSizeText(
			request.FitPreference,
			30,
			true,
		)
	if err != nil {
		return SavedSize{},
			err
	}

	notes, err :=
		normalizeOptionalSavedSizeText(
			request.Notes,
			255,
			false,
		)
	if err != nil {
		return SavedSize{},
			err
	}

	return SavedSize{
			ID: sizeID,

			CustomerID: customerID,

			CategoryKey: categoryKey,

			Brand: brand,

			SizeSystem: sizeSystem,

			SizeLabel: sizeLabel,

			FitPreference: fitPreference,

			Notes: notes,
		},
		nil
}

func normalizeOptionalSavedSizeText(
	value *string,
	maxLength int,
	lowercase bool,
) (*string, error) {
	if value == nil {
		return nil, nil
	}

	normalized :=
		strings.Join(
			strings.Fields(
				*value,
			),
			" ",
		)

	if normalized == "" {
		return nil, nil
	}

	if lowercase {
		normalized =
			strings.ToLower(
				normalized,
			)
	}

	if len(
		normalized,
	) > maxLength {
		return nil,
			ErrInvalidSavedSize
	}

	return &normalized,
		nil
}

func isSavedSizeUniqueViolation(
	err error,
) bool {
	var pgErr *pgconn.PgError

	if !errors.As(
		err,
		&pgErr,
	) ||
		pgErr.Code != "23505" {
		return false
	}

	switch pgErr.ConstraintName {
	case "customer_saved_sizes_generic_key",
		"customer_saved_sizes_brand_key":
		return true

	default:
		return false
	}
}

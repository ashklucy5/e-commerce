package customer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidStyleProfile = errors.New(
	"invalid customer style profile",
)

type StyleProfile struct {
	CustomerID          string   `json:"customer_id"`
	PreferredColors     []string `json:"preferred_colors"`
	PreferredStyles     []string `json:"preferred_styles"`
	PreferredCategories []string `json:"preferred_categories"`
	FitPreference       *string  `json:"fit_preference,omitempty"`
	BudgetMinAmount     *int64   `json:"budget_min_amount,omitempty"`
	BudgetMaxAmount     *int64   `json:"budget_max_amount,omitempty"`
	Currency            string   `json:"currency"`
	Notes               *string  `json:"notes,omitempty"`
}

type PutStyleProfileRequest struct {
	PreferredColors     []string `json:"preferred_colors"`
	PreferredStyles     []string `json:"preferred_styles"`
	PreferredCategories []string `json:"preferred_categories"`
	FitPreference       *string  `json:"fit_preference"`
	BudgetMinAmount     *int64   `json:"budget_min_amount"`
	BudgetMaxAmount     *int64   `json:"budget_max_amount"`
	Currency            string   `json:"currency"`
	Notes               *string  `json:"notes"`
}

func (s *Service) GetStyleProfile(
	ctx context.Context,
	customerID string,
) (StyleProfile, error) {
	return s.repository.GetStyleProfile(
		ctx,
		customerID,
	)
}

func (s *Service) PutStyleProfile(
	ctx context.Context,
	customerID string,
	request PutStyleProfileRequest,
) (StyleProfile, error) {
	colors, err :=
		normalizePreferenceList(
			request.PreferredColors,
		)
	if err != nil {
		return StyleProfile{},
			err
	}

	styles, err :=
		normalizePreferenceList(
			request.PreferredStyles,
		)
	if err != nil {
		return StyleProfile{},
			err
	}

	categories, err :=
		normalizePreferenceList(
			request.PreferredCategories,
		)
	if err != nil {
		return StyleProfile{},
			err
	}

	fitPreference, err :=
		normalizeOptionalStyleText(
			request.FitPreference,
			30,
			true,
		)
	if err != nil {
		return StyleProfile{},
			err
	}

	notes, err :=
		normalizeOptionalStyleText(
			request.Notes,
			500,
			false,
		)
	if err != nil {
		return StyleProfile{},
			err
	}

	if request.BudgetMinAmount != nil &&
		*request.BudgetMinAmount < 0 {
		return StyleProfile{},
			ErrInvalidStyleProfile
	}

	if request.BudgetMaxAmount != nil &&
		*request.BudgetMaxAmount < 0 {
		return StyleProfile{},
			ErrInvalidStyleProfile
	}

	if request.BudgetMinAmount != nil &&
		request.BudgetMaxAmount != nil &&
		*request.BudgetMaxAmount <
			*request.BudgetMinAmount {
		return StyleProfile{},
			ErrInvalidStyleProfile
	}

	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				request.Currency,
			),
		)

	if currency == "" {
		currency =
			"BDT"
	}

	if currency != "BDT" {
		return StyleProfile{},
			ErrInvalidStyleProfile
	}

	profile :=
		StyleProfile{
			CustomerID: customerID,

			PreferredColors: colors,

			PreferredStyles: styles,

			PreferredCategories: categories,

			FitPreference: fitPreference,

			BudgetMinAmount: request.BudgetMinAmount,

			BudgetMaxAmount: request.BudgetMaxAmount,

			Currency: currency,

			Notes: notes,
		}

	return s.repository.PutStyleProfile(
		ctx,
		profile,
	)
}

func (r *Repository) GetStyleProfile(
	ctx context.Context,
	customerID string,
) (StyleProfile, error) {
	_, err :=
		r.db.Exec(
			ctx,
			`
				INSERT INTO customer_style_profiles (
					customer_id
				)
				VALUES (
					$1::uuid
				)
				ON CONFLICT (customer_id)
				DO NOTHING
			`,
			customerID,
		)
	if err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"ensure customer style profile: %w",
				err,
			)
	}

	return r.selectStyleProfile(
		ctx,
		customerID,
	)
}

func (r *Repository) PutStyleProfile(
	ctx context.Context,
	profile StyleProfile,
) (StyleProfile, error) {
	colors, err :=
		json.Marshal(
			profile.PreferredColors,
		)
	if err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"encode preferred colors: %w",
				err,
			)
	}

	styles, err :=
		json.Marshal(
			profile.PreferredStyles,
		)
	if err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"encode preferred styles: %w",
				err,
			)
	}

	categories, err :=
		json.Marshal(
			profile.PreferredCategories,
		)
	if err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"encode preferred categories: %w",
				err,
			)
	}

	row :=
		r.db.QueryRow(
			ctx,
			`
				INSERT INTO customer_style_profiles (
					customer_id,
					preferred_colors,
					preferred_styles,
					preferred_categories,
					fit_preference,
					budget_min_amount,
					budget_max_amount,
					currency,
					notes
				)
				VALUES (
					$1::uuid,
					$2::jsonb,
					$3::jsonb,
					$4::jsonb,
					$5::varchar,
					$6::bigint,
					$7::bigint,
					$8::varchar,
					$9::varchar
				)
				ON CONFLICT (customer_id)
				DO UPDATE SET
					preferred_colors =
						EXCLUDED.preferred_colors,
					preferred_styles =
						EXCLUDED.preferred_styles,
					preferred_categories =
						EXCLUDED.preferred_categories,
					fit_preference =
						EXCLUDED.fit_preference,
					budget_min_amount =
						EXCLUDED.budget_min_amount,
					budget_max_amount =
						EXCLUDED.budget_max_amount,
					currency =
						EXCLUDED.currency,
					notes =
						EXCLUDED.notes,
					updated_at =
						now()
				RETURNING
					customer_id::text,
					preferred_colors::text,
					preferred_styles::text,
					preferred_categories::text,
					fit_preference,
					budget_min_amount,
					budget_max_amount,
					currency,
					notes
			`,
			profile.CustomerID,
			string(
				colors,
			),
			string(
				styles,
			),
			string(
				categories,
			),
			optionalStringValue(
				profile.FitPreference,
			),
			optionalInt64Value(
				profile.BudgetMinAmount,
			),
			optionalInt64Value(
				profile.BudgetMaxAmount,
			),
			profile.Currency,
			optionalStringValue(
				profile.Notes,
			),
		)

	result, err :=
		scanStyleProfile(
			row,
		)
	if err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"put customer style profile: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) selectStyleProfile(
	ctx context.Context,
	customerID string,
) (StyleProfile, error) {
	result, err :=
		scanStyleProfile(
			r.db.QueryRow(
				ctx,
				`
					SELECT
						customer_id::text,
						preferred_colors::text,
						preferred_styles::text,
						preferred_categories::text,
						fit_preference,
						budget_min_amount,
						budget_max_amount,
						currency,
						notes
					FROM customer_style_profiles
					WHERE customer_id = $1::uuid
				`,
				customerID,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return StyleProfile{},
				ErrCustomerNotFound
		}

		return StyleProfile{},
			fmt.Errorf(
				"get customer style profile: %w",
				err,
			)
	}

	return result, nil
}

type styleProfileScanner interface {
	Scan(dest ...any) error
}

func scanStyleProfile(
	row styleProfileScanner,
) (StyleProfile, error) {
	var result StyleProfile

	var colorsJSON string

	var stylesJSON string

	var categoriesJSON string

	var fitPreference pgtype.Text

	var budgetMin pgtype.Int8

	var budgetMax pgtype.Int8

	var notes pgtype.Text

	if err :=
		row.Scan(
			&result.CustomerID,
			&colorsJSON,
			&stylesJSON,
			&categoriesJSON,
			&fitPreference,
			&budgetMin,
			&budgetMax,
			&result.Currency,
			&notes,
		); err != nil {
		return StyleProfile{},
			err
	}

	if err :=
		json.Unmarshal(
			[]byte(
				colorsJSON,
			),
			&result.PreferredColors,
		); err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"decode preferred colors: %w",
				err,
			)
	}

	if err :=
		json.Unmarshal(
			[]byte(
				stylesJSON,
			),
			&result.PreferredStyles,
		); err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"decode preferred styles: %w",
				err,
			)
	}

	if err :=
		json.Unmarshal(
			[]byte(
				categoriesJSON,
			),
			&result.PreferredCategories,
		); err != nil {
		return StyleProfile{},
			fmt.Errorf(
				"decode preferred categories: %w",
				err,
			)
	}

	if fitPreference.Valid {
		value :=
			fitPreference.String

		result.FitPreference =
			&value
	}

	if budgetMin.Valid {
		value :=
			budgetMin.Int64

		result.BudgetMinAmount =
			&value
	}

	if budgetMax.Valid {
		value :=
			budgetMax.Int64

		result.BudgetMaxAmount =
			&value
	}

	if notes.Valid {
		value :=
			notes.String

		result.Notes =
			&value
	}

	if result.PreferredColors == nil {
		result.PreferredColors =
			[]string{}
	}

	if result.PreferredStyles == nil {
		result.PreferredStyles =
			[]string{}
	}

	if result.PreferredCategories == nil {
		result.PreferredCategories =
			[]string{}
	}

	return result, nil
}

func normalizePreferenceList(
	values []string,
) ([]string, error) {
	if len(
		values,
	) > 30 {
		return nil,
			ErrInvalidStyleProfile
	}

	result :=
		make(
			[]string,
			0,
			len(
				values,
			),
		)

	seen :=
		make(
			map[string]struct{},
			len(
				values,
			),
		)

	for _, raw := range values {
		value :=
			strings.ToLower(
				strings.Join(
					strings.Fields(
						raw,
					),
					" ",
				),
			)

		if value == "" ||
			len(
				value,
			) > 80 {
			return nil,
				ErrInvalidStyleProfile
		}

		if _, exists :=
			seen[value]; exists {
			continue
		}

		seen[value] =
			struct{}{}

		result =
			append(
				result,
				value,
			)
	}

	return result, nil
}

func normalizeOptionalStyleText(
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
			ErrInvalidStyleProfile
	}

	return &normalized,
		nil
}

func optionalStringValue(
	value *string,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func optionalInt64Value(
	value *int64,
) any {
	if value == nil {
		return nil
	}

	return *value
}

package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

const (
	FinanceVariantCostSourceManual = "manual"
	FinanceVariantCostSourceImport = "import"

	adminEventFinanceVariantCostRecorded = "finance_variant_cost_recorded"
)

var (
	ErrFinanceVariantCostNotFound         = errors.New("finance variant cost history record not found")
	ErrFinanceVariantNotFound             = errors.New("finance product variant not found")
	ErrInvalidFinanceVariantCost          = errors.New("invalid finance variant cost")
	ErrFinanceVariantCostCurrencyMismatch = errors.New(
		"finance variant cost currency does not match the variant currency",
	)
)

type FinanceVariantCost struct {
	ID          string `json:"id"`
	VariantID   string `json:"variant_id"`
	ProductID   string `json:"product_id"`
	ProductCode string `json:"product_code"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`

	UnitCostAmount         int64     `json:"unit_cost_amount"`
	PreviousUnitCostAmount *int64    `json:"previous_unit_cost_amount,omitempty"`
	Currency               string    `json:"currency"`
	EffectiveAt            time.Time `json:"effective_at"`
	Source                 string    `json:"source"`
	Description            string    `json:"description,omitempty"`
	Reference              string    `json:"reference,omitempty"`
	CreatedByStaffID       string    `json:"created_by_staff_id"`
	CreatedAt              time.Time `json:"created_at"`

	CurrentUnitCostAmount *int64 `json:"current_unit_cost_amount,omitempty"`
	VariantCurrency       string `json:"variant_currency"`
}

type FinanceVariantCostPage struct {
	Items []FinanceVariantCost    `json:"items"`
	Meta  platformpagination.Meta `json:"meta"`
}

type FinanceVariantCostFilter struct {
	ProductID string
	VariantID string
	SKU       string
	Source    string
	Currency  string
	FromSet   bool
	From      time.Time
	ToSet     bool
	To        time.Time
}

type FinanceVariantCostInput struct {
	VariantID      string
	SKU            string
	UnitCostAmount int64
	Currency       string
	EffectiveAt    time.Time
	Source         string
	Description    string
	Reference      string
	SetCurrent     bool
}

type FinanceVariantCostCreateResult struct {
	Cost               FinanceVariantCost `json:"cost"`
	CurrentCostUpdated bool               `json:"current_cost_updated"`
}

type financeVariantSnapshot struct {
	VariantID         string
	ProductID         string
	ProductCode       string
	ProductName       string
	SKU               string
	CurrentCostAmount pgtype.Int8
	Currency          string
}

func (s *Service) ListFinanceVariantCosts(
	ctx context.Context,
	params platformpagination.Params,
	filter FinanceVariantCostFilter,
) (FinanceVariantCostPage, error) {
	filter = normalizeFinanceVariantCostFilter(filter)

	if err := validateFinanceVariantCostFilter(filter); err != nil {
		return FinanceVariantCostPage{}, err
	}

	const whereSQL = `
		FROM finance_variant_cost_history h
		JOIN product_variants v ON v.id = h.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE
			($1 = '' OR v.product_id = NULLIF($1, '')::uuid)
			AND ($2 = '' OR h.variant_id = NULLIF($2, '')::uuid)
			AND ($3 = '' OR v.sku = $3)
			AND ($4 = '' OR h.source = $4)
			AND ($5 = '' OR h.currency = $5)
			AND (NOT $6::boolean OR h.effective_at >= $7::timestamptz)
			AND (NOT $8::boolean OR h.effective_at < $9::timestamptz)
	`

	args := []any{
		filter.ProductID,
		filter.VariantID,
		filter.SKU,
		filter.Source,
		filter.Currency,
		filter.FromSet,
		filter.From,
		filter.ToSet,
		filter.To,
	}

	var total int64

	if err := s.db.QueryRow(
		ctx,
		"SELECT COUNT(*)::bigint "+whereSQL,
		args...,
	).Scan(&total); err != nil {
		return FinanceVariantCostPage{},
			fmt.Errorf(
				"count finance variant cost history: %w",
				err,
			)
	}

	rows, err := s.db.Query(
		ctx,
		`SELECT `+financeVariantCostColumnsSQL+whereSQL+`
		ORDER BY
			h.effective_at DESC,
			h.created_at DESC,
			h.id DESC
		LIMIT $10
		OFFSET $11`,
		append(
			args,
			params.Limit,
			params.Offset(),
		)...,
	)
	if err != nil {
		return FinanceVariantCostPage{},
			fmt.Errorf(
				"list finance variant cost history: %w",
				err,
			)
	}

	defer rows.Close()

	items := make(
		[]FinanceVariantCost,
		0,
		params.Limit,
	)

	for rows.Next() {
		item, err := scanFinanceVariantCost(rows)
		if err != nil {
			return FinanceVariantCostPage{},
				fmt.Errorf(
					"scan finance variant cost history: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return FinanceVariantCostPage{},
			fmt.Errorf(
				"iterate finance variant cost history: %w",
				err,
			)
	}

	return FinanceVariantCostPage{
		Items: items,
		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetFinanceVariantCost(
	ctx context.Context,
	costID string,
) (
	FinanceVariantCost,
	error,
) {
	item, err := scanFinanceVariantCost(
		s.db.QueryRow(
			ctx,
			"SELECT "+financeVariantCostColumnsSQL+`
			FROM finance_variant_cost_history h
			JOIN product_variants v
				ON v.id = h.variant_id
			JOIN products p
				ON p.id = v.product_id
			WHERE h.id = $1::uuid
			`,
			costID,
		),
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return FinanceVariantCost{},
			ErrFinanceVariantCostNotFound
	}

	if err != nil {
		return FinanceVariantCost{},
			fmt.Errorf(
				"get finance variant cost history: %w",
				err,
			)
	}

	return item,
		nil
}

func (s *Service) CreateFinanceVariantCost(
	ctx context.Context,
	input FinanceVariantCostInput,
	metadata AdminActionMetadata,
) (
	FinanceVariantCostCreateResult,
	error,
) {
	normalized, err :=
		normalizeFinanceVariantCostInput(
			input,
		)
	if err != nil {
		return FinanceVariantCostCreateResult{},
			err
	}

	var costID string
	var currentUpdated bool

	err =
		platformdatabase.WithinTx(
			ctx,
			s.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var txErr error

				costID,
					currentUpdated,
					txErr =
					createFinanceVariantCostTx(
						ctx,
						tx,
						normalized,
						metadata,
						true,
					)

				return txErr
			},
		)
	if err != nil {
		return FinanceVariantCostCreateResult{},
			err
	}

	cost, err :=
		s.GetFinanceVariantCost(
			ctx,
			costID,
		)
	if err != nil {
		return FinanceVariantCostCreateResult{},
			err
	}

	return FinanceVariantCostCreateResult{
		Cost: cost,

		CurrentCostUpdated: currentUpdated,
	}, nil
}

// createFinanceVariantCostTx is intentionally reusable by the Excel importer.
// That lets a complete workbook be applied inside one database transaction.
func createFinanceVariantCostTx(
	ctx context.Context,
	tx pgx.Tx,
	input FinanceVariantCostInput,
	metadata AdminActionMetadata,
	writeAudit bool,
) (
	string,
	bool,
	error,
) {
	normalized, err :=
		normalizeFinanceVariantCostInput(
			input,
		)
	if err != nil {
		return "",
			false,
			err
	}

	input = normalized

	variant, err :=
		lockFinanceVariantForCostTx(
			ctx,
			tx,
			input.VariantID,
			input.SKU,
		)
	if err != nil {
		return "",
			false,
			err
	}

	if input.VariantID != "" &&
		input.VariantID != variant.VariantID {

		return "",
			false,
			ErrFinanceVariantNotFound
	}

	if input.SKU != "" &&
		input.SKU != variant.SKU {

		return "",
			false,
			fmt.Errorf(
				"%w: variant_id and sku identify different variants",
				ErrInvalidFinanceVariantCost,
			)
	}

	if input.Currency == "" {
		input.Currency =
			variant.Currency
	}

	if input.SetCurrent &&
		input.Currency != variant.Currency {

		return "",
			false,
			fmt.Errorf(
				"%w: current catalog cost must use %s",
				ErrFinanceVariantCostCurrencyMismatch,
				variant.Currency,
			)
	}

	if input.SetCurrent &&
		input.EffectiveAt.After(
			time.Now().UTC(),
		) {

		return "",
			false,
			fmt.Errorf(
				"%w: a future effective_at cannot be applied as the current catalog cost",
				ErrInvalidFinanceVariantCost,
			)
	}

	var previousCost any

	if variant.CurrentCostAmount.Valid {
		previousCost =
			variant.CurrentCostAmount.Int64
	}

	var costID string

	if err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO finance_variant_cost_history (
					variant_id,
					unit_cost_amount,
					previous_unit_cost_amount,
					currency,
					effective_at,
					source,
					description,
					reference,
					created_by_staff_id,
					created_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					$4,
					$5,
					$6,
					NULLIF($7, ''),
					NULLIF($8, ''),
					$9::uuid,
					now()
				)
				RETURNING id::text
			`,
			variant.VariantID,
			input.UnitCostAmount,
			previousCost,
			input.Currency,
			input.EffectiveAt,
			input.Source,
			input.Description,
			input.Reference,
			metadata.StaffAccountID,
		).Scan(
			&costID,
		); err != nil {

		return "",
			false,
			fmt.Errorf(
				"insert finance variant cost history: %w",
				err,
			)
	}

	currentUpdated :=
		false

	if input.SetCurrent {
		if _, err :=
			tx.Exec(
				ctx,
				`
					UPDATE product_variants
					SET
						cost_amount = $2,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				variant.VariantID,
				input.UnitCostAmount,
			); err != nil {

			return "",
				false,
				fmt.Errorf(
					"update current product variant cost: %w",
					err,
				)
		}

		currentUpdated =
			true
	}

	if writeAudit {
		if err :=
			insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventFinanceVariantCostRecorded,
				map[string]any{
					"finance_variant_cost_id": costID,

					"variant_id": variant.VariantID,

					"product_id": variant.ProductID,

					"product_code": variant.ProductCode,

					"sku": variant.SKU,

					"unit_cost_amount": input.UnitCostAmount,

					"previous_unit_cost_amount": previousCost,

					"currency": input.Currency,

					"effective_at": input.EffectiveAt,

					"source": input.Source,

					"set_current": input.SetCurrent,
				},
			); err != nil {

			return "",
				false,
				err
		}
	}

	return costID,
		currentUpdated,
		nil
}

func lockFinanceVariantForCostTx(
	ctx context.Context,
	tx pgx.Tx,
	variantID string,
	sku string,
) (
	financeVariantSnapshot,
	error,
) {
	var result financeVariantSnapshot

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					v.id::text,
					v.product_id::text,
					p.product_code,
					p.name,
					v.sku,
					v.cost_amount,
					v.currency
				FROM product_variants v
				JOIN products p
					ON p.id = v.product_id
				WHERE
					(
						$1 <> ''
						AND v.id = NULLIF($1, '')::uuid
					)
					OR (
						$1 = ''
						AND v.sku = $2
					)
				LIMIT 1
				FOR UPDATE OF v
			`,
			variantID,
			sku,
		).Scan(
			&result.VariantID,
			&result.ProductID,
			&result.ProductCode,
			&result.ProductName,
			&result.SKU,
			&result.CurrentCostAmount,
			&result.Currency,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return financeVariantSnapshot{},
			ErrFinanceVariantNotFound
	}

	if err != nil {
		return financeVariantSnapshot{},
			fmt.Errorf(
				"lock product variant for finance cost: %w",
				err,
			)
	}

	return result,
		nil
}

const financeVariantCostColumnsSQL = `
	h.id::text,
	h.variant_id::text,
	v.product_id::text,
	p.product_code,
	p.name,
	v.sku,
	h.unit_cost_amount,
	h.previous_unit_cost_amount,
	h.currency,
	h.effective_at,
	h.source,
	COALESCE(h.description, ''),
	COALESCE(h.reference, ''),
	h.created_by_staff_id::text,
	h.created_at,
	v.cost_amount,
	v.currency
`

type financeVariantCostScanner interface {
	Scan(dest ...any) error
}

func scanFinanceVariantCost(
	scanner financeVariantCostScanner,
) (
	FinanceVariantCost,
	error,
) {
	var item FinanceVariantCost
	var previousCost pgtype.Int8
	var currentCost pgtype.Int8

	err :=
		scanner.Scan(
			&item.ID,
			&item.VariantID,
			&item.ProductID,
			&item.ProductCode,
			&item.ProductName,
			&item.SKU,
			&item.UnitCostAmount,
			&previousCost,
			&item.Currency,
			&item.EffectiveAt,
			&item.Source,
			&item.Description,
			&item.Reference,
			&item.CreatedByStaffID,
			&item.CreatedAt,
			&currentCost,
			&item.VariantCurrency,
		)

	if err != nil {
		return FinanceVariantCost{},
			err
	}

	item.PreviousUnitCostAmount =
		financeInt64Pointer(
			previousCost,
		)

	item.CurrentUnitCostAmount =
		financeInt64Pointer(
			currentCost,
		)

	return item,
		nil
}

func financeInt64Pointer(
	value pgtype.Int8,
) *int64 {
	if !value.Valid {
		return nil
	}

	result :=
		value.Int64

	return &result
}

func normalizeFinanceVariantCostFilter(
	filter FinanceVariantCostFilter,
) FinanceVariantCostFilter {
	filter.ProductID =
		strings.TrimSpace(
			filter.ProductID,
		)

	filter.VariantID =
		strings.TrimSpace(
			filter.VariantID,
		)

	filter.SKU =
		strings.TrimSpace(
			filter.SKU,
		)

	filter.Source =
		strings.ToLower(
			strings.TrimSpace(
				filter.Source,
			),
		)

	filter.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				filter.Currency,
			),
		)

	return filter
}

func validateFinanceVariantCostFilter(
	filter FinanceVariantCostFilter,
) error {
	if filter.ProductID != "" &&
		!platformvalidation.IsUUID(
			filter.ProductID,
		) {

		return fmt.Errorf(
			"%w: product_id filter must be a valid UUID",
			ErrInvalidFinanceVariantCost,
		)
	}

	if filter.VariantID != "" &&
		!platformvalidation.IsUUID(
			filter.VariantID,
		) {

		return fmt.Errorf(
			"%w: variant_id filter must be a valid UUID",
			ErrInvalidFinanceVariantCost,
		)
	}

	if utf8.RuneCountInString(
		filter.SKU,
	) > 100 {

		return fmt.Errorf(
			"%w: sku filter must be at most 100 characters",
			ErrInvalidFinanceVariantCost,
		)
	}

	if !validFinanceVariantCostSource(
		filter.Source,
		true,
	) {
		return fmt.Errorf(
			"%w: invalid source filter",
			ErrInvalidFinanceVariantCost,
		)
	}

	if filter.Currency != "" &&
		!validFinanceCurrency(
			filter.Currency,
		) {

		return fmt.Errorf(
			"%w: currency filter must be a 3-letter uppercase code",
			ErrInvalidFinanceVariantCost,
		)
	}

	if filter.FromSet &&
		filter.ToSet &&
		!filter.To.After(
			filter.From,
		) {

		return fmt.Errorf(
			"%w: to must be after from",
			ErrInvalidFinanceVariantCost,
		)
	}

	return nil
}

func normalizeFinanceVariantCostInput(
	input FinanceVariantCostInput,
) (
	FinanceVariantCostInput,
	error,
) {
	input.VariantID =
		strings.TrimSpace(
			input.VariantID,
		)

	input.SKU =
		strings.TrimSpace(
			input.SKU,
		)

	if input.VariantID == "" &&
		input.SKU == "" {

		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: variant_id or sku is required",
				ErrInvalidFinanceVariantCost,
			)
	}

	if input.VariantID != "" &&
		!platformvalidation.IsUUID(
			input.VariantID,
		) {

		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: variant_id must be a valid UUID",
				ErrInvalidFinanceVariantCost,
			)
	}

	if utf8.RuneCountInString(
		input.SKU,
	) > 100 {

		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: sku must be at most 100 characters",
				ErrInvalidFinanceVariantCost,
			)
	}

	if input.UnitCostAmount < 0 {
		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: unit_cost_amount cannot be negative",
				ErrInvalidFinanceVariantCost,
			)
	}

	input.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				input.Currency,
			),
		)

	if input.Currency != "" &&
		!validFinanceCurrency(
			input.Currency,
		) {

		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: currency must be a 3-letter uppercase code",
				ErrInvalidFinanceVariantCost,
			)
	}

	if input.EffectiveAt.IsZero() {
		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: effective_at is required",
				ErrInvalidFinanceVariantCost,
			)
	}

	input.EffectiveAt =
		input.EffectiveAt.UTC()

	input.Source =
		strings.ToLower(
			strings.TrimSpace(
				input.Source,
			),
		)

	if input.Source == "" {
		input.Source =
			FinanceVariantCostSourceManual
	}

	if !validFinanceVariantCostSource(
		input.Source,
		false,
	) {
		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: source must be manual or import",
				ErrInvalidFinanceVariantCost,
			)
	}

	input.Description =
		strings.TrimSpace(
			input.Description,
		)

	if utf8.RuneCountInString(
		input.Description,
	) > 500 {

		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: description must be at most 500 characters",
				ErrInvalidFinanceVariantCost,
			)
	}

	input.Reference =
		strings.TrimSpace(
			input.Reference,
		)

	if utf8.RuneCountInString(
		input.Reference,
	) > 160 {

		return FinanceVariantCostInput{},
			fmt.Errorf(
				"%w: reference must be at most 160 characters",
				ErrInvalidFinanceVariantCost,
			)
	}

	return input,
		nil
}

func validFinanceVariantCostSource(
	value string,
	allowEmpty bool,
) bool {
	if value == "" {
		return allowEmpty
	}

	return value == FinanceVariantCostSourceManual ||
		value == FinanceVariantCostSourceImport
}

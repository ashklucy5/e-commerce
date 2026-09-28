package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

const (
	FinanceExpenseCategoryDelivery            = "delivery"
	FinanceExpenseCategoryPaymentFee          = "payment_fee"
	FinanceExpenseCategoryChinaFreight        = "china_freight"
	FinanceExpenseCategoryCustoms             = "customs"
	FinanceExpenseCategoryPackaging           = "packaging"
	FinanceExpenseCategoryMarketing           = "marketing"
	FinanceExpenseCategoryWarehouse           = "warehouse"
	FinanceExpenseCategorySalary              = "salary"
	FinanceExpenseCategoryStaffBenefit        = "staff_benefit"
	FinanceExpenseCategoryRent                = "rent"
	FinanceExpenseCategoryUtilities           = "utilities"
	FinanceExpenseCategorySoftware            = "software"
	FinanceExpenseCategoryProfessionalService = "professional_service"
	FinanceExpenseCategoryBankFee             = "bank_fee"
	FinanceExpenseCategoryInsurance           = "insurance"
	FinanceExpenseCategoryTaxFee              = "tax_fee"
	FinanceExpenseCategoryOffice              = "office"
	FinanceExpenseCategoryTravel              = "travel"
	FinanceExpenseCategoryOther               = "other"

	FinanceExpenseScopeBusiness  = "business"
	FinanceExpenseScopeOrder     = "order"
	FinanceExpenseScopeProduct   = "product"
	FinanceExpenseScopeVariant   = "variant"
	FinanceExpenseScopeStaff     = "staff"
	FinanceExpenseScopeWarehouse = "warehouse"

	FinanceExpenseStatusActive = "active"
	FinanceExpenseStatusVoided = "voided"

	adminEventFinanceExpenseCreated = "finance_expense_created"
	adminEventFinanceExpenseVoided  = "finance_expense_voided"
)

var (
	ErrFinanceExpenseNotFound = errors.New(
		"finance expense not found",
	)

	ErrInvalidFinanceExpense = errors.New(
		"invalid finance expense",
	)

	ErrFinanceExpenseOrderNotFound = errors.New(
		"finance expense order not found",
	)

	ErrFinanceExpenseProductNotFound = errors.New(
		"finance expense product not found",
	)

	ErrFinanceExpenseVariantNotFound = errors.New(
		"finance expense variant not found",
	)

	ErrFinanceExpenseStaffNotFound = errors.New(
		"finance expense staff account not found",
	)

	ErrFinanceExpenseWarehouseNotFound = errors.New(
		"finance expense warehouse not found",
	)

	ErrFinanceExpenseIdempotencyConflict = errors.New(
		"finance expense idempotency key reused with different request",
	)
)

type FinanceExpense struct {
	ID string `json:"id"`

	Category string `json:"category"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	OccurredAt time.Time `json:"occurred_at"`

	Description string `json:"description"`

	Scope string `json:"scope"`

	OrderID        string `json:"order_id,omitempty"`
	ProductID      string `json:"product_id,omitempty"`
	VariantID      string `json:"variant_id,omitempty"`
	StaffAccountID string `json:"staff_account_id,omitempty"`
	WarehouseID    string `json:"warehouse_id,omitempty"`

	Reference string `json:"reference,omitempty"`

	Status string `json:"status"`

	CreatedByStaffID string `json:"created_by_staff_id"`

	VoidedByStaffID string     `json:"voided_by_staff_id,omitempty"`
	VoidReason      string     `json:"void_reason,omitempty"`
	VoidedAt        *time.Time `json:"voided_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FinanceExpensePage struct {
	Items []FinanceExpense `json:"items"`

	Meta platformpagination.Meta `json:"meta"`
}

type FinanceExpenseFilter struct {
	Category string
	Status   string
	Currency string
	Scope    string

	OrderID        string
	ProductID      string
	VariantID      string
	StaffAccountID string
	WarehouseID    string

	FromSet bool
	From    time.Time

	ToSet bool
	To    time.Time
}

type FinanceExpenseInput struct {
	Category string
	Amount   int64
	Currency string

	OccurredAt time.Time

	Description string

	OrderID        string
	ProductID      string
	VariantID      string
	StaffAccountID string
	WarehouseID    string

	Reference string
}

type FinanceExpenseCreateResult struct {
	Expense FinanceExpense `json:"expense"`

	Inserted bool `json:"inserted"`
}

func (s *Service) ListFinanceExpenses(
	ctx context.Context,
	params platformpagination.Params,
	filter FinanceExpenseFilter,
) (
	FinanceExpensePage,
	error,
) {
	filter =
		normalizeFinanceExpenseFilter(
			filter,
		)

	if err :=
		validateFinanceExpenseFilter(
			filter,
		); err != nil {

		return FinanceExpensePage{},
			err
	}

	var total int64

	if err :=
		s.db.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::bigint
				FROM finance_expenses
				WHERE
					($1 = '' OR category = $1)
					AND ($2 = '' OR status = $2)
					AND ($3 = '' OR currency = $3)
					AND (
						$4 = ''
						OR (
							$4 = 'business'
							AND order_id IS NULL
							AND product_id IS NULL
							AND variant_id IS NULL
							AND staff_account_id IS NULL
							AND warehouse_id IS NULL
						)
						OR ($4 = 'order' AND order_id IS NOT NULL)
						OR ($4 = 'product' AND product_id IS NOT NULL)
						OR ($4 = 'variant' AND variant_id IS NOT NULL)
						OR ($4 = 'staff' AND staff_account_id IS NOT NULL)
						OR ($4 = 'warehouse' AND warehouse_id IS NOT NULL)
					)
					AND (
						$5 = ''
						OR order_id = NULLIF($5, '')::uuid
					)
					AND (
						$6 = ''
						OR product_id = NULLIF($6, '')::uuid
					)
					AND (
						$7 = ''
						OR variant_id = NULLIF($7, '')::uuid
					)
					AND (
						$8 = ''
						OR staff_account_id = NULLIF($8, '')::uuid
					)
					AND (
						$9 = ''
						OR warehouse_id = NULLIF($9, '')::uuid
					)
					AND (
						NOT $10::boolean
						OR occurred_at >= $11::timestamptz
					)
					AND (
						NOT $12::boolean
						OR occurred_at < $13::timestamptz
					)
			`,
			filter.Category,
			filter.Status,
			filter.Currency,
			filter.Scope,
			filter.OrderID,
			filter.ProductID,
			filter.VariantID,
			filter.StaffAccountID,
			filter.WarehouseID,
			filter.FromSet,
			filter.From,
			filter.ToSet,
			filter.To,
		).Scan(
			&total,
		); err != nil {

		return FinanceExpensePage{},
			fmt.Errorf(
				"count finance expenses: %w",
				err,
			)
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					id::text,
					category,
					amount,
					currency,
					occurred_at,
					description,
					COALESCE(order_id::text, ''),
					COALESCE(product_id::text, ''),
					COALESCE(variant_id::text, ''),
					COALESCE(staff_account_id::text, ''),
					COALESCE(warehouse_id::text, ''),
					COALESCE(reference, ''),
					status,
					created_by_staff_id::text,
					COALESCE(voided_by_staff_id::text, ''),
					COALESCE(void_reason, ''),
					voided_at,
					created_at,
					updated_at
				FROM finance_expenses
				WHERE
					($1 = '' OR category = $1)
					AND ($2 = '' OR status = $2)
					AND ($3 = '' OR currency = $3)
					AND (
						$4 = ''
						OR (
							$4 = 'business'
							AND order_id IS NULL
							AND product_id IS NULL
							AND variant_id IS NULL
							AND staff_account_id IS NULL
							AND warehouse_id IS NULL
						)
						OR ($4 = 'order' AND order_id IS NOT NULL)
						OR ($4 = 'product' AND product_id IS NOT NULL)
						OR ($4 = 'variant' AND variant_id IS NOT NULL)
						OR ($4 = 'staff' AND staff_account_id IS NOT NULL)
						OR ($4 = 'warehouse' AND warehouse_id IS NOT NULL)
					)
					AND (
						$5 = ''
						OR order_id = NULLIF($5, '')::uuid
					)
					AND (
						$6 = ''
						OR product_id = NULLIF($6, '')::uuid
					)
					AND (
						$7 = ''
						OR variant_id = NULLIF($7, '')::uuid
					)
					AND (
						$8 = ''
						OR staff_account_id = NULLIF($8, '')::uuid
					)
					AND (
						$9 = ''
						OR warehouse_id = NULLIF($9, '')::uuid
					)
					AND (
						NOT $10::boolean
						OR occurred_at >= $11::timestamptz
					)
					AND (
						NOT $12::boolean
						OR occurred_at < $13::timestamptz
					)
				ORDER BY
					occurred_at DESC,
					created_at DESC,
					id DESC
				LIMIT $14
				OFFSET $15
			`,
			filter.Category,
			filter.Status,
			filter.Currency,
			filter.Scope,
			filter.OrderID,
			filter.ProductID,
			filter.VariantID,
			filter.StaffAccountID,
			filter.WarehouseID,
			filter.FromSet,
			filter.From,
			filter.ToSet,
			filter.To,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return FinanceExpensePage{},
			fmt.Errorf(
				"list finance expenses: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]FinanceExpense,
			0,
			params.Limit,
		)

	for rows.Next() {
		item, err :=
			scanFinanceExpense(
				rows,
			)
		if err != nil {
			return FinanceExpensePage{},
				fmt.Errorf(
					"scan finance expense: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return FinanceExpensePage{},
			fmt.Errorf(
				"iterate finance expenses: %w",
				err,
			)
	}

	return FinanceExpensePage{
		Items: items,

		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetFinanceExpense(
	ctx context.Context,
	expenseID string,
) (
	FinanceExpense,
	error,
) {
	item, err :=
		scanFinanceExpense(
			s.db.QueryRow(
				ctx,
				financeExpenseSelectByIDSQL,
				expenseID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return FinanceExpense{},
			ErrFinanceExpenseNotFound
	}

	if err != nil {
		return FinanceExpense{},
			fmt.Errorf(
				"get finance expense: %w",
				err,
			)
	}

	return item,
		nil
}

func (s *Service) CreateFinanceExpense(
	ctx context.Context,
	input FinanceExpenseInput,
	idempotencyKey string,
	metadata AdminActionMetadata,
) (
	FinanceExpenseCreateResult,
	error,
) {
	normalized, err :=
		normalizeFinanceExpenseInput(
			input,
		)
	if err != nil {
		return FinanceExpenseCreateResult{},
			err
	}

	keyHash, err :=
		financeExpenseIdempotencyHash(
			metadata.StaffAccountID,
			idempotencyKey,
		)
	if err != nil {
		return FinanceExpenseCreateResult{},
			err
	}

	fingerprint, err :=
		financeExpenseRequestFingerprint(
			normalized,
		)
	if err != nil {
		return FinanceExpenseCreateResult{},
			err
	}

	var expenseID string

	inserted :=
		false

	err =
		platformdatabase.WithinTx(
			ctx,
			s.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				if err :=
					validateFinanceExpenseTargetTx(
						ctx,
						tx,
						normalized,
					); err != nil {

					return err
				}

				err :=
					tx.QueryRow(
						ctx,
						`
							INSERT INTO finance_expenses (
								category,
								amount,
								currency,
								occurred_at,
								description,
								order_id,
								product_id,
								variant_id,
								staff_account_id,
								warehouse_id,
								reference,
								status,
								idempotency_key_hash,
								request_fingerprint,
								created_by_staff_id,
								created_at,
								updated_at
							)
							VALUES (
								$1,
								$2,
								$3,
								$4,
								$5,
								NULLIF($6, '')::uuid,
								NULLIF($7, '')::uuid,
								NULLIF($8, '')::uuid,
								NULLIF($9, '')::uuid,
								NULLIF($10, '')::uuid,
								NULLIF($11, ''),
								'active',
								$12,
								$13,
								$14::uuid,
								now(),
								now()
							)
							ON CONFLICT (idempotency_key_hash)
							DO NOTHING
							RETURNING id::text
						`,
						normalized.Category,
						normalized.Amount,
						normalized.Currency,
						normalized.OccurredAt,
						normalized.Description,
						normalized.OrderID,
						normalized.ProductID,
						normalized.VariantID,
						normalized.StaffAccountID,
						normalized.WarehouseID,
						normalized.Reference,
						keyHash,
						fingerprint,
						metadata.StaffAccountID,
					).Scan(
						&expenseID,
					)

				switch {
				case err == nil:
					inserted =
						true

				case errors.Is(
					err,
					pgx.ErrNoRows,
				):
					var existingFingerprint string

					if err :=
						tx.QueryRow(
							ctx,
							`
								SELECT
									id::text,
									request_fingerprint
								FROM finance_expenses
								WHERE idempotency_key_hash = $1
							`,
							keyHash,
						).Scan(
							&expenseID,
							&existingFingerprint,
						); err != nil {

						return fmt.Errorf(
							"load existing finance expense idempotency record: %w",
							err,
						)
					}

					if existingFingerprint !=
						fingerprint {

						return ErrFinanceExpenseIdempotencyConflict
					}

					return nil

				default:
					return fmt.Errorf(
						"create finance expense: %w",
						err,
					)
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventFinanceExpenseCreated,
						map[string]any{
							"finance_expense_id": expenseID,

							"category": normalized.Category,

							"amount": normalized.Amount,

							"currency": normalized.Currency,

							"occurred_at": normalized.OccurredAt,

							"scope": financeExpenseScope(
								normalized.OrderID,
								normalized.ProductID,
								normalized.VariantID,
								normalized.StaffAccountID,
								normalized.WarehouseID,
							),

							"order_id": normalized.OrderID,

							"product_id": normalized.ProductID,

							"variant_id": normalized.VariantID,

							"staff_account_id": normalized.StaffAccountID,

							"warehouse_id": normalized.WarehouseID,
						},
					); err != nil {

					return err
				}

				return nil
			},
		)
	if err != nil {
		return FinanceExpenseCreateResult{},
			err
	}

	expense, err :=
		s.GetFinanceExpense(
			ctx,
			expenseID,
		)
	if err != nil {
		return FinanceExpenseCreateResult{},
			err
	}

	return FinanceExpenseCreateResult{
		Expense: expense,

		Inserted: inserted,
	}, nil
}

func (s *Service) VoidFinanceExpense(
	ctx context.Context,
	expenseID string,
	reason string,
	metadata AdminActionMetadata,
) (
	FinanceExpense,
	error,
) {
	reason =
		strings.TrimSpace(
			reason,
		)

	if reason == "" ||
		utf8.RuneCountInString(
			reason,
		) > 500 {

		return FinanceExpense{},
			fmt.Errorf(
				"%w: void_reason is required and must be at most 500 characters",
				ErrInvalidFinanceExpense,
			)
	}

	err :=
		platformdatabase.WithinTx(
			ctx,
			s.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var status string

				if err :=
					tx.QueryRow(
						ctx,
						`
							SELECT status
							FROM finance_expenses
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						expenseID,
					).Scan(
						&status,
					); err != nil {

					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						return ErrFinanceExpenseNotFound
					}

					return fmt.Errorf(
						"lock finance expense: %w",
						err,
					)
				}

				if status ==
					FinanceExpenseStatusVoided {

					return nil
				}

				if _, err :=
					tx.Exec(
						ctx,
						`
							UPDATE finance_expenses
							SET
								status = 'voided',
								voided_by_staff_id = $2::uuid,
								void_reason = $3,
								voided_at = now(),
								updated_at = now()
							WHERE id = $1::uuid
						`,
						expenseID,
						metadata.StaffAccountID,
						reason,
					); err != nil {

					return fmt.Errorf(
						"void finance expense: %w",
						err,
					)
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventFinanceExpenseVoided,
						map[string]any{
							"finance_expense_id": expenseID,

							"void_reason": reason,
						},
					); err != nil {

					return err
				}

				return nil
			},
		)
	if err != nil {
		return FinanceExpense{},
			err
	}

	return s.GetFinanceExpense(
		ctx,
		expenseID,
	)
}

const financeExpenseSelectByIDSQL = `
	SELECT
		id::text,
		category,
		amount,
		currency,
		occurred_at,
		description,
		COALESCE(order_id::text, ''),
		COALESCE(product_id::text, ''),
		COALESCE(variant_id::text, ''),
		COALESCE(staff_account_id::text, ''),
		COALESCE(warehouse_id::text, ''),
		COALESCE(reference, ''),
		status,
		created_by_staff_id::text,
		COALESCE(voided_by_staff_id::text, ''),
		COALESCE(void_reason, ''),
		voided_at,
		created_at,
		updated_at
	FROM finance_expenses
	WHERE id = $1::uuid
`

type financeExpenseScanner interface {
	Scan(dest ...any) error
}

func scanFinanceExpense(
	scanner financeExpenseScanner,
) (
	FinanceExpense,
	error,
) {
	var item FinanceExpense

	err :=
		scanner.Scan(
			&item.ID,
			&item.Category,
			&item.Amount,
			&item.Currency,
			&item.OccurredAt,
			&item.Description,
			&item.OrderID,
			&item.ProductID,
			&item.VariantID,
			&item.StaffAccountID,
			&item.WarehouseID,
			&item.Reference,
			&item.Status,
			&item.CreatedByStaffID,
			&item.VoidedByStaffID,
			&item.VoidReason,
			&item.VoidedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)

	if err == nil {
		item.Scope =
			financeExpenseScope(
				item.OrderID,
				item.ProductID,
				item.VariantID,
				item.StaffAccountID,
				item.WarehouseID,
			)
	}

	return item,
		err
}

func normalizeFinanceExpenseFilter(
	filter FinanceExpenseFilter,
) FinanceExpenseFilter {
	filter.Category =
		strings.ToLower(
			strings.TrimSpace(
				filter.Category,
			),
		)

	filter.Status =
		strings.ToLower(
			strings.TrimSpace(
				filter.Status,
			),
		)

	filter.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				filter.Currency,
			),
		)

	filter.Scope =
		strings.ToLower(
			strings.TrimSpace(
				filter.Scope,
			),
		)

	filter.OrderID =
		strings.TrimSpace(
			filter.OrderID,
		)

	filter.ProductID =
		strings.TrimSpace(
			filter.ProductID,
		)

	filter.VariantID =
		strings.TrimSpace(
			filter.VariantID,
		)

	filter.StaffAccountID =
		strings.TrimSpace(
			filter.StaffAccountID,
		)

	filter.WarehouseID =
		strings.TrimSpace(
			filter.WarehouseID,
		)

	return filter
}

func validateFinanceExpenseFilter(
	filter FinanceExpenseFilter,
) error {
	if filter.Category != "" &&
		!validFinanceExpenseCategory(
			filter.Category,
		) {

		return fmt.Errorf(
			"%w: invalid category filter",
			ErrInvalidFinanceExpense,
		)
	}

	if !validFinanceExpenseStatus(
		filter.Status,
	) {
		return fmt.Errorf(
			"%w: invalid status filter",
			ErrInvalidFinanceExpense,
		)
	}

	if filter.Currency != "" &&
		!validFinanceCurrency(
			filter.Currency,
		) {

		return fmt.Errorf(
			"%w: currency filter must be a 3-letter uppercase code",
			ErrInvalidFinanceExpense,
		)
	}

	if !validFinanceExpenseScope(
		filter.Scope,
	) {
		return fmt.Errorf(
			"%w: invalid scope filter",
			ErrInvalidFinanceExpense,
		)
	}

	targets :=
		[]struct {
			label string
			value string
		}{
			{
				label: "order_id",
				value: filter.OrderID,
			},
			{
				label: "product_id",
				value: filter.ProductID,
			},
			{
				label: "variant_id",
				value: filter.VariantID,
			},
			{
				label: "staff_account_id",
				value: filter.StaffAccountID,
			},
			{
				label: "warehouse_id",
				value: filter.WarehouseID,
			},
		}

	targetCount :=
		0

	for _, target := range targets {

		if target.value == "" {
			continue
		}

		targetCount++

		if !platformvalidation.IsUUID(
			target.value,
		) {
			return fmt.Errorf(
				"%w: %s filter must be a valid UUID",
				ErrInvalidFinanceExpense,
				target.label,
			)
		}
	}

	if targetCount > 1 {
		return fmt.Errorf(
			"%w: at most one expense target filter may be specified",
			ErrInvalidFinanceExpense,
		)
	}

	if filter.Scope != "" &&
		targetCount == 1 {

		expectedScope :=
			financeExpenseScope(
				filter.OrderID,
				filter.ProductID,
				filter.VariantID,
				filter.StaffAccountID,
				filter.WarehouseID,
			)

		if filter.Scope !=
			expectedScope {

			return fmt.Errorf(
				"%w: scope filter does not match the supplied target filter",
				ErrInvalidFinanceExpense,
			)
		}
	}

	if filter.FromSet &&
		filter.ToSet &&
		!filter.To.After(
			filter.From,
		) {

		return fmt.Errorf(
			"%w: to must be after from",
			ErrInvalidFinanceExpense,
		)
	}

	return nil
}

func normalizeFinanceExpenseInput(
	input FinanceExpenseInput,
) (
	FinanceExpenseInput,
	error,
) {
	input.Category =
		strings.ToLower(
			strings.TrimSpace(
				input.Category,
			),
		)

	if !validFinanceExpenseCategory(
		input.Category,
	) {
		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: invalid category",
				ErrInvalidFinanceExpense,
			)
	}

	if input.Amount <= 0 {
		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: amount must be greater than zero",
				ErrInvalidFinanceExpense,
			)
	}

	input.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				input.Currency,
			),
		)

	if input.Currency == "" {
		input.Currency =
			"BDT"
	}

	if !validFinanceCurrency(
		input.Currency,
	) {
		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: currency must be a 3-letter uppercase code",
				ErrInvalidFinanceExpense,
			)
	}

	if input.OccurredAt.IsZero() {
		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: occurred_at is required",
				ErrInvalidFinanceExpense,
			)
	}

	input.OccurredAt =
		input.OccurredAt.UTC()

	input.Description =
		strings.TrimSpace(
			input.Description,
		)

	if input.Description == "" ||
		utf8.RuneCountInString(
			input.Description,
		) > 500 {

		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: description is required and must be at most 500 characters",
				ErrInvalidFinanceExpense,
			)
	}

	input.OrderID =
		strings.TrimSpace(
			input.OrderID,
		)

	input.ProductID =
		strings.TrimSpace(
			input.ProductID,
		)

	input.VariantID =
		strings.TrimSpace(
			input.VariantID,
		)

	input.StaffAccountID =
		strings.TrimSpace(
			input.StaffAccountID,
		)

	input.WarehouseID =
		strings.TrimSpace(
			input.WarehouseID,
		)

	targets :=
		[]struct {
			label string
			value string
		}{
			{
				label: "order_id",
				value: input.OrderID,
			},
			{
				label: "product_id",
				value: input.ProductID,
			},
			{
				label: "variant_id",
				value: input.VariantID,
			},
			{
				label: "staff_account_id",
				value: input.StaffAccountID,
			},
			{
				label: "warehouse_id",
				value: input.WarehouseID,
			},
		}

	targetCount :=
		0

	for _, target := range targets {

		if target.value == "" {
			continue
		}

		targetCount++

		if !platformvalidation.IsUUID(
			target.value,
		) {
			return FinanceExpenseInput{},
				fmt.Errorf(
					"%w: %s must be a valid UUID",
					ErrInvalidFinanceExpense,
					target.label,
				)
		}
	}

	if targetCount > 1 {
		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: an expense may target only one order, product, variant, staff account, or warehouse",
				ErrInvalidFinanceExpense,
			)
	}

	input.Reference =
		strings.TrimSpace(
			input.Reference,
		)

	if utf8.RuneCountInString(
		input.Reference,
	) > 160 {

		return FinanceExpenseInput{},
			fmt.Errorf(
				"%w: reference must be at most 160 characters",
				ErrInvalidFinanceExpense,
			)
	}

	return input,
		nil
}

func validFinanceExpenseCategory(
	value string,
) bool {
	switch value {
	case FinanceExpenseCategoryDelivery,
		FinanceExpenseCategoryPaymentFee,
		FinanceExpenseCategoryChinaFreight,
		FinanceExpenseCategoryCustoms,
		FinanceExpenseCategoryPackaging,
		FinanceExpenseCategoryMarketing,
		FinanceExpenseCategoryWarehouse,
		FinanceExpenseCategorySalary,
		FinanceExpenseCategoryStaffBenefit,
		FinanceExpenseCategoryRent,
		FinanceExpenseCategoryUtilities,
		FinanceExpenseCategorySoftware,
		FinanceExpenseCategoryProfessionalService,
		FinanceExpenseCategoryBankFee,
		FinanceExpenseCategoryInsurance,
		FinanceExpenseCategoryTaxFee,
		FinanceExpenseCategoryOffice,
		FinanceExpenseCategoryTravel,
		FinanceExpenseCategoryOther:

		return true

	default:
		return false
	}
}

func validFinanceExpenseScope(
	value string,
) bool {
	switch value {
	case "",
		FinanceExpenseScopeBusiness,
		FinanceExpenseScopeOrder,
		FinanceExpenseScopeProduct,
		FinanceExpenseScopeVariant,
		FinanceExpenseScopeStaff,
		FinanceExpenseScopeWarehouse:

		return true

	default:
		return false
	}
}

func financeExpenseScope(
	orderID string,
	productID string,
	variantID string,
	staffAccountID string,
	warehouseID string,
) string {
	switch {
	case orderID != "":
		return FinanceExpenseScopeOrder

	case productID != "":
		return FinanceExpenseScopeProduct

	case variantID != "":
		return FinanceExpenseScopeVariant

	case staffAccountID != "":
		return FinanceExpenseScopeStaff

	case warehouseID != "":
		return FinanceExpenseScopeWarehouse

	default:
		return FinanceExpenseScopeBusiness
	}
}

func validateFinanceExpenseTargetTx(
	ctx context.Context,
	tx pgx.Tx,
	input FinanceExpenseInput,
) error {
	switch {
	case input.OrderID != "":
		return financeExpenseTargetExistsTx(
			ctx,
			tx,
			"orders",
			input.OrderID,
			ErrFinanceExpenseOrderNotFound,
			"order",
		)

	case input.ProductID != "":
		return financeExpenseTargetExistsTx(
			ctx,
			tx,
			"products",
			input.ProductID,
			ErrFinanceExpenseProductNotFound,
			"product",
		)

	case input.VariantID != "":
		return financeExpenseTargetExistsTx(
			ctx,
			tx,
			"product_variants",
			input.VariantID,
			ErrFinanceExpenseVariantNotFound,
			"variant",
		)

	case input.StaffAccountID != "":
		return financeExpenseTargetExistsTx(
			ctx,
			tx,
			"staff_accounts",
			input.StaffAccountID,
			ErrFinanceExpenseStaffNotFound,
			"staff account",
		)

	case input.WarehouseID != "":
		return financeExpenseTargetExistsTx(
			ctx,
			tx,
			"warehouses",
			input.WarehouseID,
			ErrFinanceExpenseWarehouseNotFound,
			"warehouse",
		)

	default:
		return nil
	}
}

func financeExpenseTargetExistsTx(
	ctx context.Context,
	tx pgx.Tx,
	table string,
	targetID string,
	notFound error,
	label string,
) error {
	var query string

	switch table {
	case "orders":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM orders
				WHERE id = $1::uuid
			)
		`

	case "products":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM products
				WHERE id = $1::uuid
			)
		`

	case "product_variants":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM product_variants
				WHERE id = $1::uuid
			)
		`

	case "staff_accounts":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM staff_accounts
				WHERE id = $1::uuid
			)
		`

	case "warehouses":
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM warehouses
				WHERE id = $1::uuid
			)
		`

	default:
		return fmt.Errorf(
			"unsupported finance expense target table %q",
			table,
		)
	}

	var exists bool

	if err :=
		tx.QueryRow(
			ctx,
			query,
			targetID,
		).Scan(
			&exists,
		); err != nil {

		return fmt.Errorf(
			"check finance expense %s: %w",
			label,
			err,
		)
	}

	if !exists {
		return notFound
	}

	return nil
}

func validFinanceExpenseStatus(
	value string,
) bool {
	return value == "" ||
		value == FinanceExpenseStatusActive ||
		value == FinanceExpenseStatusVoided
}

func validFinanceCurrency(
	value string,
) bool {
	if len(value) != 3 {
		return false
	}

	for index :=
		0; index < len(value); index++ {

		if value[index] < 'A' ||
			value[index] > 'Z' {

			return false
		}
	}

	return true
}

func financeExpenseIdempotencyHash(
	staffAccountID string,
	key string,
) (
	string,
	error,
) {
	staffAccountID =
		strings.TrimSpace(
			staffAccountID,
		)

	key =
		strings.TrimSpace(
			key,
		)

	if staffAccountID == "" ||
		!platformvalidation.IsUUID(
			staffAccountID,
		) {

		return "",
			fmt.Errorf(
				"%w: invalid Admin staff identity",
				ErrInvalidFinanceExpense,
			)
	}

	if key == "" ||
		utf8.RuneCountInString(
			key,
		) > 200 {

		return "",
			fmt.Errorf(
				"%w: Idempotency-Key is required and must be at most 200 characters",
				ErrInvalidFinanceExpense,
			)
	}

	digest :=
		sha256.Sum256(
			[]byte(
				staffAccountID +
					"\x00" +
					key,
			),
		)

	return hex.EncodeToString(
		digest[:],
	), nil
}

func financeExpenseRequestFingerprint(
	input FinanceExpenseInput,
) (
	string,
	error,
) {
	payload :=
		struct {
			Category string `json:"category"`
			Amount   int64  `json:"amount"`
			Currency string `json:"currency"`

			OccurredAt string `json:"occurred_at"`

			Description string `json:"description"`

			OrderID        string `json:"order_id"`
			ProductID      string `json:"product_id"`
			VariantID      string `json:"variant_id"`
			StaffAccountID string `json:"staff_account_id"`
			WarehouseID    string `json:"warehouse_id"`

			Reference string `json:"reference"`
		}{
			Category: input.Category,
			Amount:   input.Amount,
			Currency: input.Currency,

			OccurredAt: input.OccurredAt.
				UTC().
				Format(
					time.RFC3339Nano,
				),

			Description: input.Description,

			OrderID: input.OrderID,

			ProductID: input.ProductID,

			VariantID: input.VariantID,

			StaffAccountID: input.StaffAccountID,

			WarehouseID: input.WarehouseID,

			Reference: input.Reference,
		}

	encoded, err :=
		json.Marshal(
			payload,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"encode finance expense request fingerprint: %w",
				err,
			)
	}

	digest :=
		sha256.Sum256(
			encoded,
		)

	return hex.EncodeToString(
		digest[:],
	), nil
}

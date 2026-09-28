package crm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin CRM transaction: %w", err)
	}

	return tx, nil
}

func (r *Repository) GetVariantContextTx(
	ctx context.Context,
	tx pgx.Tx,
	variantID string,
) (ProductContext, error) {
	const query = `
		SELECT
			p.id::text,
			p.product_code,
			p.name,
			p.status,
			v.id::text,
			v.sku,
			v.minimum_order_quantity,
			v.price_amount,
			v.currency,
			COALESCE(i.quantity_on_hand, 0),
			COALESCE(i.quantity_reserved, 0),
			GREATEST(
				COALESCE(i.quantity_on_hand, 0)
				- COALESCE(i.quantity_reserved, 0),
				0
			)
		FROM product_variants v
		JOIN products p
			ON p.id = v.product_id
		LEFT JOIN inventory i
			ON i.variant_id = v.id
		WHERE v.id = $1::uuid
	`

	var result ProductContext

	err := tx.QueryRow(
		ctx,
		query,
		variantID,
	).Scan(
		&result.ProductID,
		&result.ProductCode,
		&result.ProductName,
		&result.Status,
		&result.VariantID,
		&result.SKU,
		&result.MinimumOrderQuantity,
		&result.PriceAmount,
		&result.Currency,
		&result.QuantityOnHand,
		&result.QuantityReserved,
		&result.AvailableQuantity,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return ProductContext{}, ErrVariantNotFound
	}

	if err != nil {
		return ProductContext{}, fmt.Errorf(
			"load CRM variant context: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) GetProductContextTx(
	ctx context.Context,
	tx pgx.Tx,
	productID string,
) (ProductContext, error) {
	var result ProductContext

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				product_code,
				name,
				status
			FROM products
			WHERE id = $1::uuid
		`,
		productID,
	).Scan(
		&result.ProductID,
		&result.ProductCode,
		&result.ProductName,
		&result.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return ProductContext{}, ErrProductNotFound
	}

	if err != nil {
		return ProductContext{}, fmt.Errorf(
			"load CRM product context: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) GetOrderContextTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	orderID string,
) (OrderContext, error) {
	var result OrderContext

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				order_number,
				status,
				payment_status,
				payment_method,
				total_amount,
				currency
			FROM orders
			WHERE
				id = $1::uuid
				AND customer_id = $2::uuid
		`,
		orderID,
		customerID,
	).Scan(
		&result.OrderID,
		&result.OrderNumber,
		&result.Status,
		&result.PaymentStatus,
		&result.PaymentMethod,
		&result.TotalAmount,
		&result.Currency,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return OrderContext{}, ErrOrderNotFound
	}

	if err != nil {
		return OrderContext{}, fmt.Errorf(
			"load CRM order context: %w",
			err,
		)
	}

	return result, nil
}

type createCaseInput struct {
	CaseNumber string
	CustomerID string
	CaseType   string
	Subject    string
	Priority   string

	ProductID string
	VariantID string
	OrderID   string

	RequestedQuantity         *int
	AvailableQuantitySnapshot *int
	ContextSnapshot           []byte
}

func (r *Repository) CreateCaseTx(
	ctx context.Context,
	tx pgx.Tx,
	input createCaseInput,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO crm_cases (
				case_number,
				customer_id,
				case_type,
				subject,
				status,
				priority,
				product_id,
				variant_id,
				order_id,
				requested_quantity,
				available_quantity_snapshot,
				context_snapshot,
				last_message_at,
				last_customer_message_at,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2::uuid,
				$3,
				$4,
				'waiting_support',
				$5,
				NULLIF($6, '')::uuid,
				NULLIF($7, '')::uuid,
				NULLIF($8, '')::uuid,
				$9,
				$10,
				$11::jsonb,
				now(),
				now(),
				now(),
				now()
			)
			RETURNING id::text
		`,
		input.CaseNumber,
		input.CustomerID,
		input.CaseType,
		input.Subject,
		input.Priority,
		input.ProductID,
		input.VariantID,
		input.OrderID,
		input.RequestedQuantity,
		input.AvailableQuantitySnapshot,
		string(input.ContextSnapshot),
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf(
			"create CRM case: %w",
			err,
		)
	}

	return id, nil
}

func (r *Repository) InsertCustomerMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	body string,
	attachments json.RawMessage,
) (Message, error) {
	var result Message

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO crm_messages (
				case_id,
				author_type,
				support_actor_id,
				visibility,
				body,
				attachments,
				created_at
			)
			VALUES (
				$1::uuid,
				'customer',
				NULL,
				'customer',
				$2,
				$3::jsonb,
				now()
			)
			RETURNING
				id::text,
				case_id::text,
				author_type,
				visibility,
				body,
				created_at
		`,
		caseID,
		body,
		crmJSONArgument(attachments),
	).Scan(
		&result.ID,
		&result.CaseID,
		&result.AuthorType,
		&result.Visibility,
		&result.Body,
		&result.CreatedAt,
	)

	if err != nil {
		return Message{}, fmt.Errorf(
			"insert CRM customer message: %w",
			err,
		)
	}

	result.Attachments = cloneCRMRawMessage(
		attachments,
	)

	return result, nil
}

func (r *Repository) InsertCustomerEventTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	eventType string,
	fromStatus string,
	toStatus string,
	payload any,
) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(
			"encode CRM event payload: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO crm_case_events (
				case_id,
				event_type,
				actor_type,
				support_actor_id,
				from_status,
				to_status,
				payload,
				created_at
			)
			VALUES (
				$1::uuid,
				$2,
				'customer',
				NULL,
				NULLIF($3, ''),
				NULLIF($4, ''),
				$5::jsonb,
				now()
			)
		`,
		caseID,
		eventType,
		fromStatus,
		toStatus,
		string(data),
	)

	if err != nil {
		return fmt.Errorf(
			"insert CRM case event: %w",
			err,
		)
	}

	return nil
}

const caseSelect = `
	SELECT
		id::text,
		case_number,
		customer_id::text,
		case_type,
		subject,
		status,
		priority,
		COALESCE(product_id::text, ''),
		COALESCE(variant_id::text, ''),
		COALESCE(order_id::text, ''),
		requested_quantity,
		available_quantity_snapshot,
		context_snapshot::text,
		last_message_at,
		last_customer_message_at,
		last_support_message_at,
		resolved_at,
		closed_at,
		created_at,
		updated_at
	FROM crm_cases
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCase(row rowScanner) (Case, error) {
	var result Case
	var contextText string

	var requested pgtype.Int4
	var available pgtype.Int4

	var lastCustomer pgtype.Timestamptz
	var lastSupport pgtype.Timestamptz
	var resolved pgtype.Timestamptz
	var closed pgtype.Timestamptz

	err := row.Scan(
		&result.ID,
		&result.CaseNumber,
		&result.CustomerID,
		&result.CaseType,
		&result.Subject,
		&result.Status,
		&result.Priority,
		&result.ProductID,
		&result.VariantID,
		&result.OrderID,
		&requested,
		&available,
		&contextText,
		&result.LastMessageAt,
		&lastCustomer,
		&lastSupport,
		&resolved,
		&closed,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		return Case{}, err
	}

	result.ContextSnapshot = json.RawMessage(
		contextText,
	)

	if requested.Valid {
		value := int(
			requested.Int32,
		)

		result.RequestedQuantity = &value
	}

	if available.Valid {
		value := int(
			available.Int32,
		)

		result.AvailableQuantitySnapshot = &value
	}

	result.LastCustomerMessageAt = timestampPtr(
		lastCustomer,
	)

	result.LastSupportMessageAt = timestampPtr(
		lastSupport,
	)

	result.ResolvedAt = timestampPtr(
		resolved,
	)

	result.ClosedAt = timestampPtr(
		closed,
	)

	return result, nil
}

func timestampPtr(
	value pgtype.Timestamptz,
) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time

	return &result
}

func (r *Repository) GetCustomerCase(
	ctx context.Context,
	customerID string,
	caseID string,
) (Case, error) {
	result, err := scanCase(
		r.db.QueryRow(
			ctx,
			caseSelect+`
				WHERE
					id = $1::uuid
					AND customer_id = $2::uuid
			`,
			caseID,
			customerID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Case{}, ErrCaseNotFound
	}

	if err != nil {
		return Case{}, fmt.Errorf(
			"get CRM case: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) CustomerCanAccessAttachment(
	ctx context.Context,
	customerID string,
	attachmentID string,
) (bool, error) {
	var allowed bool

	err := r.db.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM crm_support_attachments a
				JOIN crm_cases c
					ON c.id = a.case_id
				JOIN crm_messages m
					ON m.id = a.message_id
					AND m.case_id = a.case_id
				WHERE
					a.id = $1::uuid
					AND c.customer_id = $2::uuid
					AND m.visibility = 'customer'
			)
		`,
		attachmentID,
		customerID,
	).Scan(
		&allowed,
	)
	if err != nil {
		return false, fmt.Errorf(
			"authorize customer support attachment: %w",
			err,
		)
	}

	return allowed, nil
}

func (r *Repository) ListCustomerCases(
	ctx context.Context,
	customerID string,
	limit int,
	offset int,
) ([]Case, error) {
	rows, err := r.db.Query(
		ctx,
		caseSelect+`
			WHERE customer_id = $1::uuid
			ORDER BY updated_at DESC, id DESC
			LIMIT $2 OFFSET $3
		`,
		customerID,
		limit,
		offset,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"list CRM cases: %w",
			err,
		)
	}

	defer rows.Close()

	result := make(
		[]Case,
		0,
	)

	for rows.Next() {
		item, err := scanCase(
			rows,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan CRM case: %w",
				err,
			)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate CRM cases: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) ListCustomerMessages(
	ctx context.Context,
	customerID string,
	caseID string,
	limit int,
	offset int,
) ([]Message, error) {
	var exists bool

	err := r.db.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM crm_cases
				WHERE
					id = $1::uuid
					AND customer_id = $2::uuid
			)
		`,
		caseID,
		customerID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf(
			"verify CRM case ownership: %w",
			err,
		)
	}

	if !exists {
		return nil, ErrCaseNotFound
	}

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				m.id::text,
				m.case_id::text,
				m.author_type,
				m.visibility,
				m.body,
				COALESCE(
					m.attachments::text,
					''
				),
				COALESCE(
					a.actor_code,
					''
				),
				COALESCE(
					a.actor_type,
					''
				),
				COALESCE(
					a.display_name,
					''
				),
				m.created_at
			FROM crm_messages m
			LEFT JOIN support_actors a
				ON a.id = m.support_actor_id
			WHERE
				m.case_id = $1::uuid
				AND m.visibility = 'customer'
			ORDER BY
				m.created_at ASC,
				m.id ASC
			LIMIT $2 OFFSET $3
		`,
		caseID,
		limit,
		offset,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"list CRM messages: %w",
			err,
		)
	}

	defer rows.Close()

	result := make(
		[]Message,
		0,
	)

	for rows.Next() {
		var item Message
		var attachmentsText string

		err := rows.Scan(
			&item.ID,
			&item.CaseID,
			&item.AuthorType,
			&item.Visibility,
			&item.Body,
			&attachmentsText,
			&item.SupportActorCode,
			&item.SupportActorType,
			&item.SupportActorName,
			&item.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"scan CRM message: %w",
				err,
			)
		}

		if attachmentsText != "" {
			item.Attachments = json.RawMessage(
				attachmentsText,
			)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate CRM messages: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) LockCustomerCaseTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	caseID string,
) (Case, error) {
	result, err := scanCase(
		tx.QueryRow(
			ctx,
			caseSelect+`
				WHERE
					id = $1::uuid
					AND customer_id = $2::uuid
				FOR UPDATE
			`,
			caseID,
			customerID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Case{}, ErrCaseNotFound
	}

	if err != nil {
		return Case{}, fmt.Errorf(
			"lock CRM case: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) MarkCustomerMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	clearResolved bool,
) error {
	_, err := tx.Exec(
		ctx,
		`
			UPDATE crm_cases
			SET
				status = 'waiting_support',
				last_message_at = now(),
				last_customer_message_at = now(),
				resolved_at = CASE
					WHEN $2 THEN NULL
					ELSE resolved_at
				END,
				updated_at = now()
			WHERE id = $1::uuid
		`,
		caseID,
		clearResolved,
	)

	if err != nil {
		return fmt.Errorf(
			"update CRM customer message state: %w",
			err,
		)
	}

	return nil
}

func crmJSONArgument(
	value json.RawMessage,
) any {
	if len(value) == 0 {
		return nil
	}

	return string(value)
}

func cloneCRMRawMessage(
	value json.RawMessage,
) json.RawMessage {
	if len(value) == 0 {
		return nil
	}

	result := make(
		json.RawMessage,
		len(value),
	)

	copy(
		result,
		value,
	)

	return result
}

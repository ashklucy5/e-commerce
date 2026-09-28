package support

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TicketQueue struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type TicketActor struct {
	ID          string `json:"id"`
	ActorCode   string `json:"actor_code"`
	ActorType   string `json:"actor_type"`
	DisplayName string `json:"display_name"`
}

type TicketAssignment struct {
	ID             string       `json:"id"`
	AssignmentType string       `json:"assignment_type"`
	AssignedActor  *TicketActor `json:"assigned_actor,omitempty"`
	AssignedAt     time.Time    `json:"assigned_at"`
	AcceptedAt     *time.Time   `json:"accepted_at,omitempty"`
}

type Ticket struct {
	ID         string `json:"id"`
	CaseNumber string `json:"case_number"`
	CustomerID string `json:"customer_id"`

	CaseType string `json:"case_type"`
	Subject  string `json:"subject"`
	Status   string `json:"status"`
	Priority string `json:"priority"`

	ProductID string `json:"product_id,omitempty"`
	VariantID string `json:"variant_id,omitempty"`
	OrderID   string `json:"order_id,omitempty"`

	RequestedQuantity         *int `json:"requested_quantity,omitempty"`
	AvailableQuantitySnapshot *int `json:"available_quantity_snapshot,omitempty"`

	ContextSnapshot json.RawMessage `json:"context_snapshot"`

	Queue      TicketQueue      `json:"queue"`
	Assignment TicketAssignment `json:"assignment"`

	LastMessageAt         time.Time  `json:"last_message_at"`
	LastCustomerMessageAt *time.Time `json:"last_customer_message_at,omitempty"`
	LastSupportMessageAt  *time.Time `json:"last_support_message_at,omitempty"`

	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const ticketSelect = `
	SELECT
		c.id::text,
		c.case_number,
		c.customer_id::text,
		c.case_type,
		c.subject,
		c.status,
		c.priority,
		COALESCE(c.product_id::text, ''),
		COALESCE(c.variant_id::text, ''),
		COALESCE(c.order_id::text, ''),
		c.requested_quantity,
		c.available_quantity_snapshot,
		c.context_snapshot::text,
		a.id::text,
		q.id::text,
		q.code,
		q.name,
		COALESCE(q.description, ''),
		a.assignment_type,
		COALESCE(a.support_actor_id::text, ''),
		COALESCE(sa.actor_code, ''),
		COALESCE(sa.actor_type, ''),
		COALESCE(sa.display_name, ''),
		a.assigned_at,
		a.accepted_at,
		c.last_message_at,
		c.last_customer_message_at,
		c.last_support_message_at,
		c.resolved_at,
		c.closed_at,
		c.created_at,
		c.updated_at
	FROM crm_cases c
	JOIN support_case_assignments a
		ON a.case_id = c.id
		AND a.released_at IS NULL
	JOIN support_queues q
		ON q.id = a.queue_id
	LEFT JOIN support_actors sa
		ON sa.id = a.support_actor_id
`

type ticketScanner interface {
	Scan(dest ...any) error
}

func scanTicket(
	row ticketScanner,
) (Ticket, error) {
	var result Ticket

	var requested pgtype.Int4
	var available pgtype.Int4

	var accepted pgtype.Timestamptz
	var lastCustomer pgtype.Timestamptz
	var lastSupport pgtype.Timestamptz
	var resolved pgtype.Timestamptz
	var closed pgtype.Timestamptz

	var contextText string

	var assignedActorID string
	var actorCode string
	var actorType string
	var actorName string

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
		&result.Assignment.ID,
		&result.Queue.ID,
		&result.Queue.Code,
		&result.Queue.Name,
		&result.Queue.Description,
		&result.Assignment.AssignmentType,
		&assignedActorID,
		&actorCode,
		&actorType,
		&actorName,
		&result.Assignment.AssignedAt,
		&accepted,
		&result.LastMessageAt,
		&lastCustomer,
		&lastSupport,
		&resolved,
		&closed,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return Ticket{}, err
	}

	result.ContextSnapshot =
		json.RawMessage(
			contextText,
		)

	if requested.Valid {
		value := int(
			requested.Int32,
		)
		result.RequestedQuantity =
			&value
	}

	if available.Valid {
		value := int(
			available.Int32,
		)
		result.AvailableQuantitySnapshot =
			&value
	}

	result.Assignment.AcceptedAt =
		supportTimePtr(
			accepted,
		)

	result.LastCustomerMessageAt =
		supportTimePtr(
			lastCustomer,
		)

	result.LastSupportMessageAt =
		supportTimePtr(
			lastSupport,
		)

	result.ResolvedAt =
		supportTimePtr(
			resolved,
		)

	result.ClosedAt =
		supportTimePtr(
			closed,
		)

	if assignedActorID != "" {
		result.Assignment.AssignedActor =
			&TicketActor{
				ID:          assignedActorID,
				ActorCode:   actorCode,
				ActorType:   actorType,
				DisplayName: actorName,
			}
	}

	return result, nil
}

func supportTimePtr(
	value pgtype.Timestamptz,
) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time
	return &result
}

func (r *PostgresRepository) ListCasesForActor(
	ctx context.Context,
	actorID string,
	limit int,
	offset int,
) ([]Ticket, error) {
	rows, err := r.db.Query(
		ctx,
		ticketSelect+`
			WHERE EXISTS (
				SELECT 1
				FROM support_queue_members qm
				WHERE
					qm.queue_id = a.queue_id
					AND qm.support_actor_id = $1::uuid
			)
			ORDER BY
				CASE c.status
					WHEN 'waiting_support' THEN 0
					WHEN 'waiting_customer' THEN 1
					WHEN 'resolved' THEN 2
					ELSE 3
				END,
				CASE c.priority
					WHEN 'urgent' THEN 0
					WHEN 'high' THEN 1
					WHEN 'normal' THEN 2
					ELSE 3
				END,
				c.last_message_at ASC,
				c.id ASC
			LIMIT $2 OFFSET $3
		`,
		actorID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list support cases: %w",
			err,
		)
	}

	defer rows.Close()

	items := make([]Ticket, 0)

	for rows.Next() {
		item, err := scanTicket(
			rows,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan support case: %w",
				err,
			)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate support cases: %w",
			err,
		)
	}

	return items, nil
}

func (r *PostgresRepository) GetCaseForActor(
	ctx context.Context,
	actorID string,
	caseID string,
) (Ticket, error) {
	result, err := scanTicket(
		r.db.QueryRow(
			ctx,
			ticketSelect+`
				WHERE
					c.id = $1::uuid
					AND EXISTS (
						SELECT 1
						FROM support_queue_members qm
						WHERE
							qm.queue_id = a.queue_id
							AND qm.support_actor_id = $2::uuid
					)
			`,
			caseID,
			actorID,
		),
	)
	if err == pgx.ErrNoRows {
		return Ticket{},
			ErrCaseNotFound
	}
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"get support case: %w",
				err,
			)
	}

	return result, nil
}

func (r *PostgresRepository) ResolveCase(
	ctx context.Context,
	actorID string,
	caseID string,
) (Ticket, error) {
	tx, err := r.db.Begin(
		ctx,
	)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"begin resolve case transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	assignment, err :=
		r.lockActiveAssignmentTx(
			ctx,
			tx,
			caseID,
		)
	if err != nil {
		return Ticket{}, err
	}

	if assignment.CaseStatus ==
		"closed" {
		return Ticket{},
			ErrCaseClosed
	}

	if assignment.SupportActorID !=
		actorID {
		return Ticket{},
			ErrCaseNotOwned
	}

	if assignment.CaseStatus !=
		"resolved" {
		_, err :=
			tx.Exec(
				ctx,
				`
					UPDATE crm_cases
					SET
						status = 'resolved',
						resolved_at = now(),
						updated_at = now()
					WHERE id = $1::uuid
				`,
				caseID,
			)
		if err != nil {
			return Ticket{},
				fmt.Errorf(
					"resolve CRM case: %w",
					err,
				)
		}

		if err :=
			insertSupportEventTx(
				ctx,
				tx,
				caseID,
				actorID,
				"case_resolved",
				assignment.CaseStatus,
				"resolved",
				map[string]any{
					"queue_code": assignment.QueueCode,
				},
			); err != nil {
			return Ticket{}, err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Ticket{},
			fmt.Errorf(
				"commit resolve case: %w",
				err,
			)
	}

	return r.GetCaseForActor(
		ctx,
		actorID,
		caseID,
	)
}

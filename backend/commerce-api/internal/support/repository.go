package support

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetActorByStaffID(
		ctx context.Context,
		staffID string,
	) (Actor, error)

	ListQueuesForActor(
		ctx context.Context,
		actorID string,
	) ([]Queue, error)

	UpdatePresence(
		ctx context.Context,
		actorID string,
		presence string,
	) (Actor, error)

	ListCasesForActor(
		ctx context.Context,
		actorID string,
		limit int,
		offset int,
	) ([]Ticket, error)

	GetCaseForActor(
		ctx context.Context,
		actorID string,
		caseID string,
	) (Ticket, error)

	ListCaseMessagesForActor(
		ctx context.Context,
		actorID string,
		caseID string,
		limit int,
		offset int,
	) ([]SupportMessage, error)

	ClaimCase(
		ctx context.Context,
		actorID string,
		caseID string,
	) (Ticket, error)

	AddSupportMessage(
		ctx context.Context,
		actorID string,
		caseID string,
		body string,
		visibility string,
	) (SupportMessage, error)

	ResolveCase(
		ctx context.Context,
		actorID string,
		caseID string,
	) (Ticket, error)

	CloseCase(
		ctx context.Context,
		actorID string,
		caseID string,
	) (Ticket, error)

	EscalateCase(
		ctx context.Context,
		actorID string,
		caseID string,
		queueCode string,
	) (EscalationResult, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) Repository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) GetActorByStaffID(
	ctx context.Context,
	staffID string,
) (Actor, error) {
	const query = `
		SELECT
			id::text,
			actor_code,
			actor_type,
			staff_account_id::text,
			display_name,
			status,
			presence,
			max_active_cases
		FROM support_actors
		WHERE
			staff_account_id = $1::uuid
			AND actor_type = 'human'
		LIMIT 1
	`

	var actor Actor

	err := r.db.QueryRow(
		ctx,
		query,
		staffID,
	).Scan(
		&actor.ID,
		&actor.ActorCode,
		&actor.ActorType,
		&actor.StaffAccountID,
		&actor.DisplayName,
		&actor.Status,
		&actor.Presence,
		&actor.MaxActiveCases,
	)
	if err != nil {
		return Actor{}, err
	}

	return actor, nil
}

func (r *PostgresRepository) ListQueuesForActor(
	ctx context.Context,
	actorID string,
) ([]Queue, error) {
	const query = `
		SELECT
			q.id::text,
			q.code,
			q.name,
			COALESCE(q.description, ''),
			m.membership_role
		FROM support_queue_members m
		JOIN support_queues q
			ON q.id = m.queue_id
		WHERE
			m.support_actor_id = $1::uuid
			AND q.status = 'active'
		ORDER BY
			q.sort_order,
			q.code
	`

	rows, err := r.db.Query(
		ctx,
		query,
		actorID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list support queues: %w",
			err,
		)
	}

	defer rows.Close()

	queues := make([]Queue, 0)

	for rows.Next() {
		var queue Queue

		if err := rows.Scan(
			&queue.ID,
			&queue.Code,
			&queue.Name,
			&queue.Description,
			&queue.MembershipRole,
		); err != nil {
			return nil, fmt.Errorf(
				"scan support queue: %w",
				err,
			)
		}

		queues = append(
			queues,
			queue,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate support queues: %w",
			err,
		)
	}

	return queues, nil
}

func (r *PostgresRepository) UpdatePresence(
	ctx context.Context,
	actorID string,
	presence string,
) (Actor, error) {
	const query = `
		UPDATE support_actors
		SET
			presence = $2,
			last_presence_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND actor_type = 'human'
			AND status = 'active'
		RETURNING
			id::text,
			actor_code,
			actor_type,
			staff_account_id::text,
			display_name,
			status,
			presence,
			max_active_cases
	`

	var actor Actor

	err := r.db.QueryRow(
		ctx,
		query,
		actorID,
		presence,
	).Scan(
		&actor.ID,
		&actor.ActorCode,
		&actor.ActorType,
		&actor.StaffAccountID,
		&actor.DisplayName,
		&actor.Status,
		&actor.Presence,
		&actor.MaxActiveCases,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Actor{}, pgx.ErrNoRows
		}

		return Actor{}, fmt.Errorf(
			"update support presence: %w",
			err,
		)
	}

	return actor, nil
}

type activeAssignmentLock struct {
	AssignmentID   string
	QueueID        string
	QueueCode      string
	CaseStatus     string
	SupportActorID string
}

func (r *PostgresRepository) lockActiveAssignmentTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
) (activeAssignmentLock, error) {
	const query = `
		SELECT
			a.id::text,
			a.queue_id::text,
			q.code,
			c.status,
			COALESCE(a.support_actor_id::text, '')
		FROM support_case_assignments a
		JOIN support_queues q
			ON q.id = a.queue_id
		JOIN crm_cases c
			ON c.id = a.case_id
		WHERE
			a.case_id = $1::uuid
			AND a.released_at IS NULL
		FOR UPDATE OF a, c
	`

	var result activeAssignmentLock

	err := tx.QueryRow(
		ctx,
		query,
		caseID,
	).Scan(
		&result.AssignmentID,
		&result.QueueID,
		&result.QueueCode,
		&result.CaseStatus,
		&result.SupportActorID,
	)
	if err == pgx.ErrNoRows {
		return activeAssignmentLock{},
			ErrCaseNotFound
	}
	if err != nil {
		return activeAssignmentLock{},
			fmt.Errorf(
				"lock support assignment: %w",
				err,
			)
	}

	return result, nil
}

func actorHasQueueAccessTx(
	ctx context.Context,
	tx pgx.Tx,
	actorID string,
	queueID string,
) (bool, error) {
	var allowed bool

	err := tx.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM support_queue_members
				WHERE
					support_actor_id = $1::uuid
					AND queue_id = $2::uuid
			)
		`,
		actorID,
		queueID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf(
			"check support queue access: %w",
			err,
		)
	}

	return allowed, nil
}

func insertSupportEventTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	actorID string,
	eventType string,
	fromStatus string,
	toStatus string,
	payload any,
) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(
			"encode support event payload: %w",
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
				'support',
				$3::uuid,
				NULLIF($4, ''),
				NULLIF($5, ''),
				$6::jsonb,
				now()
			)
		`,
		caseID,
		eventType,
		actorID,
		fromStatus,
		toStatus,
		string(data),
	)
	if err != nil {
		return fmt.Errorf(
			"insert support case event: %w",
			err,
		)
	}

	return nil
}

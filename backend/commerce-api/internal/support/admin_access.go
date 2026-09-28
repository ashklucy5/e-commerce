package support

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminService exposes the existing CRM/support domain to the Admin panel.
//
// Support Agent/Supervisor roles remain queue-scoped. Other principals that
// explicitly hold admin.crm.* are allowed to work across queues. Mutating
// global Admin operations are still attributed to a real human support actor
// tied to the staff account; no second CRM or message model is introduced.
type AdminService struct {
	db      *pgxpool.Pool
	regular *Service
	repo    *PostgresRepository
}

type AdminQueue struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	MembershipRole string `json:"membership_role,omitempty"`
}

func NewAdminService(db *pgxpool.Pool) *AdminService {
	repo := &PostgresRepository{db: db}
	return &AdminService{
		db:      db,
		repo:    repo,
		regular: NewService(repo),
	}
}

func (s *AdminService) ListQueues(
	ctx context.Context,
	staffID string,
	queueScoped bool,
) ([]AdminQueue, error) {
	if queueScoped {
		queues, err := s.regular.Queues(ctx, staffID)
		if err != nil {
			return nil, err
		}
		items := make([]AdminQueue, 0, len(queues))
		for _, queue := range queues {
			items = append(items, AdminQueue{
				ID:             queue.ID,
				Code:           queue.Code,
				Name:           queue.Name,
				Description:    queue.Description,
				MembershipRole: queue.MembershipRole,
			})
		}
		return items, nil
	}

	rows, err := s.db.Query(
		ctx,
		`
			SELECT
				id::text,
				code,
				name,
				COALESCE(description, '')
			FROM support_queues
			WHERE status = 'active'
			ORDER BY sort_order, code
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("list Admin CRM queues: %w", err)
	}
	defer rows.Close()

	items := make([]AdminQueue, 0)
	for rows.Next() {
		var item AdminQueue
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Name,
			&item.Description,
		); err != nil {
			return nil, fmt.Errorf("scan Admin CRM queue: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Admin CRM queues: %w", err)
	}
	return items, nil
}

func (s *AdminService) ListCases(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	limit int,
	offset int,
) ([]Ticket, int, int, error) {
	if queueScoped {
		return s.regular.ListCases(ctx, staffID, limit, offset)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		return nil, 0, 0, ErrCaseNotFound
	}

	rows, err := s.db.Query(
		ctx,
		ticketSelect+`
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
			LIMIT $1 OFFSET $2
		`,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list Admin CRM cases: %w", err)
	}
	defer rows.Close()

	items := make([]Ticket, 0)
	for rows.Next() {
		item, err := scanTicket(rows)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan Admin CRM case: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("iterate Admin CRM cases: %w", err)
	}

	return items, limit, offset, nil
}

func (s *AdminService) GetCase(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
) (Ticket, error) {
	if !validSupportUUID(caseID) {
		return Ticket{}, ErrCaseNotFound
	}
	if queueScoped {
		return s.regular.GetCase(ctx, staffID, caseID)
	}

	result, err := scanTicket(
		s.db.QueryRow(
			ctx,
			ticketSelect+` WHERE c.id = $1::uuid`,
			caseID,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrCaseNotFound
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("get Admin CRM case: %w", err)
	}
	return result, nil
}

func (s *AdminService) ListMessages(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
	limit int,
	offset int,
) ([]SupportMessage, int, int, error) {
	if !validSupportUUID(caseID) {
		return nil, 0, 0, ErrCaseNotFound
	}
	if queueScoped {
		return s.regular.ListMessages(ctx, staffID, caseID, limit, offset)
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		return nil, 0, 0, ErrCaseNotFound
	}

	var exists bool
	if err := s.db.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM crm_cases WHERE id = $1::uuid)`,
		caseID,
	).Scan(&exists); err != nil {
		return nil, 0, 0, fmt.Errorf("verify Admin CRM case: %w", err)
	}
	if !exists {
		return nil, 0, 0, ErrCaseNotFound
	}

	rows, err := s.db.Query(
		ctx,
		`
			SELECT
				m.id::text,
				m.case_id::text,
				m.author_type,
				m.visibility,
				m.body,
				COALESCE(m.attachments::text, ''),
				COALESCE(a.id::text, ''),
				COALESCE(a.actor_code, ''),
				COALESCE(a.actor_type, ''),
				COALESCE(a.display_name, ''),
				m.created_at
			FROM crm_messages m
			LEFT JOIN support_actors a
				ON a.id = m.support_actor_id
			WHERE m.case_id = $1::uuid
			ORDER BY m.created_at ASC, m.id ASC
			LIMIT $2 OFFSET $3
		`,
		caseID,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list Admin CRM messages: %w", err)
	}
	defer rows.Close()

	items := make([]SupportMessage, 0)
	for rows.Next() {
		var item SupportMessage
		var attachmentsText string
		var actorID, actorCode, actorType, actorName string
		if err := rows.Scan(
			&item.ID,
			&item.CaseID,
			&item.AuthorType,
			&item.Visibility,
			&item.Body,
			&attachmentsText,
			&actorID,
			&actorCode,
			&actorType,
			&actorName,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, 0, fmt.Errorf("scan Admin CRM message: %w", err)
		}
		if attachmentsText != "" && attachmentsText != "null" {
			item.Attachments = json.RawMessage(attachmentsText)
		}
		if actorID != "" {
			item.SupportActor = &TicketActor{
				ID:          actorID,
				ActorCode:   actorCode,
				ActorType:   actorType,
				DisplayName: actorName,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("iterate Admin CRM messages: %w", err)
	}
	return items, limit, offset, nil
}

func (s *AdminService) ClaimCase(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
) (Ticket, error) {
	if !validSupportUUID(caseID) {
		return Ticket{}, ErrCaseNotFound
	}
	if queueScoped {
		return s.regular.ClaimCase(ctx, staffID, caseID)
	}

	actor, err := s.ensureAdminActor(ctx, staffID)
	if err != nil {
		return Ticket{}, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("begin Admin CRM claim transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	assignment, err := s.repo.lockActiveAssignmentTx(ctx, tx, caseID)
	if err != nil {
		return Ticket{}, err
	}
	if assignment.CaseStatus == "closed" {
		return Ticket{}, ErrCaseClosed
	}
	if assignment.SupportActorID != "" {
		if assignment.SupportActorID != actor.ID {
			return Ticket{}, ErrCaseAlreadyClaimed
		}
		if err := tx.Commit(ctx); err != nil {
			return Ticket{}, err
		}
		return s.GetCase(ctx, staffID, false, caseID)
	}

	var activeCases int
	if err := tx.QueryRow(
		ctx,
		`
			SELECT COUNT(*)::integer
			FROM support_case_assignments a
			JOIN crm_cases c ON c.id = a.case_id
			WHERE
				a.support_actor_id = $1::uuid
				AND a.released_at IS NULL
				AND c.status IN ('waiting_support', 'waiting_customer')
		`,
		actor.ID,
	).Scan(&activeCases); err != nil {
		return Ticket{}, fmt.Errorf("count Admin CRM actor cases: %w", err)
	}
	if activeCases >= actor.MaxActiveCases {
		return Ticket{}, ErrActorCapacity
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE support_case_assignments SET released_at = now(), release_reason = 'claimed' WHERE id = $1::uuid`,
		assignment.AssignmentID,
	); err != nil {
		return Ticket{}, fmt.Errorf("release Admin CRM queued assignment: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`
			INSERT INTO support_case_assignments (
				case_id, queue_id, support_actor_id, assigned_by_actor_id,
				assignment_type, assigned_at, accepted_at
			)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $3::uuid, 'claimed', now(), now())
		`,
		caseID,
		assignment.QueueID,
		actor.ID,
	); err != nil {
		return Ticket{}, fmt.Errorf("create Admin CRM claimed assignment: %w", err)
	}

	if err := insertSupportEventTx(
		ctx,
		tx,
		caseID,
		actor.ID,
		"case_claimed",
		assignment.CaseStatus,
		assignment.CaseStatus,
		map[string]any{"queue_code": assignment.QueueCode, "source": "admin_panel"},
	); err != nil {
		return Ticket{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, fmt.Errorf("commit Admin CRM claim: %w", err)
	}
	return s.GetCase(ctx, staffID, false, caseID)
}

func (s *AdminService) Reply(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
	request ReplyRequest,
) (SupportMessage, error) {
	if queueScoped {
		return s.regular.Reply(ctx, staffID, caseID, request)
	}
	if !validSupportUUID(caseID) {
		return SupportMessage{}, ErrCaseNotFound
	}

	body := strings.TrimSpace(request.Message)
	if body == "" || len([]rune(body)) > 5000 {
		return SupportMessage{}, ErrInvalidMessage
	}
	visibility := strings.ToLower(strings.TrimSpace(request.Visibility))
	if visibility == "" {
		visibility = "customer"
	}
	if visibility != "customer" && visibility != "internal" {
		return SupportMessage{}, ErrInvalidVisibility
	}

	actor, err := s.ensureAdminActor(ctx, staffID)
	if err != nil {
		return SupportMessage{}, err
	}
	return s.repo.AddSupportMessage(ctx, actor.ID, caseID, body, visibility)
}

func (s *AdminService) Resolve(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
) (Ticket, error) {
	if queueScoped {
		return s.regular.Resolve(ctx, staffID, caseID)
	}
	if !validSupportUUID(caseID) {
		return Ticket{}, ErrCaseNotFound
	}
	actor, err := s.ensureAdminActor(ctx, staffID)
	if err != nil {
		return Ticket{}, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("begin Admin CRM resolve transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	assignment, err := s.repo.lockActiveAssignmentTx(ctx, tx, caseID)
	if err != nil {
		return Ticket{}, err
	}
	if assignment.CaseStatus == "closed" {
		return Ticket{}, ErrCaseClosed
	}
	if assignment.SupportActorID != actor.ID {
		return Ticket{}, ErrCaseNotOwned
	}

	if assignment.CaseStatus != "resolved" {
		if _, err := tx.Exec(
			ctx,
			`UPDATE crm_cases SET status = 'resolved', resolved_at = now(), updated_at = now() WHERE id = $1::uuid`,
			caseID,
		); err != nil {
			return Ticket{}, fmt.Errorf("resolve Admin CRM case: %w", err)
		}
		if err := insertSupportEventTx(
			ctx,
			tx,
			caseID,
			actor.ID,
			"case_resolved",
			assignment.CaseStatus,
			"resolved",
			map[string]any{"queue_code": assignment.QueueCode, "source": "admin_panel"},
		); err != nil {
			return Ticket{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, fmt.Errorf("commit Admin CRM resolve: %w", err)
	}
	return s.GetCase(ctx, staffID, false, caseID)
}

func (s *AdminService) Escalate(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
	request EscalateRequest,
) (EscalationResult, error) {
	if queueScoped {
		return s.regular.Escalate(ctx, staffID, caseID, request)
	}
	if !validSupportUUID(caseID) {
		return EscalationResult{}, ErrCaseNotFound
	}
	queueCode := strings.ToLower(strings.TrimSpace(request.QueueCode))
	if queueCode == "" || len(queueCode) > 80 {
		return EscalationResult{}, ErrInvalidQueue
	}
	actor, err := s.ensureAdminActor(ctx, staffID)
	if err != nil {
		return EscalationResult{}, err
	}
	return s.repo.EscalateCase(ctx, actor.ID, caseID, queueCode)
}

func (s *AdminService) ensureAdminActor(
	ctx context.Context,
	staffID string,
) (Actor, error) {
	if !validSupportUUID(staffID) {
		return Actor{}, ErrNotSupportActor
	}

	actor, err := s.repo.GetActorByStaffID(ctx, staffID)
	if err == nil {
		if actor.Status != "active" {
			return Actor{}, ErrSupportActorDisabled
		}
		return actor, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, err
	}

	actorCode := "SUP-ADM-" + strings.ReplaceAll(staffID, "-", "")
	if len(actorCode) > 40 {
		return Actor{}, ErrNotSupportActor
	}

	_, err = s.db.Exec(
		ctx,
		`
			INSERT INTO support_actors (
				actor_code,
				actor_type,
				staff_account_id,
				display_name,
				status,
				presence,
				max_active_cases,
				created_at,
				updated_at
			)
			SELECT
				$2,
				'human',
				sa.id,
				sa.full_name,
				'active',
				'offline',
				50,
				now(),
				now()
			FROM staff_accounts sa
			WHERE sa.id = $1::uuid AND sa.status = 'active'
			ON CONFLICT DO NOTHING
		`,
		staffID,
		actorCode,
	)
	if err != nil {
		return Actor{}, fmt.Errorf("ensure Admin CRM support actor: %w", err)
	}

	actor, err = s.repo.GetActorByStaffID(ctx, staffID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, ErrNotSupportActor
	}
	if err != nil {
		return Actor{}, err
	}
	if actor.Status != "active" {
		return Actor{}, ErrSupportActorDisabled
	}
	return actor, nil
}

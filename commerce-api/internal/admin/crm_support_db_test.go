package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"project.local/commerce-api/internal/support"
)

func TestAdminCRMSupportDBGlobalLifecycle(t *testing.T) {
	f := newAdminRecoveryDBFixture(t)
	ctx := context.Background()

	adminID := f.addStaff("crm-global-admin", "active", RoleAdministrator)
	seed := newAdminCRMTestSeed(t, f, "global")
	defer seed.cleanup()

	svc := support.NewAdminService(f.db)

	queues, err := svc.ListQueues(ctx, adminID, false)
	if err != nil {
		t.Fatalf("list Admin CRM queues: %v", err)
	}
	if queues == nil {
		t.Fatal("Admin CRM queues response is nil")
	}

	items, _, _, err := svc.ListCases(ctx, adminID, false, 50, 0)
	if err != nil {
		t.Fatalf("list Admin CRM cases: %v", err)
	}
	if len(items) != 1 || items[0].ID != seed.caseID {
		t.Fatalf("unexpected Admin CRM case list: %+v", items)
	}

	claimed, err := svc.ClaimCase(ctx, adminID, false, seed.caseID)
	if err != nil {
		t.Fatalf("claim Admin CRM case: %v", err)
	}
	if claimed.Assignment.AssignedActor == nil {
		t.Fatal("claimed Admin CRM case has no support actor")
	}
	if claimed.Assignment.AssignedActor.DisplayName == "" {
		t.Fatal("claimed Admin CRM actor has no display name")
	}

	var actorCode string
	if err := f.db.QueryRow(
		ctx,
		`SELECT actor_code FROM support_actors WHERE staff_account_id = $1::uuid`,
		adminID,
	).Scan(&actorCode); err != nil {
		t.Fatalf("load lazily-created Admin CRM actor: %v", err)
	}
	if !strings.HasPrefix(actorCode, "SUP-ADM-") {
		t.Fatalf("Admin CRM actor code = %q, want SUP-ADM-*", actorCode)
	}

	reply, err := svc.Reply(
		ctx,
		adminID,
		false,
		seed.caseID,
		support.ReplyRequest{
			Message:    "We are checking this for you.",
			Visibility: "customer",
		},
	)
	if err != nil {
		t.Fatalf("reply to Admin CRM case: %v", err)
	}
	if reply.AuthorType != "support" || reply.SupportActor == nil {
		t.Fatalf("unexpected Admin CRM reply: %+v", reply)
	}

	messages, _, _, err := svc.ListMessages(ctx, adminID, false, seed.caseID, 100, 0)
	if err != nil {
		t.Fatalf("list Admin CRM messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("Admin CRM messages count = %d, want 2", len(messages))
	}

	resolved, err := svc.Resolve(ctx, adminID, false, seed.caseID)
	if err != nil {
		t.Fatalf("resolve Admin CRM case: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("resolved Admin CRM status = %q, want resolved", resolved.Status)
	}

	escalated, err := svc.Escalate(
		ctx,
		adminID,
		false,
		seed.caseID,
		support.EscalateRequest{QueueCode: seed.targetQueueCode},
	)
	if err != nil {
		t.Fatalf("escalate Admin CRM case: %v", err)
	}
	if escalated.Status != "waiting_support" || escalated.QueueCode != seed.targetQueueCode {
		t.Fatalf("unexpected Admin CRM escalation result: %+v", escalated)
	}
}

func TestAdminCRMSupportDBSupportRoleRemainsQueueScoped(t *testing.T) {
	f := newAdminRecoveryDBFixture(t)
	ctx := context.Background()

	staffID := f.addStaff("crm-queue-scoped", "active")
	seed := newAdminCRMTestSeed(t, f, "scoped")
	defer seed.cleanup()

	var actorID string
	actorCode := "SUP-TST-" + strings.ReplaceAll(staffID, "-", "")[:12]
	if err := f.db.QueryRow(
		ctx,
		`
			INSERT INTO support_actors (
				actor_code, actor_type, staff_account_id, display_name,
				status, presence, max_active_cases, created_at, updated_at
			)
			VALUES ($1, 'human', $2::uuid, 'Queue Scoped Agent', 'active', 'offline', 10, now(), now())
			RETURNING id::text
		`,
		actorCode,
		staffID,
	).Scan(&actorID); err != nil {
		t.Fatalf("insert queue-scoped support actor: %v", err)
	}

	if _, err := f.db.Exec(
		ctx,
		`
			INSERT INTO support_queue_members (queue_id, support_actor_id, membership_role, created_at)
			VALUES ($1::uuid, $2::uuid, 'member', now())
		`,
		seed.queueID,
		actorID,
	); err != nil {
		t.Fatalf("insert queue membership: %v", err)
	}

	secondCaseID := seed.caseInTargetQueue(t, f)

	svc := support.NewAdminService(f.db)
	items, _, _, err := svc.ListCases(ctx, staffID, true, 50, 0)
	if err != nil {
		t.Fatalf("list queue-scoped Admin CRM cases: %v", err)
	}
	if len(items) != 1 || items[0].ID != seed.caseID {
		t.Fatalf("queue-scoped cases = %+v, want only %s", items, seed.caseID)
	}

	_, err = svc.GetCase(ctx, staffID, true, secondCaseID)
	if !errors.Is(err, support.ErrCaseNotFound) {
		t.Fatalf("out-of-queue CRM read error = %v, want %v", err, support.ErrCaseNotFound)
	}

	_, err = svc.ClaimCase(ctx, staffID, true, secondCaseID)
	if !errors.Is(err, support.ErrCaseNotFound) {
		t.Fatalf("out-of-queue CRM claim error = %v, want %v", err, support.ErrCaseNotFound)
	}
}

type adminCRMTestSeed struct {
	f               *adminBanDBFixture
	customerID      string
	queueID         string
	queueCode       string
	targetQueueID   string
	targetQueueCode string
	caseID          string
	caseIDs         []string
}

func newAdminCRMTestSeed(t *testing.T, f *adminBanDBFixture, label string) *adminCRMTestSeed {
	t.Helper()
	ctx := context.Background()
	tag := testHash(fmt.Sprintf("admin-crm-%s-%d", label, time.Now().UnixNano()))[:12]

	seed := &adminCRMTestSeed{
		f:               f,
		queueCode:       "crm_" + tag,
		targetQueueCode: "crm_target_" + tag,
	}

	if err := f.db.QueryRow(
		ctx,
		`
			INSERT INTO customers (phone, email, password_hash, full_name, status, created_at, updated_at)
			VALUES ($1, $2, 'integration-test-password-hash', 'CRM Test Customer', 'active', now(), now())
			RETURNING id::text
		`,
		"+8801"+tag,
		tag+"@crm.integration.test",
	).Scan(&seed.customerID); err != nil {
		t.Fatalf("insert CRM test customer: %v", err)
	}

	for code, target := range map[string]*string{
		seed.queueCode:       &seed.queueID,
		seed.targetQueueCode: &seed.targetQueueID,
	} {
		if err := f.db.QueryRow(
			ctx,
			`
				INSERT INTO support_queues (code, name, description, status, sort_order, created_at, updated_at)
				VALUES ($1, $2, 'Admin CRM integration test', 'active', 10, now(), now())
				RETURNING id::text
			`,
			code,
			"CRM Test "+code,
		).Scan(target); err != nil {
			t.Fatalf("insert CRM test queue %s: %v", code, err)
		}
	}

	seed.caseID = seed.insertCase(t, seed.queueID, seed.queueCode, "Primary CRM test case")
	return seed
}

func (s *adminCRMTestSeed) insertCase(t *testing.T, queueID, queueCode, subject string) string {
	t.Helper()
	ctx := context.Background()
	var caseID string
	caseNumber := "CRM-TST-" + testHash(subject + fmt.Sprint(time.Now().UnixNano()))[:12]

	if err := s.f.db.QueryRow(
		ctx,
		`
			INSERT INTO crm_cases (
				case_number, customer_id, case_type, subject, status, priority,
				context_snapshot, last_message_at, last_customer_message_at, created_at, updated_at
			)
			VALUES ($1, $2::uuid, 'general_question', $3, 'waiting_support', 'normal', '{}'::jsonb, now(), now(), now(), now())
			RETURNING id::text
		`,
		caseNumber,
		s.customerID,
		subject,
	).Scan(&caseID); err != nil {
		t.Fatalf("insert CRM test case: %v", err)
	}

	if _, err := s.f.db.Exec(
		ctx,
		`
			INSERT INTO support_case_assignments (
				case_id, queue_id, support_actor_id, assigned_by_actor_id, assignment_type, assigned_at
			)
			VALUES ($1::uuid, $2::uuid, NULL, NULL, 'automatic', now())
		`,
		caseID,
		queueID,
	); err != nil {
		t.Fatalf("insert CRM test assignment for %s: %v", queueCode, err)
	}

	if _, err := s.f.db.Exec(
		ctx,
		`
			INSERT INTO crm_messages (
				case_id, author_type, support_actor_id, visibility, body, attachments, created_at
			)
			VALUES ($1::uuid, 'customer', NULL, 'customer', 'Initial customer message', NULL, now())
		`,
		caseID,
	); err != nil {
		t.Fatalf("insert CRM test customer message: %v", err)
	}

	s.caseIDs = append(s.caseIDs, caseID)
	return caseID
}

func (s *adminCRMTestSeed) caseInTargetQueue(t *testing.T, f *adminBanDBFixture) string {
	t.Helper()
	_ = f
	return s.insertCase(t, s.targetQueueID, s.targetQueueCode, "Out-of-queue CRM test case")
}

func (s *adminCRMTestSeed) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exec := func(label string, query string, args ...any) {
		s.f.t.Helper()

		if _, err := s.f.db.Exec(ctx, query, args...); err != nil {
			s.f.t.Errorf("Admin CRM test cleanup %s: %v", label, err)
		}
	}

	/*
		Cases own CRM messages, case events, assignments and related
		notification/sourcing rows through cascading foreign keys. Remove
		them first so any lazily-created support actor can be deleted safely.
	*/
	for _, caseID := range s.caseIDs {
		exec(
			"case",
			`DELETE FROM crm_cases WHERE id = $1::uuid`,
			caseID,
		)
	}

	for _, queueID := range []string{s.queueID, s.targetQueueID} {
		exec(
			"queue membership",
			`DELETE FROM support_queue_members WHERE queue_id = $1::uuid`,
			queueID,
		)
	}

	/*
		The Admin CRM adapter may lazily create a human support_actor for
		an Administrator/Super Admin. support_actors intentionally has a
		NO ACTION FK to staff_accounts, so these actor rows must be removed
		before the generic Admin fixture can delete its staff identities.
	*/
	for _, staffID := range s.f.staffIDs {
		exec(
			"support actor",
			`DELETE FROM support_actors WHERE staff_account_id = $1::uuid`,
			staffID,
		)
	}

	for _, queueID := range []string{s.queueID, s.targetQueueID} {
		exec(
			"queue",
			`DELETE FROM support_queues WHERE id = $1::uuid`,
			queueID,
		)
	}

	exec(
		"customer",
		`DELETE FROM customers WHERE id = $1::uuid`,
		s.customerID,
	)

	/*
		Delete the fixture staff here as well instead of relying solely on
		the later generic fixture cleanup. That makes consecutive opt-in DB
		tests deterministic. The generic cleanup remains registered and its
		repeated deletes are intentionally harmless.
	*/
	for _, staffID := range s.f.staffIDs {
		exec(
			"Admin security events",
			`
				DELETE FROM admin_security_events
				WHERE
					staff_account_id = $1::uuid
					OR details ->> 'target_id' = $1::text
					OR details ->> 'staff_account_id' = $1::text
			`,
			staffID,
		)

		exec(
			"account bans",
			`
				DELETE FROM account_bans
				WHERE
					staff_account_id = $1::uuid
					OR issued_by_staff_id = $1::uuid
					OR revoked_by_staff_id = $1::uuid
			`,
			staffID,
		)
	}

	for index := len(s.f.staffIDs) - 1; index >= 0; index-- {
		exec(
			"staff account",
			`DELETE FROM staff_accounts WHERE id = $1::uuid`,
			s.f.staffIDs[index],
		)
	}
}

package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func TestStaffInvitationAdminDBLifecycle(
	t *testing.T,
) {
	f := newAdminRecoveryDBFixture(t)
	ctx := context.Background()

	actorID := f.addStaff(
		"invitation-super-admin",
		"active",
		RoleSuperAdmin,
	)

	targetID := f.addStaff(
		"invitation-target",
		"pending_activation",
		RoleCatalog,
	)

	oldInvitationID, oldToken := seedAdminStaffInvitation(
		t,
		f,
		targetID,
		actorID,
		time.Now().UTC().Add(2*time.Hour),
	)

	initial, err := f.service.GetStaffInvitation(ctx, targetID)
	if err != nil {
		t.Fatalf("get initial staff invitation: %v", err)
	}
	if initial.ID != oldInvitationID || initial.Status != "pending" {
		t.Fatalf("unexpected initial invitation: %+v", initial)
	}

	reissued, err := f.service.ReissueStaffInvitation(
		ctx,
		targetID,
		f.metadata(actorID),
	)
	if err != nil {
		t.Fatalf("reissue staff invitation: %v", err)
	}

	if reissued.ActivationToken == "" {
		t.Fatal("reissued invitation returned an empty activation token")
	}
	if reissued.ActivationToken == oldToken {
		t.Fatal("reissued invitation reused the previous activation token")
	}
	if reissued.Invitation.ID == oldInvitationID {
		t.Fatal("reissued invitation reused the previous invitation row")
	}
	if reissued.Invitation.Status != "pending" {
		t.Fatalf("reissued invitation status = %q, want pending", reissued.Invitation.Status)
	}
	if reissued.Invitation.DeliveryMode != StaffInvitationDeliveryManual {
		t.Fatalf(
			"reissued delivery mode = %q, want %q",
			reissued.Invitation.DeliveryMode,
			StaffInvitationDeliveryManual,
		)
	}

	var oldStatus string
	if err := f.db.QueryRow(
		ctx,
		`SELECT status FROM staff_invitations WHERE id = $1::uuid`,
		oldInvitationID,
	).Scan(&oldStatus); err != nil {
		t.Fatalf("read old invitation after reissue: %v", err)
	}
	if oldStatus != "cancelled" {
		t.Fatalf("old invitation status = %q, want cancelled", oldStatus)
	}

	var persistedTokenHash string
	if err := f.db.QueryRow(
		ctx,
		`SELECT token_hash FROM staff_invitations WHERE id = $1::uuid`,
		reissued.Invitation.ID,
	).Scan(&persistedTokenHash); err != nil {
		t.Fatalf("read reissued invitation token hash: %v", err)
	}
	if persistedTokenHash != platformsecurity.HashToken(reissued.ActivationToken) {
		t.Fatal("reissued invitation did not persist the activation token hash")
	}

	cancelled, err := f.service.CancelStaffInvitation(
		ctx,
		targetID,
		f.metadata(actorID),
	)
	if err != nil {
		t.Fatalf("cancel staff invitation: %v", err)
	}
	if cancelled.ID != reissued.Invitation.ID || cancelled.Status != "cancelled" {
		t.Fatalf("unexpected cancelled invitation: %+v", cancelled)
	}
	if cancelled.CancelledAt == nil {
		t.Fatal("cancelled invitation has no cancelled_at timestamp")
	}

	latest, err := f.service.GetStaffInvitation(ctx, targetID)
	if err != nil {
		t.Fatalf("get cancelled invitation: %v", err)
	}
	if latest.ID != cancelled.ID || latest.Status != "cancelled" {
		t.Fatalf("latest invitation after cancellation: %+v", latest)
	}

	// Cancellation does not delete the staff identity. The administrator can
	// later restart onboarding by issuing a fresh one-time token.
	restarted, err := f.service.ReissueStaffInvitation(
		ctx,
		targetID,
		f.metadata(actorID),
	)
	if err != nil {
		t.Fatalf("reissue after cancellation: %v", err)
	}
	if restarted.Invitation.Status != "pending" || restarted.ActivationToken == "" {
		t.Fatalf("unexpected restarted invitation: %+v", restarted)
	}

	assertStaffInvitationAudit(
		t,
		f,
		actorID,
		adminEventStaffInvitationReissued,
		targetID,
		2,
	)
	assertStaffInvitationAudit(
		t,
		f,
		actorID,
		adminEventStaffInvitationCancelled,
		targetID,
		1,
	)
}

func TestStaffInvitationAdminDBExpiredAndProtectedBoundaries(
	t *testing.T,
) {
	f := newAdminRecoveryDBFixture(t)
	ctx := context.Background()

	superAdminID := f.addStaff(
		"invitation-expiry-super-admin",
		"active",
		RoleSuperAdmin,
	)

	expiredTargetID := f.addStaff(
		"invitation-expired-target",
		"pending_activation",
		RoleCatalog,
	)

	expiredInvitationID, _ := seedAdminStaffInvitation(
		t,
		f,
		expiredTargetID,
		superAdminID,
		time.Now().UTC().Add(-time.Minute),
	)

	effective, err := f.service.GetStaffInvitation(ctx, expiredTargetID)
	if err != nil {
		t.Fatalf("get expired staff invitation: %v", err)
	}
	if effective.Status != "expired" {
		t.Fatalf("effective expired invitation status = %q, want expired", effective.Status)
	}

	if _, err := f.service.ReissueStaffInvitation(
		ctx,
		expiredTargetID,
		f.metadata(superAdminID),
	); err != nil {
		t.Fatalf("reissue expired staff invitation: %v", err)
	}

	var persistedExpiredStatus string
	if err := f.db.QueryRow(
		ctx,
		`SELECT status FROM staff_invitations WHERE id = $1::uuid`,
		expiredInvitationID,
	).Scan(&persistedExpiredStatus); err != nil {
		t.Fatalf("read persisted expired invitation status: %v", err)
	}
	if persistedExpiredStatus != "expired" {
		t.Fatalf(
			"persisted expired invitation status = %q, want expired",
			persistedExpiredStatus,
		)
	}

	administratorID := f.addStaff(
		"invitation-administrator",
		"active",
		RoleAdministrator,
	)

	protectedTargetID := f.addStaff(
		"invitation-protected-target",
		"pending_activation",
		RoleSecurity,
	)

	seedAdminStaffInvitation(
		t,
		f,
		protectedTargetID,
		superAdminID,
		time.Now().UTC().Add(time.Hour),
	)

	_, err = f.service.ReissueStaffInvitation(
		ctx,
		protectedTargetID,
		f.metadata(administratorID),
	)
	if !errors.Is(err, ErrAdminProtectedStaffMutation) {
		t.Fatalf(
			"Administrator reissue protected invitation error = %v, want %v",
			err,
			ErrAdminProtectedStaffMutation,
		)
	}

	_, err = f.service.CancelStaffInvitation(
		ctx,
		protectedTargetID,
		f.metadata(administratorID),
	)
	if !errors.Is(err, ErrAdminProtectedStaffMutation) {
		t.Fatalf(
			"Administrator cancel protected invitation error = %v, want %v",
			err,
			ErrAdminProtectedStaffMutation,
		)
	}
}

func seedAdminStaffInvitation(
	t *testing.T,
	f *adminBanDBFixture,
	staffID string,
	creatorID string,
	expiresAt time.Time,
) (
	string,
	string,
) {
	t.Helper()

	token, err := generateStaffOnboardingSecret(staffOnboardingTokenBytes)
	if err != nil {
		t.Fatalf("generate test staff invitation token: %v", err)
	}

	var invitationID string
	if err := f.db.QueryRow(
		context.Background(),
		`
			INSERT INTO staff_invitations (
				staff_account_id,
				email,
				token_hash,
				delivery_mode,
				delivery_status,
				status,
				expires_at,
				created_by_staff_id,
				created_at,
				updated_at
			)
			SELECT
				sa.id,
				sa.email,
				$2,
				'manual',
				'not_requested',
				'pending',
				$3,
				$4::uuid,
				now() - interval '2 minutes',
				now() - interval '2 minutes'
			FROM staff_accounts sa
			WHERE sa.id = $1::uuid
			RETURNING id::text
		`,
		staffID,
		platformsecurity.HashToken(token),
		expiresAt,
		creatorID,
	).Scan(&invitationID); err != nil {
		t.Fatalf("seed staff invitation: %v", err)
	}

	return invitationID, token
}

func assertStaffInvitationAudit(
	t *testing.T,
	f *adminBanDBFixture,
	actorID string,
	eventType string,
	targetID string,
	expected int,
) {
	t.Helper()

	var count int
	if err := f.db.QueryRow(
		context.Background(),
		`
			SELECT COUNT(*)::integer
			FROM admin_security_events
			WHERE
				staff_account_id = $1::uuid
				AND event_type = $2
				AND details ->> 'staff_account_id' = $3
		`,
		actorID,
		eventType,
		targetID,
	).Scan(&count); err != nil {
		t.Fatalf("count %s audit events: %v", eventType, err)
	}

	if count != expected {
		t.Fatalf(
			"%s audit event count = %d, want %d",
			eventType,
			count,
			expected,
		)
	}
}

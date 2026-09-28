package admin

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	runAdminRecoveryDBTestsEnv = "RUN_ADMIN_RECOVERY_DB_TEST"
	adminRecoveryTestDBURLEnv  = "ADMIN_RECOVERY_TEST_DATABASE_URL"
)

func TestStaffSecurityRecoveryDBPasswordAndMFA(
	t *testing.T,
) {
	f := newAdminRecoveryDBFixture(t)

	superAdminID := f.addStaff(
		"recovery-super-admin",
		"active",
		RoleSuperAdmin,
	)

	passwordTargetID := f.addStaff(
		"password-target",
		"active",
		RoleAdministrator,
	)
	f.seedSecurityState(passwordTargetID)
	seedAdminRecoveryMFA(t, f.db, passwordTargetID)

	const newPassword = "EneDei!Admin-Recovery-2026"

	_, err := f.service.ResetStaffPassword(
		context.Background(),
		passwordTargetID,
		newPassword,
		f.metadata(superAdminID),
	)
	if err != nil {
		t.Fatalf("Super Admin password reset: %v", err)
	}

	assertAdminRecoveryPassword(t, f.db, passwordTargetID, newPassword)
	assertAdminRecoverySessionsRevoked(t, f.db, passwordTargetID, "staff_password_reset")
	assertAdminRecoveryMFAState(t, f.db, passwordTargetID, "active", 1)
	assertAdminRecoveryAudit(t, f.db, superAdminID, adminEventStaffPasswordReset, passwordTargetID)

	mfaTargetID := f.addStaff(
		"mfa-target",
		"active",
		RoleAdministrator,
	)
	f.seedSecurityState(mfaTargetID)
	seedAdminRecoveryMFA(t, f.db, mfaTargetID)

	_, err = f.service.ResetStaffAdminMFA(
		context.Background(),
		mfaTargetID,
		f.metadata(superAdminID),
	)
	if err != nil {
		t.Fatalf("Super Admin MFA reset: %v", err)
	}

	assertAdminRecoverySessionsRevoked(t, f.db, mfaTargetID, "admin_mfa_reset")
	assertAdminRecoveryMFAState(t, f.db, mfaTargetID, "disabled", 0)
	assertAdminRecoveryAudit(t, f.db, superAdminID, adminEventStaffMFAReset, mfaTargetID)
}

func TestStaffSecurityRecoveryDBProtectedRoleBoundary(
	t *testing.T,
) {
	f := newAdminRecoveryDBFixture(t)

	administratorID := f.addStaff(
		"administrator-actor",
		"active",
		RoleAdministrator,
	)

	securityTargetID := f.addStaff(
		"protected-security-target",
		"active",
		RoleSecurity,
	)

	_, err := f.service.ResetStaffPassword(
		context.Background(),
		securityTargetID,
		"EneDei!Blocked-Recovery-2026",
		f.metadata(administratorID),
	)
	if !errors.Is(err, ErrAdminProtectedStaffMutation) {
		t.Fatalf(
			"Administrator protected password reset error = %v, want %v",
			err,
			ErrAdminProtectedStaffMutation,
		)
	}

	_, err = f.service.ResetStaffAdminMFA(
		context.Background(),
		securityTargetID,
		f.metadata(administratorID),
	)
	if !errors.Is(err, ErrAdminProtectedStaffMutation) {
		t.Fatalf(
			"Administrator protected MFA reset error = %v, want %v",
			err,
			ErrAdminProtectedStaffMutation,
		)
	}
}

func TestSuperAdminBreakGlassDB(
	t *testing.T,
) {
	f := newAdminRecoveryDBFixture(t)

	lastSuperAdminID := f.addStaff(
		"last-super-admin",
		"active",
		RoleSuperAdmin,
	)
	f.seedSecurityState(lastSuperAdminID)
	seedAdminRecoveryMFA(t, f.db, lastSuperAdminID)

	const recoveredPassword = "EneDei!Break-Glass-2026"

	result, err := RecoverLastSuperAdmin(
		context.Background(),
		f.db,
		SuperAdminBreakGlassInput{
			Identifier:  lastSuperAdminID,
			NewPassword: recoveredPassword,
			ResetMFA:    true,
			Operator:    "integration-test-operator",
		},
	)
	if err != nil {
		t.Fatalf("break-glass recovery: %v", err)
	}

	if result.StaffAccountID != lastSuperAdminID || !result.MFAReset {
		t.Fatalf("unexpected break-glass result: %+v", result)
	}

	assertAdminRecoveryPassword(t, f.db, lastSuperAdminID, recoveredPassword)
	assertAdminRecoverySessionsRevoked(
		t,
		f.db,
		lastSuperAdminID,
		"super_admin_break_glass_recovery",
	)
	assertAdminRecoveryMFAState(t, f.db, lastSuperAdminID, "disabled", 0)

	var breakGlassAuditCount int
	if err := f.db.QueryRow(
		context.Background(),
		`
			SELECT COUNT(*)::integer
			FROM admin_security_events
			WHERE
				staff_account_id = $1::uuid
				AND event_type = $2
				AND outcome = 'success'
				AND details ->> 'operator' = 'integration-test-operator'
				AND details ->> 'source' = 'offline_break_glass_cli'
				AND details ->> 'target_staff_account_id' = $1::text
		`,
		lastSuperAdminID,
		adminEventSuperAdminBreakGlassRecovery,
	).Scan(&breakGlassAuditCount); err != nil {
		t.Fatalf("count break-glass audit event: %v", err)
	}
	if breakGlassAuditCount != 1 {
		t.Fatalf("expected one break-glass audit event, got %d", breakGlassAuditCount)
	}

	otherSuperAdminID := f.addStaff(
		"other-active-super-admin",
		"active",
		RoleSuperAdmin,
	)
	_ = otherSuperAdminID

	_, err = RecoverLastSuperAdmin(
		context.Background(),
		f.db,
		SuperAdminBreakGlassInput{
			Identifier:  lastSuperAdminID,
			NewPassword: "EneDei!Must-Refuse-2026",
			ResetMFA:    false,
			Operator:    "integration-test-operator",
		},
	)
	if !errors.Is(err, ErrAdminBreakGlassOtherActiveSuperAdmin) {
		t.Fatalf(
			"break-glass with another active Super Admin error = %v, want %v",
			err,
			ErrAdminBreakGlassOtherActiveSuperAdmin,
		)
	}

	assertAdminRecoveryPassword(t, f.db, lastSuperAdminID, recoveredPassword)
}

func newAdminRecoveryDBFixture(
	t *testing.T,
) *adminBanDBFixture {
	t.Helper()

	db := openAdminRecoveryTestDB(t)

	f := &adminBanDBFixture{
		t:       t,
		db:      db,
		service: NewService(db),
	}

	f.ensureRoles()

	t.Cleanup(f.cleanup)

	return f
}

func openAdminRecoveryTestDB(
	t *testing.T,
) *pgxpool.Pool {
	t.Helper()

	if os.Getenv(runAdminRecoveryDBTestsEnv) != "1" {
		t.Skipf(
			"set %s=1 and %s to a dedicated empty PostgreSQL test database",
			runAdminRecoveryDBTestsEnv,
			adminRecoveryTestDBURLEnv,
		)
	}

	rawURL := strings.TrimSpace(os.Getenv(adminRecoveryTestDBURLEnv))
	if rawURL == "" {
		t.Fatalf(
			"%s is required when %s=1",
			adminRecoveryTestDBURLEnv,
			runAdminRecoveryDBTestsEnv,
		)
	}

	cfg, databaseName := adminBanTestConfigFromURL(t, rawURL)

	if !strings.Contains(strings.ToLower(databaseName), "test") {
		t.Fatalf(
			"refusing integration test against database %q: name must contain 'test'",
			databaseName,
		)
	}

	migrationCtx, migrationCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer migrationCancel()

	if _, err := platformdatabase.ApplyMigrations(migrationCtx, cfg); err != nil {
		t.Fatalf("apply migrations to dedicated Admin-recovery test database: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(rawURL)
	if err != nil {
		t.Fatalf("parse Admin-recovery test database URL: %v", err)
	}

	poolConfig.MaxConns = 4
	poolConfig.MinConns = 0

	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("open Admin-recovery test database: %v", err)
	}

	t.Cleanup(db.Close)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	if err := db.Ping(pingCtx); err != nil {
		t.Fatalf("ping Admin-recovery test database: %v", err)
	}

	var staffCount int
	if err := db.QueryRow(
		context.Background(),
		`SELECT COUNT(*)::integer FROM staff_accounts`,
	).Scan(&staffCount); err != nil {
		t.Fatalf("count existing Admin-recovery test staff: %v", err)
	}
	if staffCount != 0 {
		t.Fatalf(
			"refusing Admin-recovery integration test: dedicated database %q already contains %d staff account(s)",
			databaseName,
			staffCount,
		)
	}

	return db
}

func seedAdminRecoveryMFA(
	t *testing.T,
	db *pgxpool.Pool,
	staffID string,
) {
	t.Helper()

	if _, err := db.Exec(
		context.Background(),
		`
			INSERT INTO admin_totp_credentials (
				staff_account_id,
				label,
				secret_ciphertext,
				encryption_key_id,
				algorithm,
				digits,
				period_seconds,
				status,
				verified_at,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				'Integration Test Authenticator',
				'integration-test-ciphertext',
				'integration-test-key',
				'SHA1',
				6,
				30,
				'active',
				now(),
				now(),
				now()
			)
		`,
		staffID,
	); err != nil {
		t.Fatalf("seed active Admin MFA credential: %v", err)
	}

	if _, err := db.Exec(
		context.Background(),
		`
			INSERT INTO admin_mfa_recovery_codes (
				staff_account_id,
				code_hash,
				created_at
			)
			VALUES ($1::uuid, $2, now())
		`,
		staffID,
		testHash(staffID+"-recovery-code"),
	); err != nil {
		t.Fatalf("seed Admin MFA recovery code: %v", err)
	}
}

func assertAdminRecoveryPassword(
	t *testing.T,
	db *pgxpool.Pool,
	staffID string,
	password string,
) {
	t.Helper()

	var passwordHash string
	if err := db.QueryRow(
		context.Background(),
		`SELECT password_hash FROM staff_accounts WHERE id = $1::uuid`,
		staffID,
	).Scan(&passwordHash); err != nil {
		t.Fatalf("load recovered Admin password hash: %v", err)
	}

	valid, err := platformsecurity.VerifyPassword(password, passwordHash)
	if err != nil {
		t.Fatalf("verify recovered Admin password: %v", err)
	}
	if !valid {
		t.Fatal("recovered Admin password does not verify against persisted hash")
	}
}

func assertAdminRecoverySessionsRevoked(
	t *testing.T,
	db *pgxpool.Pool,
	staffID string,
	expectedReason string,
) {
	t.Helper()

	for _, check := range []struct {
		name  string
		query string
	}{
		{
			name: "staff sessions",
			query: `
				SELECT COUNT(*)::integer
				FROM staff_sessions
				WHERE staff_account_id = $1::uuid AND revoked_at IS NULL
			`,
		},
		{
			name: "Admin sessions",
			query: `
				SELECT COUNT(*)::integer
				FROM admin_sessions
				WHERE staff_account_id = $1::uuid AND revoked_at IS NULL
			`,
		},
		{
			name: "pending Admin login challenges",
			query: `
				SELECT COUNT(*)::integer
				FROM admin_login_challenges
				WHERE staff_account_id = $1::uuid AND status = 'pending'
			`,
		},
	} {
		var count int
		if err := db.QueryRow(context.Background(), check.query, staffID).Scan(&count); err != nil {
			t.Fatalf("count active %s: %v", check.name, err)
		}
		if count != 0 {
			t.Fatalf("expected zero active %s, got %d", check.name, count)
		}
	}

	var wrongReasonCount int
	if err := db.QueryRow(
		context.Background(),
		`
			SELECT COUNT(*)::integer
			FROM admin_sessions
			WHERE
				staff_account_id = $1::uuid
				AND revoke_reason IS DISTINCT FROM $2
		`,
		staffID,
		expectedReason,
	).Scan(&wrongReasonCount); err != nil {
		t.Fatalf("verify Admin session revoke reason: %v", err)
	}
	if wrongReasonCount != 0 {
		t.Fatalf(
			"expected all Admin sessions to use revoke reason %q; mismatches=%d",
			expectedReason,
			wrongReasonCount,
		)
	}
}

func assertAdminRecoveryMFAState(
	t *testing.T,
	db *pgxpool.Pool,
	staffID string,
	expectedStatus string,
	expectedRecoveryCodes int,
) {
	t.Helper()

	var status string
	if err := db.QueryRow(
		context.Background(),
		`SELECT status FROM admin_totp_credentials WHERE staff_account_id = $1::uuid`,
		staffID,
	).Scan(&status); err != nil {
		t.Fatalf("load Admin MFA status: %v", err)
	}
	if status != expectedStatus {
		t.Fatalf("expected Admin MFA status %q, got %q", expectedStatus, status)
	}

	var recoveryCodes int
	if err := db.QueryRow(
		context.Background(),
		`SELECT COUNT(*)::integer FROM admin_mfa_recovery_codes WHERE staff_account_id = $1::uuid`,
		staffID,
	).Scan(&recoveryCodes); err != nil {
		t.Fatalf("count Admin MFA recovery codes: %v", err)
	}
	if recoveryCodes != expectedRecoveryCodes {
		t.Fatalf(
			"expected %d Admin MFA recovery code(s), got %d",
			expectedRecoveryCodes,
			recoveryCodes,
		)
	}
}

func assertAdminRecoveryAudit(
	t *testing.T,
	db *pgxpool.Pool,
	actorID string,
	eventType string,
	targetID string,
) {
	t.Helper()

	var count int
	if err := db.QueryRow(
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
		t.Fatalf("count Admin recovery audit event: %v", err)
	}
	if count != 1 {
		t.Fatalf(
			"expected exactly one %s audit event for target %s, got %d",
			eventType,
			targetID,
			count,
		)
	}
}

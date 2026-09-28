package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	platformconfig "project.local/commerce-api/internal/platform/config"
	platformdatabase "project.local/commerce-api/internal/platform/database"
)

const (
	runAdminBanDBTestsEnv = "RUN_ADMIN_BAN_DB_TEST"
	adminBanTestDBURLEnv  = "ADMIN_BAN_TEST_DATABASE_URL"
)

type adminBanDBFixture struct {
	t       *testing.T
	db      *pgxpool.Pool
	service *Service

	staffIDs []string
}

func TestStaffBanDBLifecycleAndSecurityRevocation(
	t *testing.T,
) {
	f :=
		newAdminBanDBFixture(
			t,
		)

	actorID :=
		f.addStaff(
			"super-actor",
			"active",
			RoleSuperAdmin,
		)

	targetID :=
		f.addStaff(
			"ordinary-target",
			"active",
			RoleCatalog,
		)

	f.seedSecurityState(
		targetID,
	)

	metadata :=
		f.metadata(
			actorID,
		)

	ban, err :=
		f.service.CreateStaffBan(
			context.Background(),
			targetID,
			CreateAccountBanInput{
				Scope: AccountBanScopeFullAccount,

				BanType: AccountBanTypePermanent,

				Reason: "integration test permanent staff ban",
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"create staff ban: %v",
			err,
		)
	}

	if ban.TargetType !=
		AccountBanTargetStaff ||
		ban.StaffAccountID !=
			targetID {

		t.Fatalf(
			"unexpected ban target: %+v",
			ban,
		)
	}

	if ban.Scope !=
		AccountBanScopeFullAccount ||
		ban.PreviousAccountStatus !=
			"active" ||
		!ban.Active {

		t.Fatalf(
			"unexpected created ban state: %+v",
			ban,
		)
	}

	f.assertStaffStatus(
		targetID,
		"banned",
	)

	f.assertSecurityRevoked(
		targetID,
	)

	listed, err :=
		f.service.ListStaffBans(
			context.Background(),
			targetID,
		)
	if err != nil {
		t.Fatalf(
			"list staff bans: %v",
			err,
		)
	}

	if len(
		listed,
	) != 1 ||
		listed[0].ID !=
			ban.ID {

		t.Fatalf(
			"expected exactly created ban in history, got %+v",
			listed,
		)
	}

	_, err =
		f.service.CreateStaffBan(
			context.Background(),
			targetID,
			CreateAccountBanInput{
				Scope: AccountBanScopeFullAccount,

				BanType: AccountBanTypePermanent,

				Reason: "duplicate integration test ban",
			},
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminBanConflict,
		"duplicate active ban",
	)

	_, err =
		f.service.SetStaffStatus(
			context.Background(),
			targetID,
			"active",
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminStaffBanManagedSeparately,
		"status mutation while banned",
	)

	_, err =
		f.service.ReplaceStaffRoles(
			context.Background(),
			targetID,
			[]string{
				RoleCatalog,
			},
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminStaffBanManagedSeparately,
		"role mutation while banned",
	)

	_, err =
		f.service.ResetStaffPassword(
			context.Background(),
			targetID,
			"AdminBanDB!Passw0rd-2026-Integration",
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminStaffBanManagedSeparately,
		"password reset while banned",
	)

	_, err =
		f.service.ResetStaffAdminMFA(
			context.Background(),
			targetID,
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminStaffBanManagedSeparately,
		"MFA reset while banned",
	)

	_, err =
		f.service.DeleteStaff(
			context.Background(),
			targetID,
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminStaffBanManagedSeparately,
		"delete while banned",
	)

	f.assertAuditEvent(
		actorID,
		"staff_ban_created",
		targetID,
	)

	revoked, err :=
		f.service.RevokeStaffBan(
			context.Background(),
			targetID,
			ban.ID,
			"integration test unban",
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"revoke staff ban: %v",
			err,
		)
	}

	if revoked.Active ||
		revoked.RevokedAt == nil ||
		revoked.RevokedByStaffID !=
			actorID {

		t.Fatalf(
			"unexpected revoked ban state: %+v",
			revoked,
		)
	}

	f.assertStaffStatus(
		targetID,
		"active",
	)

	/*
		Unban restores lifecycle state.

		It must not resurrect previously revoked sessions
		or cancelled login challenges.
	*/
	f.assertSecurityRevoked(
		targetID,
	)

	f.assertAuditEvent(
		actorID,
		"staff_ban_revoked",
		targetID,
	)

	_, err =
		f.service.RevokeStaffBan(
			context.Background(),
			targetID,
			ban.ID,
			"second unban attempt",
			metadata,
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminBanAlreadyRevoked,
		"duplicate unban",
	)
}

func TestStaffBanDBRestoresPreviousLifecycleState(
	t *testing.T,
) {
	f :=
		newAdminBanDBFixture(
			t,
		)

	actorID :=
		f.addStaff(
			"restore-super-actor",
			"active",
			RoleSuperAdmin,
		)

	metadata :=
		f.metadata(
			actorID,
		)

	for _, previousStatus := range []string{
		"suspended",
		"disabled",
	} {

		previousStatus :=
			previousStatus

		t.Run(
			previousStatus,
			func(
				t *testing.T,
			) {
				targetID :=
					f.addStaff(
						"restore-"+previousStatus,
						previousStatus,
						RoleCatalog,
					)

				ban, err :=
					f.service.CreateStaffBan(
						context.Background(),
						targetID,
						CreateAccountBanInput{
							Scope: AccountBanScopeFullAccount,

							BanType: AccountBanTypePermanent,

							Reason: "verify exact lifecycle restoration",
						},
						metadata,
					)
				if err != nil {
					t.Fatalf(
						"ban %s staff: %v",
						previousStatus,
						err,
					)
				}

				if ban.PreviousAccountStatus !=
					previousStatus {

					t.Fatalf(
						"expected previous status %q, got %q",
						previousStatus,
						ban.PreviousAccountStatus,
					)
				}

				f.assertStaffStatus(
					targetID,
					"banned",
				)

				_, err =
					f.service.RevokeStaffBan(
						context.Background(),
						targetID,
						ban.ID,
						"restore prior lifecycle state",
						metadata,
					)
				if err != nil {
					t.Fatalf(
						"unban %s staff: %v",
						previousStatus,
						err,
					)
				}

				f.assertStaffStatus(
					targetID,
					previousStatus,
				)
			},
		)
	}
}

func TestStaffBanDBProtectedRoleBoundary(
	t *testing.T,
) {
	f :=
		newAdminBanDBFixture(
			t,
		)

	administratorID :=
		f.addStaff(
			"administrator-actor",
			"active",
			RoleAdministrator,
		)

	superAdminID :=
		f.addStaff(
			"super-actor",
			"active",
			RoleSuperAdmin,
		)

	securityTargetID :=
		f.addStaff(
			"security-target",
			"active",
			RoleSecurity,
		)

	_, err :=
		f.service.CreateStaffBan(
			context.Background(),
			securityTargetID,
			CreateAccountBanInput{
				Scope: AccountBanScopeFullAccount,

				BanType: AccountBanTypePermanent,

				Reason: "administrator must not ban protected staff",
			},
			f.metadata(
				administratorID,
			),
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminProtectedStaffMutation,
		"Administrator banning protected staff",
	)

	f.assertStaffStatus(
		securityTargetID,
		"active",
	)

	metadata :=
		f.metadata(
			superAdminID,
		)

	ban, err :=
		f.service.CreateStaffBan(
			context.Background(),
			securityTargetID,
			CreateAccountBanInput{
				Scope: AccountBanScopeFullAccount,

				BanType: AccountBanTypePermanent,

				Reason: "Super Admin authorized protected-staff ban",
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"Super Admin ban protected staff: %v",
			err,
		)
	}

	f.assertStaffStatus(
		securityTargetID,
		"banned",
	)

	_, err =
		f.service.RevokeStaffBan(
			context.Background(),
			securityTargetID,
			ban.ID,
			"Super Admin protected-staff unban",
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"Super Admin unban protected staff: %v",
			err,
		)
	}

	f.assertStaffStatus(
		securityTargetID,
		"active",
	)
}

func TestStaffBanDBLastActiveSuperAdminGuard(
	t *testing.T,
) {
	f :=
		newAdminBanDBFixture(
			t,
		)

	/*
		The HTTP authentication layer would reject this disabled actor.

		Keeping the Super Admin role while disabling the account lets
		this direct service test exercise the service-level
		last-active-Super-Admin defense in depth.
	*/
	disabledSuperActorID :=
		f.addStaff(
			"disabled-super-actor",
			"disabled",
			RoleSuperAdmin,
		)

	lastActiveSuperID :=
		f.addStaff(
			"last-active-super",
			"active",
			RoleSuperAdmin,
		)

	_, err :=
		f.service.CreateStaffBan(
			context.Background(),
			lastActiveSuperID,
			CreateAccountBanInput{
				Scope: AccountBanScopeFullAccount,

				BanType: AccountBanTypePermanent,

				Reason: "must preserve one active Super Admin",
			},
			f.metadata(
				disabledSuperActorID,
			),
		)

	assertAdminBanDBErrorIs(
		t,
		err,
		ErrAdminLastSuperAdmin,
		"last active Super Admin ban",
	)

	f.assertStaffStatus(
		lastActiveSuperID,
		"active",
	)

	count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM account_bans
				WHERE staff_account_id = $1::uuid
			`,
			lastActiveSuperID,
		)

	if count != 0 {
		t.Fatalf(
			"expected no ban row for rejected last-Super-Admin ban, got %d",
			count,
		)
	}
}

func newAdminBanDBFixture(
	t *testing.T,
) *adminBanDBFixture {
	t.Helper()

	db :=
		openAdminBanTestDB(
			t,
		)

	f :=
		&adminBanDBFixture{
			t: t,

			db: db,

			service: NewService(
				db,
			),
		}

	f.ensureRoles()

	t.Cleanup(
		f.cleanup,
	)

	return f
}

func openAdminBanTestDB(
	t *testing.T,
) *pgxpool.Pool {
	t.Helper()

	if os.Getenv(
		runAdminBanDBTestsEnv,
	) != "1" {

		t.Skipf(
			"set %s=1 and %s to a dedicated empty PostgreSQL test database",
			runAdminBanDBTestsEnv,
			adminBanTestDBURLEnv,
		)
	}

	rawURL :=
		strings.TrimSpace(
			os.Getenv(
				adminBanTestDBURLEnv,
			),
		)

	if rawURL == "" {
		t.Fatalf(
			"%s is required when %s=1",
			adminBanTestDBURLEnv,
			runAdminBanDBTestsEnv,
		)
	}

	cfg, databaseName :=
		adminBanTestConfigFromURL(
			t,
			rawURL,
		)

	/*
		Hard safety guard.

		This integration test must never run against the normal
		development or production database.
	*/
	if !strings.Contains(
		strings.ToLower(
			databaseName,
		),
		"test",
	) {
		t.Fatalf(
			"refusing integration test against database %q: name must contain 'test'",
			databaseName,
		)
	}

	migrationCtx, migrationCancel :=
		context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
	defer migrationCancel()

	/*
		Use the application's real embedded Atlas migration runner.

		The dedicated DB therefore exercises exactly the same schema
		the application starts with.
	*/
	if _, err :=
		platformdatabase.ApplyMigrations(
			migrationCtx,
			cfg,
		); err != nil {

		t.Fatalf(
			"apply migrations to dedicated Admin-ban test database: %v",
			err,
		)
	}

	poolConfig, err :=
		pgxpool.ParseConfig(
			rawURL,
		)
	if err != nil {
		t.Fatalf(
			"parse Admin-ban test database URL: %v",
			err,
		)
	}

	poolConfig.MaxConns = 4
	poolConfig.MinConns = 0

	db, err :=
		pgxpool.NewWithConfig(
			context.Background(),
			poolConfig,
		)
	if err != nil {
		t.Fatalf(
			"open Admin-ban test database: %v",
			err,
		)
	}

	t.Cleanup(
		db.Close,
	)

	pingCtx, pingCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
	defer pingCancel()

	if err :=
		db.Ping(
			pingCtx,
		); err != nil {

		t.Fatalf(
			"ping Admin-ban test database: %v",
			err,
		)
	}

	var hasBanTable bool

	if err :=
		db.QueryRow(
			context.Background(),
			`
				SELECT
					to_regclass(
						'public.account_bans'
					) IS NOT NULL
			`,
		).Scan(
			&hasBanTable,
		); err != nil {

		t.Fatalf(
			"verify account_bans table: %v",
			err,
		)
	}

	if !hasBanTable {
		t.Fatal(
			"dedicated test database is missing public.account_bans after migrations",
		)
	}

	/*
		The last-Super-Admin test requires deterministic isolation.

		Refuse to run if this is pointed at a test database which
		already contains staff data.
	*/
	var staffCount int

	if err :=
		db.QueryRow(
			context.Background(),
			`
				SELECT COUNT(*)::integer
				FROM staff_accounts
			`,
		).Scan(
			&staffCount,
		); err != nil {

		t.Fatalf(
			"count existing test staff: %v",
			err,
		)
	}

	if staffCount != 0 {
		t.Fatalf(
			"refusing Admin-ban integration test: dedicated database %q already contains %d staff account(s)",
			databaseName,
			staffCount,
		)
	}

	return db
}

func adminBanTestConfigFromURL(
	t *testing.T,
	rawURL string,
) (
	platformconfig.Config,
	string,
) {
	t.Helper()

	parsed, err :=
		url.Parse(
			rawURL,
		)
	if err != nil {
		t.Fatalf(
			"parse %s: %v",
			adminBanTestDBURLEnv,
			err,
		)
	}

	if parsed.Scheme !=
		"postgres" &&
		parsed.Scheme !=
			"postgresql" {

		t.Fatalf(
			"%s must use postgres:// or postgresql://",
			adminBanTestDBURLEnv,
		)
	}

	if parsed.User == nil {
		t.Fatalf(
			"%s must include a database user",
			adminBanTestDBURLEnv,
		)
	}

	password, _ :=
		parsed.User.Password()

	port :=
		uint16(
			5432,
		)

	if value :=
		parsed.Port(); value != "" {

		parsedPort, parseErr :=
			strconv.ParseUint(
				value,
				10,
				16,
			)
		if parseErr != nil {
			t.Fatalf(
				"invalid PostgreSQL port %q: %v",
				value,
				parseErr,
			)
		}

		port =
			uint16(
				parsedPort,
			)
	}

	databaseName, err :=
		url.PathUnescape(
			strings.TrimPrefix(
				parsed.Path,
				"/",
			),
		)
	if err != nil ||
		databaseName == "" {

		t.Fatalf(
			"invalid test database name in %s",
			adminBanTestDBURLEnv,
		)
	}

	sslMode :=
		strings.TrimSpace(
			parsed.Query().
				Get(
					"sslmode",
				),
		)

	if sslMode == "" {
		sslMode =
			"disable"
	}

	return platformconfig.Config{
			PostgresHost: parsed.Hostname(),

			PostgresPort: port,

			PostgresDB: databaseName,

			PostgresUser: parsed.User.Username(),

			PostgresPassword: password,

			PostgresSSLMode: sslMode,
		},
		databaseName
}

func (f *adminBanDBFixture) ensureRoles() {
	f.t.Helper()

	roles :=
		[]struct {
			code string
			name string
		}{
			{
				RoleSuperAdmin,
				"Super Admin",
			},
			{
				RoleAdministrator,
				"Administrator",
			},
			{
				RoleSecurity,
				"Security",
			},
			{
				RoleCatalog,
				"Catalog",
			},
		}

	for _, role := range roles {

		f.exec(
			`
				INSERT INTO staff_roles (
					code,
					name,
					description,
					is_system_role,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					'Admin ban DB integration test role',
					true,
					now(),
					now()
				)
				ON CONFLICT (code)
				DO NOTHING
			`,
			role.code,
			role.name,
		)
	}
}

func (f *adminBanDBFixture) addStaff(
	label string,
	status string,
	roleCodes ...string,
) string {
	f.t.Helper()

	tag :=
		testHash(
			fmt.Sprintf(
				"%s-%d-%d",
				label,
				time.Now().
					UnixNano(),
				len(
					f.staffIDs,
				),
			),
		)

	staffCode :=
		"TST-" +
			strings.ToUpper(
				tag[:12],
			)

	email :=
		tag +
			"@admin-ban.integration.test"

	var staffID string

	if err :=
		f.db.QueryRow(
			context.Background(),
			`
				INSERT INTO staff_accounts (
					staff_code,
					full_name,
					email,
					phone,
					password_hash,
					status,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					NULL,
					'integration-test-password-hash',
					$4,
					now(),
					now()
				)
				RETURNING id::text
			`,
			staffCode,
			"Admin Ban DB "+label,
			email,
			status,
		).Scan(
			&staffID,
		); err != nil {

		f.t.Fatalf(
			"insert test staff %s: %v",
			label,
			err,
		)
	}

	f.staffIDs =
		append(
			f.staffIDs,
			staffID,
		)

	for _, roleCode := range roleCodes {

		tag, err :=
			f.db.Exec(
				context.Background(),
				`
					INSERT INTO staff_account_roles (
						staff_account_id,
						role_id,
						assigned_by_staff_id,
						created_at
					)
					SELECT
						$1::uuid,
						id,
						NULL,
						now()
					FROM staff_roles
					WHERE code = $2
					ON CONFLICT DO NOTHING
				`,
				staffID,
				roleCode,
			)

		if err != nil {
			f.t.Fatalf(
				"assign test role %s to %s: %v",
				roleCode,
				label,
				err,
			)
		}

		if tag.RowsAffected() !=
			1 {

			f.t.Fatalf(
				"test role %s was not assigned to %s",
				roleCode,
				label,
			)
		}
	}

	return staffID
}

func (f *adminBanDBFixture) seedSecurityState(
	staffID string,
) {
	f.t.Helper()

	seed :=
		testHash(
			staffID +
				"-security",
		)

	now :=
		time.Now().
			UTC()

	accessExpiry :=
		now.Add(
			15 * time.Minute,
		)

	refreshExpiry :=
		now.Add(
			2 * time.Hour,
		)

	f.exec(
		`
			INSERT INTO staff_sessions (
				staff_account_id,
				access_token_hash,
				refresh_token_hash,
				access_expires_at,
				refresh_expires_at,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				$5,
				now(),
				now()
			)
		`,
		staffID,
		testHash(
			seed+
				"-staff-access",
		),
		testHash(
			seed+
				"-staff-refresh",
		),
		accessExpiry,
		refreshExpiry,
	)

	var challengeID string

	if err :=
		f.db.QueryRow(
			context.Background(),
			`
				INSERT INTO admin_login_challenges (
					staff_account_id,
					challenge_token_hash,
					purpose,
					status,
					failed_attempts,
					max_attempts,
					expires_at,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					'login',
					'pending',
					0,
					5,
					$3,
					now(),
					now()
				)
				RETURNING id::text
			`,
			staffID,
			testHash(
				seed+
					"-challenge",
			),
			now.Add(
				10*time.Minute,
			),
		).Scan(
			&challengeID,
		); err != nil {

		f.t.Fatalf(
			"insert test Admin login challenge: %v",
			err,
		)
	}

	f.exec(
		`
			INSERT INTO admin_sessions (
				staff_account_id,
				login_challenge_id,
				access_token_hash,
				refresh_token_hash,
				csrf_token_hash,
				refresh_generation,
				access_expires_at,
				refresh_expires_at,
				authenticated_at,
				mfa_verified_at,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3,
				$4,
				$5,
				0,
				$6,
				$7,
				$8,
				$8,
				now(),
				now()
			)
		`,
		staffID,
		challengeID,
		testHash(
			seed+
				"-admin-access",
		),
		testHash(
			seed+
				"-admin-refresh",
		),
		testHash(
			seed+
				"-admin-csrf",
		),
		accessExpiry,
		refreshExpiry,
		now,
	)
}

func (f *adminBanDBFixture) metadata(
	actorStaffID string,
) AdminActionMetadata {
	return AdminActionMetadata{
		StaffAccountID: actorStaffID,

		RequestID: "admin-ban-db-" +
			testHash(
				fmt.Sprintf(
					"%s-%d",
					actorStaffID,
					time.Now().
						UnixNano(),
				),
			)[:20],

		IPAddress: "127.0.0.1",

		UserAgent: "admin-ban-db-integration-test",
	}
}

func (f *adminBanDBFixture) assertStaffStatus(
	staffID string,
	expected string,
) {
	f.t.Helper()

	var actual string

	if err :=
		f.db.QueryRow(
			context.Background(),
			`
				SELECT status
				FROM staff_accounts
				WHERE id = $1::uuid
			`,
			staffID,
		).Scan(
			&actual,
		); err != nil {

		f.t.Fatalf(
			"load staff status: %v",
			err,
		)
	}

	if actual !=
		expected {

		f.t.Fatalf(
			"expected staff status %q, got %q",
			expected,
			actual,
		)
	}
}

func (f *adminBanDBFixture) assertSecurityRevoked(
	staffID string,
) {
	f.t.Helper()

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM staff_sessions
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffID,
		); count != 0 {

		f.t.Fatalf(
			"expected zero active staff sessions, got %d",
			count,
		)
	}

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_sessions
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffID,
		); count != 0 {

		f.t.Fatalf(
			"expected zero active Admin sessions, got %d",
			count,
		)
	}

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_sessions
				WHERE
					staff_account_id = $1::uuid
					AND revoke_reason
						IS DISTINCT FROM
						'staff_account_banned'
			`,
			staffID,
		); count != 0 {

		f.t.Fatalf(
			"expected all Admin sessions to use staff_account_banned revoke reason; mismatches=%d",
			count,
		)
	}

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_login_challenges
				WHERE
					staff_account_id = $1::uuid
					AND status = 'pending'
			`,
			staffID,
		); count != 0 {

		f.t.Fatalf(
			"expected zero pending Admin login challenges, got %d",
			count,
		)
	}

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_login_challenges
				WHERE
					staff_account_id = $1::uuid
					AND status = 'cancelled'
			`,
			staffID,
		); count < 1 {

		f.t.Fatal(
			"expected at least one cancelled Admin login challenge",
		)
	}
}

func (f *adminBanDBFixture) assertAuditEvent(
	actorID string,
	eventType string,
	targetID string,
) {
	f.t.Helper()

	count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_security_events
				WHERE
					staff_account_id = $1::uuid
					AND event_type = $2
					AND details ->> 'target_id' = $3
			`,
			actorID,
			eventType,
			targetID,
		)

	if count != 1 {
		f.t.Fatalf(
			"expected exactly one %s audit event for target %s, got %d",
			eventType,
			targetID,
			count,
		)
	}
}

func (f *adminBanDBFixture) count(
	query string,
	args ...any,
) int {
	f.t.Helper()

	var value int

	if err :=
		f.db.QueryRow(
			context.Background(),
			query,
			args...,
		).Scan(
			&value,
		); err != nil {

		f.t.Fatalf(
			"count query failed: %v",
			err,
		)
	}

	return value
}

func (f *adminBanDBFixture) exec(
	query string,
	args ...any,
) {
	f.t.Helper()

	if _, err :=
		f.db.Exec(
			context.Background(),
			query,
			args...,
		); err != nil {

		f.t.Fatalf(
			"fixture SQL failed: %v",
			err,
		)
	}
}

func (f *adminBanDBFixture) cleanup() {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			15*time.Second,
		)
	defer cancel()

	for _, id := range f.staffIDs {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM admin_security_events
					WHERE
						staff_account_id = $1::uuid
						OR details ->> 'target_id' = $1::text
						OR details ->> 'staff_account_id' = $1::text
				`,
				id,
			)
	}

	/*
		account_bans has references to staff identities,
		so ban rows must be removed before fixture staff rows.
	*/
	for _, id := range f.staffIDs {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM account_bans
					WHERE
						staff_account_id = $1::uuid
						OR issued_by_staff_id = $1::uuid
						OR revoked_by_staff_id = $1::uuid
				`,
				id,
			)
	}

	for i :=
		len(
			f.staffIDs,
		) - 1; i >= 0; i-- {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM staff_accounts
					WHERE id = $1::uuid
				`,
				f.staffIDs[i],
			)
	}
}

func testHash(
	value string,
) string {
	sum :=
		sha256.Sum256(
			[]byte(
				value,
			),
		)

	return hex.EncodeToString(
		sum[:],
	)
}

func assertAdminBanDBErrorIs(
	t *testing.T,
	actual error,
	expected error,
	label string,
) {
	t.Helper()

	if !errors.Is(
		actual,
		expected,
	) {
		t.Fatalf(
			"%s: expected error %v, got %v",
			label,
			expected,
			actual,
		)
	}
}

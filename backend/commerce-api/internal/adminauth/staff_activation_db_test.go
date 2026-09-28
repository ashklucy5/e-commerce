package adminauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	platformconfig "project.local/commerce-api/internal/platform/config"
	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	runStaffActivationDBTestsEnv = "RUN_ADMINAUTH_ACTIVATION_DB_TEST"

	staffActivationTestDBURLEnv = "ADMINAUTH_ACTIVATION_TEST_DATABASE_URL"

	staffActivationTestRedisURLEnv = "ADMINAUTH_ACTIVATION_TEST_REDIS_URL"

	staffActivationTestRoleCode = "activation_test_admin"

	staffActivationPanelPermission = "admin.panel.access"
)

type staffActivationDBFixture struct {
	t *testing.T

	db    *pgxpool.Pool
	redis *redis.Client

	repository *Repository
	service    *Service

	now time.Time

	ipAddress string

	staffIDs []string
}

type staffActivationTestStaff struct {
	ID string

	Email string
}

func TestStaffActivationDBLifecycle(
	t *testing.T,
) {
	f :=
		newStaffActivationDBFixture(
			t,
		)

	ctx :=
		context.Background()

	creator :=
		f.addStaff(
			"creator",
			"active",
			false,
		)

	target :=
		f.addStaff(
			"target",
			"pending_activation",
			true,
		)

	activationToken :=
		f.addInvitation(
			target,
			creator.ID,
			f.now.Add(
				-time.Minute,
			),
			f.now.Add(
				time.Hour,
			),
		)

	metadata :=
		f.metadata()

	const newPassword = "ActivationDB!NewPassw0rd-2026"

	/*
		A pending account must not be able to use the normal Admin
		login path, even when the supplied value is the password that
		will later be chosen during activation.
	*/
	_, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: target.Email,

				Password: newPassword,
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrInvalidCredentials,
		"normal login before activation",
	)

	f.assertStaffStatus(
		target.ID,
		"pending_activation",
	)

	/*
		The activation token is bound to the invitation email.

		Neither the right token with another email nor the right
		email with another token may authorize password setup.
	*/
	_, err =
		f.service.SetActivationPassword(
			ctx,
			StaffActivationPasswordRequest{
				Email: "wrong-" +
					target.Email,

				ActivationToken: activationToken,

				Password: newPassword,
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrInvalidStaffActivation,
		"activation with wrong email",
	)

	_, err =
		f.service.SetActivationPassword(
			ctx,
			StaffActivationPasswordRequest{
				Email: target.Email,

				ActivationToken: activationToken +
					"-wrong",

				Password: newPassword,
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrInvalidStaffActivation,
		"activation with wrong token",
	)

	/*
		MFA enrollment cannot begin until the staff member has
		chosen their own password.
	*/
	_, err =
		f.service.BeginActivationMFA(
			ctx,
			StaffActivationMFAEnrollRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Label: "Integration test authenticator",
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrStaffActivationPasswordNotSet,
		"MFA enrollment before password setup",
	)

	/*
		Step 1: the staff member chooses their own password.
	*/
	passwordResult, err :=
		f.service.SetActivationPassword(
			ctx,
			StaffActivationPasswordRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Password: newPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"set activation password: %v",
			err,
		)
	}

	if passwordResult.StaffAccountID !=
		target.ID {

		t.Fatalf(
			"unexpected activation password staff ID: got %q want %q",
			passwordResult.StaffAccountID,
			target.ID,
		)
	}

	if !strings.EqualFold(
		passwordResult.Email,
		target.Email,
	) {
		t.Fatalf(
			"unexpected activation password email: got %q want %q",
			passwordResult.Email,
			target.Email,
		)
	}

	if passwordResult.Next !=
		"mfa_enroll" {

		t.Fatalf(
			"expected next=mfa_enroll, got %q",
			passwordResult.Next,
		)
	}

	/*
		Setting a password must not make the account active.

		The account remains pending until MFA has been successfully
		verified.
	*/
	f.assertStaffStatus(
		target.ID,
		"pending_activation",
	)

	f.assertInvitationState(
		target.ID,
		"pending",
		true,
		false,
	)

	f.assertNoAdminSessions(
		target.ID,
	)

	f.assertSecurityEvent(
		target.ID,
		SecurityEventStaffActivationPasswordSet,
		1,
	)

	/*
		Password ownership is one-time for this invitation.

		The same invitation must not overwrite the staff member's
		chosen password after password_set_at has been established.
	*/
	_, err =
		f.service.SetActivationPassword(
			ctx,
			StaffActivationPasswordRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Password: "ActivationDB!DifferentPassw0rd-2026",
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrStaffActivationPasswordAlreadySet,
		"repeat activation password setup",
	)

	/*
		Step 2: create the pending TOTP credential.
	*/
	enrollment, err :=
		f.service.BeginActivationMFA(
			ctx,
			StaffActivationMFAEnrollRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Label: "Integration test authenticator",
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"begin activation MFA: %v",
			err,
		)
	}

	if enrollment.StaffAccountID !=
		target.ID {

		t.Fatalf(
			"unexpected MFA enrollment staff ID: got %q want %q",
			enrollment.StaffAccountID,
			target.ID,
		)
	}

	if enrollment.Enrollment.CredentialID ==
		"" {

		t.Fatal(
			"activation MFA enrollment returned empty credential ID",
		)
	}

	if enrollment.Enrollment.Secret ==
		"" {

		t.Fatal(
			"activation MFA enrollment returned empty TOTP secret",
		)
	}

	if enrollment.Enrollment.EnrollmentURI ==
		"" {

		t.Fatal(
			"activation MFA enrollment returned empty enrollment URI",
		)
	}

	f.assertTOTPStatus(
		target.ID,
		TOTPCredentialStatusPending,
		false,
	)

	f.assertStaffStatus(
		target.ID,
		"pending_activation",
	)

	f.assertNoAdminSessions(
		target.ID,
	)

	f.assertSecurityEvent(
		target.ID,
		SecurityEventStaffActivationMFAEnrollmentStarted,
		1,
	)

	/*
		Generate the exact valid code for the service's fixed test
		clock, then derive a guaranteed-different same-length code for
		the failure path.
	*/
	validCode,
		_,
		err :=
		GenerateTOTP(
			enrollment.Enrollment.Secret,
			f.now,
			DefaultTOTPConfig(),
		)
	if err != nil {
		t.Fatalf(
			"generate activation TOTP: %v",
			err,
		)
	}

	invalidCode :=
		differentTOTPCode(
			validCode,
		)

	/*
		A failed TOTP confirmation must not activate any part of the
		account.
	*/
	_, err =
		f.service.ConfirmActivationMFA(
			ctx,
			StaffActivationMFAConfirmRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Code: invalidCode,
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrInvalidMFACode,
		"wrong activation TOTP",
	)

	f.assertStaffStatus(
		target.ID,
		"pending_activation",
	)

	f.assertInvitationState(
		target.ID,
		"pending",
		true,
		false,
	)

	f.assertTOTPStatus(
		target.ID,
		TOTPCredentialStatusPending,
		false,
	)

	f.assertRecoveryCodeCount(
		target.ID,
		0,
	)

	f.assertNoAdminSessions(
		target.ID,
	)

	/*
		Step 3: successful TOTP verification completes activation.
	*/
	completed, err :=
		f.service.ConfirmActivationMFA(
			ctx,
			StaffActivationMFAConfirmRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Code: validCode,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"confirm activation MFA: %v",
			err,
		)
	}

	if !completed.Activated {
		t.Fatal(
			"expected completed activation to report activated=true",
		)
	}

	if completed.StaffAccountID !=
		target.ID {

		t.Fatalf(
			"unexpected activated staff ID: got %q want %q",
			completed.StaffAccountID,
			target.ID,
		)
	}

	if !strings.EqualFold(
		completed.Email,
		target.Email,
	) {
		t.Fatalf(
			"unexpected activated email: got %q want %q",
			completed.Email,
			target.Email,
		)
	}

	if completed.Next !=
		"login" {

		t.Fatalf(
			"expected activation next=login, got %q",
			completed.Next,
		)
	}

	if len(
		completed.RecoveryCodes,
	) !=
		DefaultRecoveryCodeCount {

		t.Fatalf(
			"expected %d recovery codes, got %d",
			DefaultRecoveryCodeCount,
			len(
				completed.RecoveryCodes,
			),
		)
	}

	f.assertStaffStatus(
		target.ID,
		"active",
	)

	f.assertInvitationState(
		target.ID,
		"accepted",
		true,
		true,
	)

	f.assertTOTPStatus(
		target.ID,
		TOTPCredentialStatusActive,
		true,
	)

	f.assertRecoveryCodeCount(
		target.ID,
		DefaultRecoveryCodeCount,
	)

	f.assertRecoveryCodesMatch(
		target.ID,
		completed.RecoveryCodes,
	)

	/*
		Activation itself must never authenticate the browser.

		No Admin session is allowed to exist merely because activation
		was completed.
	*/
	f.assertNoAdminSessions(
		target.ID,
	)

	f.assertSecurityEvent(
		target.ID,
		SecurityEventStaffActivationCompleted,
		1,
	)

	/*
		The accepted invitation is terminal.

		The raw token may no longer be replayed against any activation
		stage.
	*/
	_, err =
		f.service.SetActivationPassword(
			ctx,
			StaffActivationPasswordRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Password: newPassword,
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrStaffActivationCompleted,
		"activation replay after completion",
	)

	_, err =
		f.service.BeginActivationMFA(
			ctx,
			StaffActivationMFAEnrollRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Label: "Replay",
			},
			metadata,
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrStaffActivationCompleted,
		"MFA enrollment replay after completion",
	)

	/*
		After activation, the staff member can finally use the normal
		Admin password-login endpoint.

		This should produce an MFA login challenge, not a session.
	*/
	loginResult, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: target.Email,

				Password: newPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"normal login after activation: %v",
			err,
		)
	}

	if loginResult.ChallengeToken ==
		"" {

		t.Fatal(
			"normal login after activation returned empty challenge token",
		)
	}

	if loginResult.MFAEnrollmentRequired {
		t.Fatal(
			"activated staff should already have active MFA",
		)
	}

	/*
		Password login only creates an MFA challenge.

		A session still must not exist until normal MFA verification.
	*/
	f.assertNoAdminSessions(
		target.ID,
	)
}

func TestStaffActivationDBExpiredInvitation(
	t *testing.T,
) {
	f :=
		newStaffActivationDBFixture(
			t,
		)

	ctx :=
		context.Background()

	creator :=
		f.addStaff(
			"expiry-creator",
			"active",
			false,
		)

	target :=
		f.addStaff(
			"expiry-target",
			"pending_activation",
			true,
		)

	activationToken :=
		f.addInvitation(
			target,
			creator.ID,
			f.now.Add(
				-2*time.Hour,
			),
			f.now.Add(
				-time.Minute,
			),
		)

	_, err :=
		f.service.SetActivationPassword(
			ctx,
			StaffActivationPasswordRequest{
				Email: target.Email,

				ActivationToken: activationToken,

				Password: "ActivationDB!ExpiredPassw0rd-2026",
			},
			f.metadata(),
		)

	assertStaffActivationDBErrorIs(
		t,
		err,
		ErrStaffActivationExpired,
		"expired activation invitation",
	)

	f.assertStaffStatus(
		target.ID,
		"pending_activation",
	)

	f.assertInvitationState(
		target.ID,
		"expired",
		false,
		false,
	)

	f.assertTOTPAbsent(
		target.ID,
	)

	f.assertRecoveryCodeCount(
		target.ID,
		0,
	)

	f.assertNoAdminSessions(
		target.ID,
	)
}

func newStaffActivationDBFixture(
	t *testing.T,
) *staffActivationDBFixture {
	t.Helper()

	db :=
		openStaffActivationTestDB(
			t,
		)

	redisClient :=
		openStaffActivationTestRedis(
			t,
		)

	repository :=
		NewRepository(
			db,
		)

	keyring, err :=
		platformsecurity.NewSingleKeyKeyring(
			"staff-activation-db-test-key-v1",
			[]byte{
				0x11, 0x12, 0x13, 0x14,
				0x15, 0x16, 0x17, 0x18,
				0x21, 0x22, 0x23, 0x24,
				0x25, 0x26, 0x27, 0x28,
				0x31, 0x32, 0x33, 0x34,
				0x35, 0x36, 0x37, 0x38,
				0x41, 0x42, 0x43, 0x44,
				0x45, 0x46, 0x47, 0x48,
			},
		)
	if err != nil {
		t.Fatalf(
			"create Admin activation test encryption keyring: %v",
			err,
		)
	}

	mfaService, err :=
		NewMFAService(
			repository,
			keyring,
			"Ene Dei Admin Activation DB Test",
		)
	if err != nil {
		t.Fatalf(
			"create Admin activation test MFA service: %v",
			err,
		)
	}

	lockoutService, err :=
		NewLockoutService(
			redisClient,
			DefaultLockoutConfig(),
		)
	if err != nil {
		t.Fatalf(
			"create Admin activation test lockout service: %v",
			err,
		)
	}

	service, err :=
		NewService(
			repository,
			mfaService,
			lockoutService,
		)
	if err != nil {
		t.Fatalf(
			"create Admin activation test auth service: %v",
			err,
		)
	}

	var databaseNow time.Time

	if err :=
		db.QueryRow(
			context.Background(),
			`SELECT now()`,
		).Scan(
			&databaseNow,
		); err != nil {

		t.Fatalf(
			"read PostgreSQL clock for Admin activation test: %v",
			err,
		)
	}

	/*
		Freeze the service and MFA clocks at the next midpoint of a
		30-second TOTP window, derived from PostgreSQL's own clock.

		Using the database clock keeps application-generated expiry
		timestamps compatible with rows whose created_at value is set
		by PostgreSQL now(), while still preventing the test from
		racing a real TOTP boundary.
	*/
	testNow :=
		databaseNow.UTC().
			Truncate(time.Minute).
			Add(15 * time.Second)

	if !testNow.After(
		databaseNow.UTC(),
	) {
		testNow =
			testNow.Add(
				time.Minute,
			)
	}

	f :=
		&staffActivationDBFixture{
			t: t,

			db: db,

			redis: redisClient,

			repository: repository,

			service: service,

			now: testNow,

			ipAddress: "127.0.0.1",
		}

	/*
		Service and MFA must observe exactly the same frozen clock.
		This avoids a test racing a real 30-second TOTP boundary.
	*/
	f.service.now =
		func() time.Time {
			return f.now
		}

	mfaService.now =
		func() time.Time {
			return f.now
		}

	f.ensureAdminAuthorization()

	t.Cleanup(
		f.cleanup,
	)

	return f
}

func openStaffActivationTestDB(
	t *testing.T,
) *pgxpool.Pool {
	t.Helper()

	if os.Getenv(
		runStaffActivationDBTestsEnv,
	) != "1" {

		t.Skipf(
			"set %s=1, %s to a dedicated empty PostgreSQL test database, and %s to a dedicated Redis test database",
			runStaffActivationDBTestsEnv,
			staffActivationTestDBURLEnv,
			staffActivationTestRedisURLEnv,
		)
	}

	rawURL :=
		strings.TrimSpace(
			os.Getenv(
				staffActivationTestDBURLEnv,
			),
		)

	if rawURL == "" {
		t.Fatalf(
			"%s is required when %s=1",
			staffActivationTestDBURLEnv,
			runStaffActivationDBTestsEnv,
		)
	}

	cfg,
		databaseName :=
		staffActivationTestConfigFromURL(
			t,
			rawURL,
		)

	/*
		Hard PostgreSQL safety guard.

		Exactly like the existing Admin-ban integration harness, this
		must never run against a normal development or production DB.
	*/
	if !strings.Contains(
		strings.ToLower(
			databaseName,
		),
		"test",
	) {
		t.Fatalf(
			"refusing staff-activation integration test against database %q: name must contain 'test'",
			databaseName,
		)
	}

	migrationCtx,
		migrationCancel :=
		context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
	defer migrationCancel()

	if _, err :=
		platformdatabase.ApplyMigrations(
			migrationCtx,
			cfg,
		); err != nil {

		t.Fatalf(
			"apply migrations to dedicated staff-activation test database: %v",
			err,
		)
	}

	poolConfig, err :=
		pgxpool.ParseConfig(
			rawURL,
		)
	if err != nil {
		t.Fatalf(
			"parse staff-activation test database URL: %v",
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
			"open staff-activation test database: %v",
			err,
		)
	}

	t.Cleanup(
		db.Close,
	)

	pingCtx,
		pingCancel :=
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
			"ping staff-activation test database: %v",
			err,
		)
	}

	requiredTables :=
		[]string{
			"staff_accounts",
			"staff_invitations",
			"admin_totp_credentials",
			"admin_mfa_recovery_codes",
			"admin_security_events",
			"admin_sessions",
		}

	for _, table := range requiredTables {

		var exists bool

		if err :=
			db.QueryRow(
				context.Background(),
				`
					SELECT
						to_regclass(
							'public.' || $1
						) IS NOT NULL
				`,
				table,
			).Scan(
				&exists,
			); err != nil {

			t.Fatalf(
				"verify test table %s: %v",
				table,
				err,
			)
		}

		if !exists {
			t.Fatalf(
				"dedicated activation test database is missing public.%s after migrations",
				table,
			)
		}
	}

	/*
		Require deterministic isolation.

		Do not silently delete unknown data from a database someone
		accidentally reused for another test.
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
			"count existing activation test staff: %v",
			err,
		)
	}

	if staffCount != 0 {
		t.Fatalf(
			"refusing staff-activation integration test: dedicated database %q already contains %d staff account(s)",
			databaseName,
			staffCount,
		)
	}

	return db
}

func openStaffActivationTestRedis(
	t *testing.T,
) *redis.Client {
	t.Helper()

	rawURL :=
		strings.TrimSpace(
			os.Getenv(
				staffActivationTestRedisURLEnv,
			),
		)

	if rawURL == "" {
		t.Fatalf(
			"%s is required when %s=1",
			staffActivationTestRedisURLEnv,
			runStaffActivationDBTestsEnv,
		)
	}

	options, err :=
		redis.ParseURL(
			rawURL,
		)
	if err != nil {
		t.Fatalf(
			"parse %s: %v",
			staffActivationTestRedisURLEnv,
			err,
		)
	}

	/*
		Hard Redis safety guard.

		We FlushDB below because lockout state must be deterministic.
		Only allow loopback Redis and require a non-default logical DB.
	*/
	host :=
		strings.ToLower(
			strings.TrimSpace(
				options.Addr,
			),
		)

	if !strings.HasPrefix(
		host,
		"127.0.0.1:",
	) &&
		!strings.HasPrefix(
			host,
			"localhost:",
		) &&
		!strings.HasPrefix(
			host,
			"[::1]:",
		) {

		t.Fatalf(
			"refusing activation integration test against non-loopback Redis %q",
			options.Addr,
		)
	}

	if options.DB <= 0 {
		t.Fatalf(
			"refusing activation integration test against Redis DB %d: use a dedicated non-zero logical DB",
			options.DB,
		)
	}

	client :=
		redis.NewClient(
			options,
		)

	t.Cleanup(
		func() {
			_ =
				client.Close()
		},
	)

	ctx,
		cancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
	defer cancel()

	if err :=
		client.Ping(
			ctx,
		).Err(); err != nil {

		t.Fatalf(
			"ping activation test Redis: %v",
			err,
		)
	}

	/*
		This DB is explicitly dedicated by the safety requirements
		above. Clear stale lockout counters from previous failed runs.
	*/
	if err :=
		client.FlushDB(
			ctx,
		).Err(); err != nil {

		t.Fatalf(
			"flush activation test Redis DB: %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			cleanupCtx,
				cleanupCancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
			defer cleanupCancel()

			_ =
				client.FlushDB(
					cleanupCtx,
				).Err()
		},
	)

	return client
}

func staffActivationTestConfigFromURL(
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
			staffActivationTestDBURLEnv,
			err,
		)
	}

	if parsed.Scheme !=
		"postgres" &&
		parsed.Scheme !=
			"postgresql" {

		t.Fatalf(
			"%s must use postgres:// or postgresql://",
			staffActivationTestDBURLEnv,
		)
	}

	if parsed.User == nil {
		t.Fatalf(
			"%s must include a database user",
			staffActivationTestDBURLEnv,
		)
	}

	password,
		_ :=
		parsed.User.Password()

	port :=
		uint16(
			5432,
		)

	if value :=
		parsed.Port(); value != "" {

		parsedPort,
			parseErr :=
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
			staffActivationTestDBURLEnv,
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

func (
	f *staffActivationDBFixture,
) ensureAdminAuthorization() {
	f.t.Helper()

	/*
		The activation service checks panel access using the normal
		RBAC relationship, so the test creates a real permission,
		role, role-permission assignment and staff-role assignment.
	*/
	f.exec(
		`
			INSERT INTO staff_permissions (
				code,
				description,
				created_at
			)
			VALUES (
				$1,
				'Staff activation DB integration-test panel access',
				now()
			)
			ON CONFLICT (code)
			DO NOTHING
		`,
		staffActivationPanelPermission,
	)

	var roleID string

	if err :=
		f.db.QueryRow(
			context.Background(),
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
					'Activation Test Admin',
					'Staff activation DB integration-test role',
					false,
					now(),
					now()
				)
				ON CONFLICT (code)
				DO UPDATE SET
					name = EXCLUDED.name,
					description = EXCLUDED.description,
					updated_at = now()
				RETURNING id::text
			`,
			staffActivationTestRoleCode,
		).Scan(
			&roleID,
		); err != nil {

		f.t.Fatalf(
			"ensure activation test role: %v",
			err,
		)
	}

	tag, err :=
		f.db.Exec(
			context.Background(),
			`
				INSERT INTO staff_role_permissions (
					role_id,
					permission_id,
					created_at
				)
				SELECT
					$1::uuid,
					id,
					now()
				FROM staff_permissions
				WHERE code = $2
				ON CONFLICT DO NOTHING
			`,
			roleID,
			staffActivationPanelPermission,
		)
	if err != nil {
		f.t.Fatalf(
			"assign Admin panel permission to activation test role: %v",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		var exists bool

		if err :=
			f.db.QueryRow(
				context.Background(),
				`
					SELECT EXISTS (
						SELECT 1
						FROM staff_role_permissions srp
						JOIN staff_permissions sp
							ON sp.id = srp.permission_id
						WHERE
							srp.role_id = $1::uuid
							AND sp.code = $2
					)
				`,
				roleID,
				staffActivationPanelPermission,
			).Scan(
				&exists,
			); err != nil {

			f.t.Fatalf(
				"verify activation role panel permission: %v",
				err,
			)
		}

		if !exists {
			f.t.Fatal(
				"activation test role did not receive Admin panel permission",
			)
		}
	}
}

func (
	f *staffActivationDBFixture,
) addStaff(
	label string,
	status string,
	panelAccess bool,
) staffActivationTestStaff {
	f.t.Helper()

	tag :=
		platformsecurity.HashToken(
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
		"ACT-" +
			strings.ToUpper(
				tag[:12],
			)

	email :=
		tag +
			"@adminauth-activation.integration.test"

	placeholderHash, err :=
		HashNewPassword(
			"ActivationDB!PlaceholderPassw0rd-" +
				tag[:16],
		)
	if err != nil {
		f.t.Fatalf(
			"hash activation fixture password: %v",
			err,
		)
	}

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
					$4,
					$5,
					now(),
					now()
				)
				RETURNING id::text
			`,
			staffCode,
			"Staff Activation DB "+label,
			email,
			placeholderHash,
			status,
		).Scan(
			&staffID,
		); err != nil {

		f.t.Fatalf(
			"insert activation test staff %s: %v",
			label,
			err,
		)
	}

	f.staffIDs =
		append(
			f.staffIDs,
			staffID,
		)

	if panelAccess {
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
				staffActivationTestRoleCode,
			)
		if err != nil {
			f.t.Fatalf(
				"assign activation test role to %s: %v",
				label,
				err,
			)
		}

		if tag.RowsAffected() !=
			1 {

			f.t.Fatalf(
				"activation test role was not assigned to %s",
				label,
			)
		}
	}

	return staffActivationTestStaff{
		ID: staffID,

		Email: email,
	}
}

func (
	f *staffActivationDBFixture,
) addInvitation(
	target staffActivationTestStaff,
	creatorStaffID string,
	createdAt time.Time,
	expiresAt time.Time,
) string {
	f.t.Helper()

	token, err :=
		platformsecurity.RandomURLSafe(
			32,
		)
	if err != nil {
		f.t.Fatalf(
			"generate activation fixture token: %v",
			err,
		)
	}

	tokenHash :=
		platformsecurity.HashToken(
			token,
		)

	if _, err :=
		f.db.Exec(
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
					password_set_at,
					accepted_at,
					cancelled_at,
					delivered_at,
					delivery_error,
					created_by_staff_id,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					'manual',
					'not_requested',
					'pending',
					$4,
					NULL,
					NULL,
					NULL,
					NULL,
					NULL,
					$5::uuid,
					$6,
					$6
				)
			`,
			target.ID,
			target.Email,
			tokenHash,
			expiresAt,
			creatorStaffID,
			createdAt,
		); err != nil {

		f.t.Fatalf(
			"insert staff activation invitation: %v",
			err,
		)
	}

	return token
}

func (
	f *staffActivationDBFixture,
) metadata() ClientMetadata {
	return ClientMetadata{
		IPAddress: f.ipAddress,

		UserAgent: "staff-activation-db-integration-test",

		RequestID: "activation-db-" +
			platformsecurity.HashToken(
				fmt.Sprintf(
					"%d",
					time.Now().
						UnixNano(),
				),
			)[:20],
	}
}

func (
	f *staffActivationDBFixture,
) assertStaffStatus(
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
			"load activation staff status: %v",
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

func (
	f *staffActivationDBFixture,
) assertInvitationState(
	staffID string,
	expectedStatus string,
	expectPasswordSet bool,
	expectAccepted bool,
) {
	f.t.Helper()

	var status string
	var passwordSet bool
	var accepted bool

	if err :=
		f.db.QueryRow(
			context.Background(),
			`
				SELECT
					status,
					password_set_at IS NOT NULL,
					accepted_at IS NOT NULL
				FROM staff_invitations
				WHERE staff_account_id = $1::uuid
				ORDER BY created_at DESC
				LIMIT 1
			`,
			staffID,
		).Scan(
			&status,
			&passwordSet,
			&accepted,
		); err != nil {

		f.t.Fatalf(
			"load staff invitation state: %v",
			err,
		)
	}

	if status !=
		expectedStatus {

		f.t.Fatalf(
			"expected invitation status %q, got %q",
			expectedStatus,
			status,
		)
	}

	if passwordSet !=
		expectPasswordSet {

		f.t.Fatalf(
			"expected invitation password_set=%v, got %v",
			expectPasswordSet,
			passwordSet,
		)
	}

	if accepted !=
		expectAccepted {

		f.t.Fatalf(
			"expected invitation accepted=%v, got %v",
			expectAccepted,
			accepted,
		)
	}
}

func (
	f *staffActivationDBFixture,
) assertTOTPStatus(
	staffID string,
	expectedStatus string,
	expectVerified bool,
) {
	f.t.Helper()

	var status string
	var verified bool

	if err :=
		f.db.QueryRow(
			context.Background(),
			`
				SELECT
					status,
					verified_at IS NOT NULL
				FROM admin_totp_credentials
				WHERE staff_account_id = $1::uuid
			`,
			staffID,
		).Scan(
			&status,
			&verified,
		); err != nil {

		f.t.Fatalf(
			"load activation TOTP credential: %v",
			err,
		)
	}

	if status !=
		expectedStatus {

		f.t.Fatalf(
			"expected TOTP status %q, got %q",
			expectedStatus,
			status,
		)
	}

	if verified !=
		expectVerified {

		f.t.Fatalf(
			"expected TOTP verified=%v, got %v",
			expectVerified,
			verified,
		)
	}
}

func (
	f *staffActivationDBFixture,
) assertTOTPAbsent(
	staffID string,
) {
	f.t.Helper()

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_totp_credentials
				WHERE staff_account_id = $1::uuid
			`,
			staffID,
		); count != 0 {

		f.t.Fatalf(
			"expected no TOTP credential, got %d",
			count,
		)
	}
}

func (
	f *staffActivationDBFixture,
) assertRecoveryCodeCount(
	staffID string,
	expected int,
) {
	f.t.Helper()

	count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_mfa_recovery_codes
				WHERE staff_account_id = $1::uuid
			`,
			staffID,
		)

	if count !=
		expected {

		f.t.Fatalf(
			"expected %d persisted recovery codes, got %d",
			expected,
			count,
		)
	}
}

func (
	f *staffActivationDBFixture,
) assertRecoveryCodesMatch(
	staffID string,
	recoveryCodes []string,
) {
	f.t.Helper()

	for _, code := range recoveryCodes {

		hash, err :=
			HashRecoveryCode(
				code,
			)
		if err != nil {
			f.t.Fatalf(
				"hash returned recovery code: %v",
				err,
			)
		}

		count :=
			f.count(
				`
					SELECT COUNT(*)::integer
					FROM admin_mfa_recovery_codes
					WHERE
						staff_account_id = $1::uuid
						AND code_hash = $2
						AND used_at IS NULL
				`,
				staffID,
				hash,
			)

		if count !=
			1 {

			f.t.Fatalf(
				"expected recovery-code hash to exist exactly once, got %d",
				count,
			)
		}
	}
}

func (
	f *staffActivationDBFixture,
) assertNoAdminSessions(
	staffID string,
) {
	f.t.Helper()

	if count :=
		f.count(
			`
				SELECT COUNT(*)::integer
				FROM admin_sessions
				WHERE staff_account_id = $1::uuid
			`,
			staffID,
		); count != 0 {

		f.t.Fatalf(
			"expected activation to create zero Admin sessions, got %d",
			count,
		)
	}
}

func (
	f *staffActivationDBFixture,
) assertSecurityEvent(
	staffID string,
	eventType string,
	expected int,
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
			`,
			staffID,
			eventType,
		)

	if count !=
		expected {

		f.t.Fatalf(
			"expected %d %s security event(s), got %d",
			expected,
			eventType,
			count,
		)
	}
}

func (
	f *staffActivationDBFixture,
) count(
	query string,
	args ...any,
) int {
	f.t.Helper()

	var result int

	if err :=
		f.db.QueryRow(
			context.Background(),
			query,
			args...,
		).Scan(
			&result,
		); err != nil {

		f.t.Fatalf(
			"activation fixture count query failed: %v",
			err,
		)
	}

	return result
}

func (
	f *staffActivationDBFixture,
) exec(
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
			"activation fixture SQL failed: %v",
			err,
		)
	}
}

func (
	f *staffActivationDBFixture,
) cleanup() {
	ctx,
		cancel :=
		context.WithTimeout(
			context.Background(),
			15*time.Second,
		)
	defer cancel()

	/*
		Most Admin-auth state cascades from staff_accounts.

		Security events use ON DELETE SET NULL rather than CASCADE,
		so explicitly remove them first to avoid leaving fixture
		history in the dedicated DB.
	*/
	for _, staffID := range f.staffIDs {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM admin_security_events
					WHERE staff_account_id = $1::uuid
				`,
				staffID,
			)
	}

	/*
		staff_invitations references created_by_staff_id with
		ON DELETE NO ACTION, so remove invitations before deleting
		the creator account.
	*/
	for _, staffID := range f.staffIDs {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM staff_invitations
					WHERE
						staff_account_id = $1::uuid
						OR created_by_staff_id = $1::uuid
				`,
				staffID,
			)
	}

	for index :=
		len(
			f.staffIDs,
		) - 1; index >= 0; index-- {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM staff_accounts
					WHERE id = $1::uuid
				`,
				f.staffIDs[index],
			)
	}

	_, _ =
		f.db.Exec(
			ctx,
			`
				DELETE FROM staff_roles
				WHERE code = $1
			`,
			staffActivationTestRoleCode,
		)

	/*
		Only remove the test-created panel permission when no other
		role references it.

		On a truly dedicated empty DB this normally deletes it.
	*/
	_, _ =
		f.db.Exec(
			ctx,
			`
				DELETE FROM staff_permissions sp
				WHERE
					sp.code = $1
					AND NOT EXISTS (
						SELECT 1
						FROM staff_role_permissions srp
						WHERE srp.permission_id = sp.id
					)
			`,
			staffActivationPanelPermission,
		)
}

func differentTOTPCode(
	valid string,
) string {
	if valid == "" {
		return "000000"
	}

	result :=
		[]byte(
			valid,
		)

	if result[0] ==
		'0' {

		result[0] =
			'1'
	} else {
		result[0] =
			'0'
	}

	return string(
		result,
	)
}

func assertStaffActivationDBErrorIs(
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

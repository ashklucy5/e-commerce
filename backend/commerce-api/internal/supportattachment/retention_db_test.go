package supportattachment_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	platformconfig "project.local/commerce-api/internal/platform/config"
	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformstorage "project.local/commerce-api/internal/platform/storage"
	"project.local/commerce-api/internal/support"
	"project.local/commerce-api/internal/supportattachment"
)

const (
	runRetentionDBTestEnv = "RUN_SUPPORT_ATTACHMENT_RETENTION_DB_TEST"

	retentionTestDatabaseURLEnv = "SUPPORT_ATTACHMENT_RETENTION_TEST_DATABASE_URL"
)

type retentionDBFixture struct {
	t *testing.T

	db *pgxpool.Pool

	storage *retentionMemoryStorage

	retention *supportattachment.RetentionService

	customerID string
	actorID    string
	queueID    string
	caseID     string
	messageID  string
}

type retentionMemoryStorage struct {
	mu sync.Mutex

	objects map[string]struct{}

	deleteCalls map[string]int
}

func TestSupportAttachmentRetentionDBLifecycle(
	t *testing.T,
) {
	f :=
		newRetentionDBFixture(
			t,
		)

	ctx :=
		context.Background()

	// ---------------------------------------------------------
	// Seed a resolved CRM case
	// ---------------------------------------------------------

	f.seedResolvedCase()

	// ---------------------------------------------------------
	// Seed two attached private support images
	//
	// One physically exists in storage.
	// The second intentionally does not exist so retention
	// verifies storage.ErrNotFound is safely tombstoned.
	// ---------------------------------------------------------

	existingKey :=
		"private/support/test-retention/" +
			"attached-existing.jpg"

	missingKey :=
		"private/support/test-retention/" +
			"attached-already-missing.jpg"

	existingAttachmentID :=
		f.addAttachedAttachment(
			existingKey,
		)

	missingAttachmentID :=
		f.addAttachedAttachment(
			missingKey,
		)

	f.storage.Put(
		existingKey,
	)

	// ---------------------------------------------------------
	// RESOLVED MUST NEVER START RETENTION
	//
	// Even using a logical time 30 days in the future, the
	// attachments must remain because the case is not closed.
	// ---------------------------------------------------------

	resolvedResult, err :=
		f.retention.Run(
			ctx,
			time.Now().
				UTC().
				Add(
					30*24*time.Hour,
				),
			100,
		)
	if err != nil {
		t.Fatalf(
			"run retention while case is resolved: %v",
			err,
		)
	}

	if resolvedResult.ClosedCandidates != 0 ||
		resolvedResult.ClosedDeleted != 0 {

		t.Fatalf(
			"resolved case unexpectedly produced retention work: %+v",
			resolvedResult,
		)
	}

	f.assertAttachmentState(
		existingAttachmentID,
		string(
			supportattachment.StatusAttached,
		),
		false,
		true,
	)

	f.assertAttachmentState(
		missingAttachmentID,
		string(
			supportattachment.StatusAttached,
		),
		false,
		true,
	)

	if !f.storage.Exists(
		existingKey,
	) {
		t.Fatal(
			"resolved case attachment was deleted from storage",
		)
	}

	// ---------------------------------------------------------
	// Explicit resolved -> closed transition
	// ---------------------------------------------------------

	supportRepository :=
		support.NewRepository(
			f.db,
		)

	closedTicket, err :=
		supportRepository.CloseCase(
			ctx,
			f.actorID,
			f.caseID,
		)
	if err != nil {
		t.Fatalf(
			"close resolved support case: %v",
			err,
		)
	}

	if closedTicket.Status !=
		"closed" {

		t.Fatalf(
			"closed ticket status = %q, want closed",
			closedTicket.Status,
		)
	}

	if closedTicket.ClosedAt ==
		nil {

		t.Fatal(
			"closed support case has no closed_at timestamp",
		)
	}

	closedAt :=
		closedTicket.ClosedAt.
			UTC()

	// ---------------------------------------------------------
	// Before 7 days: still retained
	// ---------------------------------------------------------

	beforeRetention :=
		closedAt.Add(
			supportattachment.
				RetentionAfterCaseClose -
				time.Minute,
		)

	beforeResult, err :=
		f.retention.Run(
			ctx,
			beforeRetention,
			100,
		)
	if err != nil {
		t.Fatalf(
			"run retention before seven-day cutoff: %v",
			err,
		)
	}

	if beforeResult.ClosedCandidates != 0 ||
		beforeResult.ClosedDeleted != 0 {

		t.Fatalf(
			"attachment became eligible before seven days: %+v",
			beforeResult,
		)
	}

	f.assertAttachmentState(
		existingAttachmentID,
		string(
			supportattachment.StatusAttached,
		),
		false,
		true,
	)

	if !f.storage.Exists(
		existingKey,
	) {
		t.Fatal(
			"storage object was deleted before seven-day cutoff",
		)
	}

	// ---------------------------------------------------------
	// After 7 days: both attached images become eligible
	// ---------------------------------------------------------

	afterRetention :=
		closedAt.Add(
			supportattachment.
				RetentionAfterCaseClose +
				time.Minute,
		)

	afterResult, err :=
		f.retention.Run(
			ctx,
			afterRetention,
			100,
		)
	if err != nil {
		t.Fatalf(
			"run retention after seven-day cutoff: %v",
			err,
		)
	}

	if afterResult.Failed != 0 {
		t.Fatalf(
			"retention reported failures: %+v",
			afterResult,
		)
	}

	if afterResult.ClosedCandidates !=
		2 {

		t.Fatalf(
			"closed candidates = %d, want 2",
			afterResult.ClosedCandidates,
		)
	}

	if afterResult.ClosedDeleted !=
		2 {

		t.Fatalf(
			"closed deleted = %d, want 2",
			afterResult.ClosedDeleted,
		)
	}

	// ---------------------------------------------------------
	// Existing B2 object deleted
	// ---------------------------------------------------------

	if f.storage.Exists(
		existingKey,
	) {
		t.Fatal(
			"eligible closed-case attachment still exists in storage",
		)
	}

	if f.storage.DeleteCalls(
		existingKey,
	) != 1 {

		t.Fatalf(
			"existing attachment delete calls = %d, want 1",
			f.storage.DeleteCalls(
				existingKey,
			),
		)
	}

	// ---------------------------------------------------------
	// Missing B2 object is still safely tombstoned
	// ---------------------------------------------------------

	if f.storage.DeleteCalls(
		missingKey,
	) != 1 {

		t.Fatalf(
			"missing attachment delete calls = %d, want 1",
			f.storage.DeleteCalls(
				missingKey,
			),
		)
	}

	// ---------------------------------------------------------
	// Metadata survives as deleted tombstones
	//
	// message_id must remain linked so CRM history can render
	// an expired attachment rather than losing the record.
	// ---------------------------------------------------------

	f.assertAttachmentState(
		existingAttachmentID,
		string(
			supportattachment.StatusDeleted,
		),
		true,
		true,
	)

	f.assertAttachmentState(
		missingAttachmentID,
		string(
			supportattachment.StatusDeleted,
		),
		true,
		true,
	)

	// ---------------------------------------------------------
	// Orphan cleanup
	// ---------------------------------------------------------

	orphanNow :=
		time.Now().
			UTC()

	oldPendingKey :=
		"private/support/test-retention/" +
			"orphan-pending.jpg"

	oldReadyKey :=
		"private/support/test-retention/" +
			"orphan-ready.jpg"

	recentReadyKey :=
		"private/support/test-retention/" +
			"recent-ready.jpg"

	oldPendingID :=
		f.addPendingOrphan(
			oldPendingKey,
			orphanNow.Add(
				-25*time.Hour,
			),
		)

	oldReadyID :=
		f.addReadyOrphan(
			oldReadyKey,
			orphanNow.Add(
				-25*time.Hour,
			),
		)

	recentReadyID :=
		f.addReadyOrphan(
			recentReadyKey,
			orphanNow.Add(
				-time.Hour,
			),
		)

	f.storage.Put(
		oldPendingKey,
	)

	f.storage.Put(
		oldReadyKey,
	)

	f.storage.Put(
		recentReadyKey,
	)

	orphanResult, err :=
		f.retention.Run(
			ctx,
			orphanNow,
			100,
		)
	if err != nil {
		t.Fatalf(
			"run orphan retention: %v",
			err,
		)
	}

	if orphanResult.Failed != 0 {
		t.Fatalf(
			"orphan retention reported failures: %+v",
			orphanResult,
		)
	}

	if orphanResult.OrphanCandidates !=
		2 {

		t.Fatalf(
			"orphan candidates = %d, want 2",
			orphanResult.OrphanCandidates,
		)
	}

	if orphanResult.OrphanDeleted !=
		2 {

		t.Fatalf(
			"orphan deleted = %d, want 2",
			orphanResult.OrphanDeleted,
		)
	}

	f.assertAttachmentState(
		oldPendingID,
		string(
			supportattachment.StatusDeleted,
		),
		true,
		false,
	)

	f.assertAttachmentState(
		oldReadyID,
		string(
			supportattachment.StatusDeleted,
		),
		true,
		false,
	)

	f.assertAttachmentState(
		recentReadyID,
		string(
			supportattachment.StatusReady,
		),
		false,
		false,
	)

	if f.storage.Exists(
		oldPendingKey,
	) {
		t.Fatal(
			"old pending orphan was not deleted",
		)
	}

	if f.storage.Exists(
		oldReadyKey,
	) {
		t.Fatal(
			"old ready orphan was not deleted",
		)
	}

	if !f.storage.Exists(
		recentReadyKey,
	) {
		t.Fatal(
			"recent ready upload was incorrectly deleted",
		)
	}

	// ---------------------------------------------------------
	// Idempotency
	// ---------------------------------------------------------

	deleteCountBefore :=
		f.storage.TotalDeleteCalls()

	repeatResult, err :=
		f.retention.Run(
			ctx,
			orphanNow,
			100,
		)
	if err != nil {
		t.Fatalf(
			"repeat retention run: %v",
			err,
		)
	}

	if repeatResult.ClosedCandidates != 0 ||
		repeatResult.ClosedDeleted != 0 ||
		repeatResult.OrphanCandidates != 0 ||
		repeatResult.OrphanDeleted != 0 ||
		repeatResult.Failed != 0 {

		t.Fatalf(
			"repeat retention run was not idempotent: %+v",
			repeatResult,
		)
	}

	if f.storage.TotalDeleteCalls() !=
		deleteCountBefore {

		t.Fatalf(
			"repeat retention performed additional storage deletes: before=%d after=%d",
			deleteCountBefore,
			f.storage.TotalDeleteCalls(),
		)
	}
}

func newRetentionDBFixture(
	t *testing.T,
) *retentionDBFixture {
	t.Helper()

	db :=
		openRetentionTestDB(
			t,
		)

	memoryStorage :=
		newRetentionMemoryStorage()

	gateway :=
		platformstorage.NewGateway(
			memoryStorage,
		)

	repository :=
		supportattachment.NewRepository(
			db,
		)

	fixture :=
		&retentionDBFixture{
			t: t,

			db: db,

			storage: memoryStorage,

			retention: supportattachment.
				NewRetentionService(
					repository,
					gateway,
				),
		}

	t.Cleanup(
		fixture.cleanup,
	)

	return fixture
}

func (f *retentionDBFixture) seedResolvedCase() {
	f.t.Helper()

	ctx :=
		context.Background()

	if err :=
		f.db.QueryRow(
			ctx,
			`
				INSERT INTO customers (
					phone,
					email,
					password_hash,
					full_name,
					status
				)
				VALUES (
					'+8801999999001',
					'support-retention-test@example.invalid',
					'integration-test-password-hash',
					'Support Retention DB Test',
					'active'
				)
				RETURNING id::text
			`,
		).Scan(
			&f.customerID,
		); err != nil {

		f.t.Fatalf(
			"seed retention customer: %v",
			err,
		)
	}

	if err :=
		f.db.QueryRow(
			ctx,
			`
				INSERT INTO support_actors (
					actor_code,
					actor_type,
					staff_account_id,
					display_name,
					status,
					presence,
					max_active_cases
				)
				VALUES (
					'retention-test-actor',
					'system',
					NULL,
					'Retention Test Actor',
					'active',
					'available',
					10
				)
				RETURNING id::text
			`,
		).Scan(
			&f.actorID,
		); err != nil {

		f.t.Fatalf(
			"seed retention support actor: %v",
			err,
		)
	}

	if err :=
		f.db.QueryRow(
			ctx,
			`
				INSERT INTO support_queues (
					code,
					name,
					status,
					sort_order
				)
				VALUES (
					'retention-test',
					'Retention Test Queue',
					'active',
					999
				)
				RETURNING id::text
			`,
		).Scan(
			&f.queueID,
		); err != nil {

		f.t.Fatalf(
			"seed retention support queue: %v",
			err,
		)
	}

	if _, err :=
		f.db.Exec(
			ctx,
			`
				INSERT INTO support_queue_members (
					queue_id,
					support_actor_id,
					membership_role
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					'lead'
				)
			`,
			f.queueID,
			f.actorID,
		); err != nil {

		f.t.Fatalf(
			"seed retention queue membership: %v",
			err,
		)
	}

	if err :=
		f.db.QueryRow(
			ctx,
			`
				INSERT INTO crm_cases (
					case_number,
					customer_id,
					case_type,
					subject,
					status,
					priority,
					context_snapshot,
					last_message_at,
					resolved_at
				)
				VALUES (
					'TEST-RETENTION-0001',
					$1::uuid,
					'general_question',
					'Support attachment retention integration test',
					'resolved',
					'normal',
					'{}'::jsonb,
					now(),
					now()
				)
				RETURNING id::text
			`,
			f.customerID,
		).Scan(
			&f.caseID,
		); err != nil {

		f.t.Fatalf(
			"seed resolved CRM case: %v",
			err,
		)
	}

	if _, err :=
		f.db.Exec(
			ctx,
			`
				INSERT INTO support_case_assignments (
					case_id,
					queue_id,
					support_actor_id,
					assigned_by_actor_id,
					assignment_type,
					assigned_at,
					accepted_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3::uuid,
					$3::uuid,
					'claimed',
					now(),
					now()
				)
			`,
			f.caseID,
			f.queueID,
			f.actorID,
		); err != nil {

		f.t.Fatalf(
			"seed retention case assignment: %v",
			err,
		)
	}

	if err :=
		f.db.QueryRow(
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
					'Support attachment retention DB test message',
					NULL,
					now()
				)
				RETURNING id::text
			`,
			f.caseID,
		).Scan(
			&f.messageID,
		); err != nil {

		f.t.Fatalf(
			"seed retention CRM message: %v",
			err,
		)
	}
}

func (f *retentionDBFixture) addAttachedAttachment(
	storageKey string,
) string {
	f.t.Helper()

	var attachmentID string

	err :=
		f.db.QueryRow(
			context.Background(),
			`
				INSERT INTO crm_support_attachments (
					case_id,
					message_id,
					uploader_type,
					customer_id,
					support_actor_id,
					storage_key,
					original_filename,
					mime_type,
					byte_size,
					status,
					uploaded_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					'customer',
					$3::uuid,
					NULL,
					$4,
					'test-image.jpg',
					'image/jpeg',
					1024,
					'attached',
					now()
				)
				RETURNING id::text
			`,
			f.caseID,
			f.messageID,
			f.customerID,
			storageKey,
		).Scan(
			&attachmentID,
		)

	if err != nil {
		f.t.Fatalf(
			"seed attached support attachment: %v",
			err,
		)
	}

	return attachmentID
}

func (f *retentionDBFixture) addPendingOrphan(
	storageKey string,
	createdAt time.Time,
) string {
	f.t.Helper()

	var attachmentID string

	err :=
		f.db.QueryRow(
			context.Background(),
			`
				INSERT INTO crm_support_attachments (
					case_id,
					message_id,
					uploader_type,
					customer_id,
					support_actor_id,
					storage_key,
					original_filename,
					mime_type,
					byte_size,
					status,
					uploaded_at,
					created_at,
					updated_at
				)
				VALUES (
					NULL,
					NULL,
					'customer',
					$1::uuid,
					NULL,
					$2,
					'orphan-pending.jpg',
					'image/jpeg',
					512,
					'pending',
					NULL,
					$3,
					$3
				)
				RETURNING id::text
			`,
			f.customerID,
			storageKey,
			createdAt,
		).Scan(
			&attachmentID,
		)

	if err != nil {
		f.t.Fatalf(
			"seed pending orphan support attachment: %v",
			err,
		)
	}

	return attachmentID
}

func (f *retentionDBFixture) addReadyOrphan(
	storageKey string,
	uploadedAt time.Time,
) string {
	f.t.Helper()

	var attachmentID string

	err :=
		f.db.QueryRow(
			context.Background(),
			`
				INSERT INTO crm_support_attachments (
					case_id,
					message_id,
					uploader_type,
					customer_id,
					support_actor_id,
					storage_key,
					original_filename,
					mime_type,
					byte_size,
					status,
					uploaded_at,
					created_at,
					updated_at
				)
				VALUES (
					NULL,
					NULL,
					'customer',
					$1::uuid,
					NULL,
					$2,
					'orphan-ready.jpg',
					'image/jpeg',
					768,
					'ready',
					$3,
					$3,
					$3
				)
				RETURNING id::text
			`,
			f.customerID,
			storageKey,
			uploadedAt,
		).Scan(
			&attachmentID,
		)

	if err != nil {
		f.t.Fatalf(
			"seed ready orphan support attachment: %v",
			err,
		)
	}

	return attachmentID
}

func (f *retentionDBFixture) assertAttachmentState(
	attachmentID string,
	wantStatus string,
	wantDeletedAt bool,
	wantMessageLink bool,
) {
	f.t.Helper()

	var (
		status       string
		hasDeletedAt bool
		messageID    string
	)

	err :=
		f.db.QueryRow(
			context.Background(),
			`
				SELECT
					status,
					deleted_at IS NOT NULL,
					COALESCE(
						message_id::text,
						''
					)
				FROM crm_support_attachments
				WHERE id = $1::uuid
			`,
			attachmentID,
		).Scan(
			&status,
			&hasDeletedAt,
			&messageID,
		)

	if err != nil {
		f.t.Fatalf(
			"load support attachment %s: %v",
			attachmentID,
			err,
		)
	}

	if status !=
		wantStatus {

		f.t.Fatalf(
			"attachment %s status = %q, want %q",
			attachmentID,
			status,
			wantStatus,
		)
	}

	if hasDeletedAt !=
		wantDeletedAt {

		f.t.Fatalf(
			"attachment %s deleted_at presence = %v, want %v",
			attachmentID,
			hasDeletedAt,
			wantDeletedAt,
		)
	}

	if wantMessageLink &&
		messageID !=
			f.messageID {

		f.t.Fatalf(
			"attachment %s message_id = %q, want %q",
			attachmentID,
			messageID,
			f.messageID,
		)
	}

	if !wantMessageLink &&
		messageID != "" {

		f.t.Fatalf(
			"orphan attachment %s unexpectedly has message_id %q",
			attachmentID,
			messageID,
		)
	}
}

func (f *retentionDBFixture) cleanup() {
	if f.db == nil {
		return
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			10*time.Second,
		)

	defer cancel()

	/*
		Delete only test-owned rows.

		Never truncate shared tables even though the harness already
		requires a dedicated test database.
	*/

	_, _ =
		f.db.Exec(
			ctx,
			`
				DELETE FROM crm_support_attachments
				WHERE storage_key LIKE
					'private/support/test-retention/%'
			`,
		)

	if f.caseID != "" {
		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM crm_case_events
					WHERE case_id = $1::uuid
				`,
				f.caseID,
			)

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM support_case_assignments
					WHERE case_id = $1::uuid
				`,
				f.caseID,
			)

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM crm_messages
					WHERE case_id = $1::uuid
				`,
				f.caseID,
			)

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM crm_cases
					WHERE id = $1::uuid
				`,
				f.caseID,
			)
	}

	if f.queueID != "" &&
		f.actorID != "" {

		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM support_queue_members
					WHERE
						queue_id = $1::uuid
						AND support_actor_id = $2::uuid
				`,
				f.queueID,
				f.actorID,
			)
	}

	if f.queueID != "" {
		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM support_queues
					WHERE id = $1::uuid
				`,
				f.queueID,
			)
	}

	if f.actorID != "" {
		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM support_actors
					WHERE id = $1::uuid
				`,
				f.actorID,
			)
	}

	if f.customerID != "" {
		_, _ =
			f.db.Exec(
				ctx,
				`
					DELETE FROM customers
					WHERE id = $1::uuid
				`,
				f.customerID,
			)
	}
}

func openRetentionTestDB(
	t *testing.T,
) *pgxpool.Pool {
	t.Helper()

	if os.Getenv(
		runRetentionDBTestEnv,
	) != "1" {

		t.Skipf(
			"set %s=1 and %s to a dedicated empty PostgreSQL test database",
			runRetentionDBTestEnv,
			retentionTestDatabaseURLEnv,
		)
	}

	rawURL :=
		strings.TrimSpace(
			os.Getenv(
				retentionTestDatabaseURLEnv,
			),
		)

	if rawURL == "" {
		t.Fatalf(
			"%s is required when %s=1",
			retentionTestDatabaseURLEnv,
			runRetentionDBTestEnv,
		)
	}

	cfg, databaseName :=
		retentionTestConfigFromURL(
			t,
			rawURL,
		)

	/*
		Hard safety boundary.

		The test must never run against development or production.
	*/
	if !strings.Contains(
		strings.ToLower(
			databaseName,
		),
		"test",
	) {
		t.Fatalf(
			"refusing support attachment retention integration test against database %q: name must contain 'test'",
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
			"apply migrations to support attachment retention test database: %v",
			err,
		)
	}

	poolConfig, err :=
		pgxpool.ParseConfig(
			rawURL,
		)
	if err != nil {
		t.Fatalf(
			"parse support attachment retention test database URL: %v",
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
			"open support attachment retention test database: %v",
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
			"ping support attachment retention test database: %v",
			err,
		)
	}

	var hasAttachmentTable bool

	if err :=
		db.QueryRow(
			context.Background(),
			`
				SELECT
					to_regclass(
						'public.crm_support_attachments'
					) IS NOT NULL
			`,
		).Scan(
			&hasAttachmentTable,
		); err != nil {

		t.Fatalf(
			"verify crm_support_attachments table: %v",
			err,
		)
	}

	if !hasAttachmentTable {
		t.Fatal(
			"dedicated test database is missing public.crm_support_attachments after migrations",
		)
	}

	/*
		Refuse to use a database already containing CRM test data.

		This prevents a mistakenly reused database from producing
		false-positive retention results.
	*/
	var existingCases int

	if err :=
		db.QueryRow(
			context.Background(),
			`
				SELECT COUNT(*)::integer
				FROM crm_cases
			`,
		).Scan(
			&existingCases,
		); err != nil {

		t.Fatalf(
			"count existing CRM cases: %v",
			err,
		)
	}

	if existingCases != 0 {
		t.Fatalf(
			"refusing support attachment retention integration test: dedicated database %q already contains %d CRM case(s)",
			databaseName,
			existingCases,
		)
	}

	var existingAttachments int

	if err :=
		db.QueryRow(
			context.Background(),
			`
				SELECT COUNT(*)::integer
				FROM crm_support_attachments
			`,
		).Scan(
			&existingAttachments,
		); err != nil {

		t.Fatalf(
			"count existing support attachments: %v",
			err,
		)
	}

	if existingAttachments != 0 {
		t.Fatalf(
			"refusing support attachment retention integration test: dedicated database %q already contains %d support attachment(s)",
			databaseName,
			existingAttachments,
		)
	}

	return db
}

func retentionTestConfigFromURL(
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
			retentionTestDatabaseURLEnv,
			err,
		)
	}

	if parsed.Scheme !=
		"postgres" &&
		parsed.Scheme !=
			"postgresql" {

		t.Fatalf(
			"%s must use postgres:// or postgresql://",
			retentionTestDatabaseURLEnv,
		)
	}

	if parsed.User == nil {
		t.Fatalf(
			"%s must include a database user",
			retentionTestDatabaseURLEnv,
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
			retentionTestDatabaseURLEnv,
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

// ---------------------------------------------------------
// Deterministic in-memory storage provider
// ---------------------------------------------------------

func newRetentionMemoryStorage() *retentionMemoryStorage {
	return &retentionMemoryStorage{
		objects: make(
			map[string]struct{},
		),

		deleteCalls: make(
			map[string]int,
		),
	}
}

func (s *retentionMemoryStorage) Name() string {
	return "retention-memory-test"
}

func (s *retentionMemoryStorage) CreateUpload(
	_ context.Context,
	request platformstorage.UploadRequest,
) (
	platformstorage.UploadTarget,
	error,
) {
	return platformstorage.UploadTarget{
			Provider: s.Name(),

			Key: request.Key,

			Method: "PUT",

			URL: "https://storage.test/upload",
		},
		nil
}

func (s *retentionMemoryStorage) CreateDownload(
	_ context.Context,
	request platformstorage.DownloadRequest,
) (
	platformstorage.DownloadTarget,
	error,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists :=
		s.objects[request.Key]; !exists {

		return platformstorage.DownloadTarget{},
			platformstorage.ErrNotFound
	}

	return platformstorage.DownloadTarget{
			Provider: s.Name(),

			Key: request.Key,

			URL: "https://storage.test/download",
		},
		nil
}

func (s *retentionMemoryStorage) PublicURL(
	key string,
) (
	string,
	error,
) {
	return "https://storage.test/public/" +
			key,
		nil
}

func (s *retentionMemoryStorage) Stat(
	_ context.Context,
	key string,
) (
	platformstorage.ObjectInfo,
	error,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists :=
		s.objects[key]; !exists {

		return platformstorage.ObjectInfo{},
			platformstorage.ErrNotFound
	}

	return platformstorage.ObjectInfo{
			Key: key,

			ContentType: "image/jpeg",

			ContentLength: 1024,
		},
		nil
}

func (s *retentionMemoryStorage) Delete(
	_ context.Context,
	key string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteCalls[key]++

	if _, exists :=
		s.objects[key]; !exists {

		return platformstorage.
			ErrNotFound
	}

	delete(
		s.objects,
		key,
	)

	return nil
}

func (s *retentionMemoryStorage) Put(
	key string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.objects[key] = struct{}{}
}

func (s *retentionMemoryStorage) Exists(
	key string,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists :=
		s.objects[key]

	return exists
}

func (s *retentionMemoryStorage) DeleteCalls(
	key string,
) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deleteCalls[key]
}

func (s *retentionMemoryStorage) TotalDeleteCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	total :=
		0

	for _, calls := range s.deleteCalls {

		total +=
			calls
	}

	return total
}

func (s *retentionMemoryStorage) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return fmt.Sprintf(
		"objects=%d delete_calls=%d",
		len(
			s.objects,
		),
		s.totalDeleteCallsLocked(),
	)
}

func (s *retentionMemoryStorage) totalDeleteCallsLocked() int {
	total :=
		0

	for _, calls := range s.deleteCalls {

		total +=
			calls
	}

	return total
}

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const customerInboxColumns = `
	n.id::text,
	n.category,
	n.event_type,
	n.title,
	n.message,
	COALESCE(n.action_url, ''),
	COALESCE(n.order_id::text, ''),
	COALESCE(n.case_id::text, ''),
	'',
	'',
	'',
	n.metadata,
	n.read_at,
	n.created_at
`

const staffInboxColumns = `
	n.id::text,
	n.category,
	n.event_type,
	n.title,
	n.message,
	COALESCE(n.action_url, ''),
	'',
	'',
	COALESCE(n.entity_type, ''),
	COALESCE(n.entity_id, ''),
	n.priority,
	n.metadata,
	r.read_at,
	n.created_at
`

func scanInboxItem(
	row rowScanner,
) (
	InboxItem,
	error,
) {
	var result InboxItem
	var metadata []byte
	var readAt pgtype.Timestamptz

	if err := row.Scan(
		&result.ID,
		&result.Category,
		&result.EventType,
		&result.Title,
		&result.Message,
		&result.ActionURL,
		&result.OrderID,
		&result.CaseID,
		&result.EntityType,
		&result.EntityID,
		&result.Priority,
		&metadata,
		&readAt,
		&result.CreatedAt,
	); err != nil {
		return InboxItem{}, err
	}

	result.Metadata = append(result.Metadata[:0], metadata...)

	if readAt.Valid {
		value := readAt.Time.UTC()
		result.ReadAt = &value
	}

	result.CreatedAt = result.CreatedAt.UTC()

	return result, nil
}

func (r *Repository) InsertCustomerInboxTx(
	ctx context.Context,
	tx pgx.Tx,
	request CustomerInboxRequest,
) (
	InboxItem,
	bool,
	error,
) {
	if tx == nil {
		return InboxItem{}, false, fmt.Errorf("notification transaction is required")
	}

	if err := normalizeCustomerInboxRequest(&request); err != nil {
		return InboxItem{}, false, err
	}

	if request.CustomerID == "" {
		return InboxItem{}, false, nil
	}

	metadata, err := json.Marshal(request.Metadata)
	if err != nil {
		return InboxItem{}, false, fmt.Errorf("encode customer notification metadata: %w", err)
	}

	const query = `
		INSERT INTO customer_notifications (
			dedupe_key,
			customer_id,
			category,
			event_type,
			title,
			message,
			action_url,
			order_id,
			case_id,
			metadata,
			created_at
		)
		VALUES (
			$1,
			$2::uuid,
			$3,
			$4,
			$5,
			$6,
			NULLIF($7, ''),
			NULLIF($8, '')::uuid,
			NULLIF($9, '')::uuid,
			$10::jsonb,
			now()
		)
		ON CONFLICT (dedupe_key) DO NOTHING
		RETURNING
			id::text,
			category,
			event_type,
			title,
			message,
			COALESCE(action_url, ''),
			COALESCE(order_id::text, ''),
			COALESCE(case_id::text, ''),
			'',
			'',
			'',
			metadata,
			read_at,
			created_at
	`

	item, err := scanInboxItem(
		tx.QueryRow(
			ctx,
			query,
			request.DedupeKey,
			request.CustomerID,
			request.Category,
			request.EventType,
			request.Title,
			request.Message,
			request.ActionURL,
			request.OrderID,
			request.CaseID,
			string(metadata),
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return InboxItem{}, false, nil
	}
	if err != nil {
		return InboxItem{}, false, fmt.Errorf("insert customer notification: %w", err)
	}

	return item, true, nil
}

func (r *Repository) InsertStaffEventTx(
	ctx context.Context,
	tx pgx.Tx,
	request StaffEventRequest,
) (
	bool,
	error,
) {
	if tx == nil {
		return false, fmt.Errorf("notification transaction is required")
	}

	if err := normalizeStaffEventRequest(&request); err != nil {
		return false, err
	}

	rows, err := tx.Query(
		ctx,
		`
			SELECT DISTINCT sa.id::text
			FROM staff_accounts sa
			WHERE
				sa.status = 'active'
				AND EXISTS (
					SELECT 1
					FROM staff_account_roles sar
					JOIN staff_role_permissions srp
						ON srp.role_id = sar.role_id
					JOIN staff_permissions sp
						ON sp.id = srp.permission_id
					WHERE
						sar.staff_account_id = sa.id
						AND sp.code = 'admin.notification.read'
				)
				AND (
					$1::text = ''
					OR EXISTS (
						SELECT 1
						FROM staff_account_roles sar
						JOIN staff_role_permissions srp
							ON srp.role_id = sar.role_id
						JOIN staff_permissions sp
							ON sp.id = srp.permission_id
						WHERE
							sar.staff_account_id = sa.id
							AND sp.code = $1
					)
				)
		`,
		request.RequiredPermission,
	)
	if err != nil {
		return false, fmt.Errorf("load staff notification recipients: %w", err)
	}
	defer rows.Close()

	recipients := make([]string, 0, 16)
	for rows.Next() {
		var staffID string
		if err := rows.Scan(&staffID); err != nil {
			return false, fmt.Errorf("scan staff notification recipient: %w", err)
		}
		recipients = append(recipients, staffID)
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate staff notification recipients: %w", err)
	}

	if len(recipients) == 0 {
		return false, nil
	}

	metadata, err := json.Marshal(request.Metadata)
	if err != nil {
		return false, fmt.Errorf("encode staff notification metadata: %w", err)
	}

	var notificationID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO staff_notifications (
				dedupe_key,
				category,
				event_type,
				priority,
				title,
				message,
				action_url,
				entity_type,
				entity_id,
				metadata,
				created_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				NULLIF($7, ''),
				NULLIF($8, ''),
				NULLIF($9, ''),
				$10::jsonb,
				now()
			)
			ON CONFLICT (dedupe_key) DO UPDATE
			SET dedupe_key = EXCLUDED.dedupe_key
			RETURNING id::text
		`,
		request.DedupeKey,
		request.Category,
		request.EventType,
		request.Priority,
		request.Title,
		request.Message,
		request.ActionURL,
		request.EntityType,
		request.EntityID,
		string(metadata),
	).Scan(&notificationID)
	if err != nil {
		return false, fmt.Errorf("insert staff notification: %w", err)
	}

	insertedRecipients := 0
	for _, staffID := range recipients {
		commandTag, err := tx.Exec(
			ctx,
			`
				INSERT INTO staff_notification_recipients (
					notification_id,
					staff_account_id,
					created_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					now()
				)
				ON CONFLICT (notification_id, staff_account_id) DO NOTHING
			`,
			notificationID,
			staffID,
		)
		if err != nil {
			return false, fmt.Errorf("insert staff notification recipient: %w", err)
		}

		insertedRecipients += int(commandTag.RowsAffected())
	}

	return insertedRecipients > 0, nil
}

func (r *Repository) ListCustomerInbox(
	ctx context.Context,
	customerID string,
	options InboxListOptions,
) (
	InboxListResult,
	error,
) {
	options = normalizeInboxListOptions(options)

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				`+customerInboxColumns+`
			FROM customer_notifications n
			WHERE
				n.customer_id = $1::uuid
				AND (NOT $2 OR n.read_at IS NULL)
				AND ($3 = '' OR n.category = $3)
			ORDER BY n.created_at DESC, n.id DESC
			LIMIT $4
			OFFSET $5
		`,
		customerID,
		options.UnreadOnly,
		options.Category,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return InboxListResult{}, fmt.Errorf("list customer notifications: %w", err)
	}
	defer rows.Close()

	items := make([]InboxItem, 0, options.Limit)
	for rows.Next() {
		item, err := scanInboxItem(rows)
		if err != nil {
			return InboxListResult{}, fmt.Errorf("scan customer notification: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return InboxListResult{}, fmt.Errorf("iterate customer notifications: %w", err)
	}

	return InboxListResult{
		Items:  items,
		Limit:  options.Limit,
		Offset: options.Offset,
	}, nil
}

func (r *Repository) CustomerInboxSummary(
	ctx context.Context,
	customerID string,
) (
	InboxSummary,
	error,
) {
	var result InboxSummary

	if err := r.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM customer_notifications
			WHERE customer_id = $1::uuid AND read_at IS NULL
		`,
		customerID,
	).Scan(&result.UnreadCount); err != nil {
		return InboxSummary{}, fmt.Errorf("summarize customer notifications: %w", err)
	}

	return result, nil
}

func (r *Repository) MarkCustomerInboxRead(
	ctx context.Context,
	customerID string,
	notificationID string,
) (bool, error) {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE customer_notifications
			SET read_at = COALESCE(read_at, now())
			WHERE id = $1::uuid AND customer_id = $2::uuid
		`,
		notificationID,
		customerID,
	)
	if err != nil {
		return false, fmt.Errorf("mark customer notification read: %w", err)
	}

	return commandTag.RowsAffected() > 0, nil
}

func (r *Repository) MarkAllCustomerInboxRead(
	ctx context.Context,
	customerID string,
) (int64, error) {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE customer_notifications
			SET read_at = now()
			WHERE customer_id = $1::uuid AND read_at IS NULL
		`,
		customerID,
	)
	if err != nil {
		return 0, fmt.Errorf("mark all customer notifications read: %w", err)
	}

	return commandTag.RowsAffected(), nil
}

func (r *Repository) ListStaffInbox(
	ctx context.Context,
	staffID string,
	options InboxListOptions,
) (
	InboxListResult,
	error,
) {
	options = normalizeInboxListOptions(options)

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				`+staffInboxColumns+`
			FROM staff_notification_recipients r
			JOIN staff_notifications n
				ON n.id = r.notification_id
			WHERE
				r.staff_account_id = $1::uuid
				AND r.dismissed_at IS NULL
				AND (NOT $2 OR r.read_at IS NULL)
				AND ($3 = '' OR n.category = $3)
			ORDER BY n.created_at DESC, n.id DESC
			LIMIT $4
			OFFSET $5
		`,
		staffID,
		options.UnreadOnly,
		options.Category,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return InboxListResult{}, fmt.Errorf("list staff notifications: %w", err)
	}
	defer rows.Close()

	items := make([]InboxItem, 0, options.Limit)
	for rows.Next() {
		item, err := scanInboxItem(rows)
		if err != nil {
			return InboxListResult{}, fmt.Errorf("scan staff notification: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return InboxListResult{}, fmt.Errorf("iterate staff notifications: %w", err)
	}

	return InboxListResult{
		Items:  items,
		Limit:  options.Limit,
		Offset: options.Offset,
	}, nil
}

func (r *Repository) StaffInboxSummary(
	ctx context.Context,
	staffID string,
) (
	InboxSummary,
	error,
) {
	var result InboxSummary

	if err := r.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM staff_notification_recipients
			WHERE
				staff_account_id = $1::uuid
				AND read_at IS NULL
				AND dismissed_at IS NULL
		`,
		staffID,
	).Scan(&result.UnreadCount); err != nil {
		return InboxSummary{}, fmt.Errorf("summarize staff notifications: %w", err)
	}

	return result, nil
}

func (r *Repository) MarkStaffInboxRead(
	ctx context.Context,
	staffID string,
	notificationID string,
) (bool, error) {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE staff_notification_recipients
			SET read_at = COALESCE(read_at, now())
			WHERE
				notification_id = $1::uuid
				AND staff_account_id = $2::uuid
				AND dismissed_at IS NULL
		`,
		notificationID,
		staffID,
	)
	if err != nil {
		return false, fmt.Errorf("mark staff notification read: %w", err)
	}

	return commandTag.RowsAffected() > 0, nil
}

func (r *Repository) MarkAllStaffInboxRead(
	ctx context.Context,
	staffID string,
) (int64, error) {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE staff_notification_recipients
			SET read_at = now()
			WHERE
				staff_account_id = $1::uuid
				AND read_at IS NULL
				AND dismissed_at IS NULL
		`,
		staffID,
	)
	if err != nil {
		return 0, fmt.Errorf("mark all staff notifications read: %w", err)
	}

	return commandTag.RowsAffected(), nil
}

func normalizeInboxListOptions(
	options InboxListOptions,
) InboxListOptions {
	if options.Limit <= 0 {
		options.Limit = 30
	}

	if options.Limit > 100 {
		options.Limit = 100
	}

	if options.Offset < 0 {
		options.Offset = 0
	}

	options.Category = strings.ToLower(
		strings.TrimSpace(options.Category),
	)

	return options
}

func normalizeCustomerInboxRequest(
	request *CustomerInboxRequest,
) error {
	if request == nil {
		return ErrInvalidInput
	}

	request.DedupeKey = strings.TrimSpace(request.DedupeKey)
	request.CustomerID = strings.TrimSpace(request.CustomerID)

	request.Category = strings.ToLower(
		strings.TrimSpace(request.Category),
	)

	request.EventType = strings.ToLower(
		strings.TrimSpace(request.EventType),
	)

	request.Title = strings.TrimSpace(request.Title)
	request.Message = strings.TrimSpace(request.Message)

	request.ActionURL = normalizeActionURL(
		request.ActionURL,
	)

	request.OrderID = strings.TrimSpace(request.OrderID)
	request.CaseID = strings.TrimSpace(request.CaseID)

	if request.Metadata == nil {
		request.Metadata = map[string]any{}
	}

	if request.DedupeKey == "" ||
		len(request.DedupeKey) > 200 ||
		request.CustomerID == "" ||
		!validOptionalUUID(request.CustomerID) ||
		!validCategory(request.Category) ||
		request.EventType == "" ||
		len(request.EventType) > 100 ||
		request.Title == "" ||
		len(request.Title) > 160 ||
		request.Message == "" ||
		len(request.Message) > 1000 ||
		len(request.ActionURL) > 500 ||
		!validOptionalUUID(request.OrderID) ||
		!validOptionalUUID(request.CaseID) {

		return ErrInvalidInput
	}

	return nil
}

func normalizeStaffEventRequest(
	request *StaffEventRequest,
) error {
	if request == nil {
		return ErrInvalidInput
	}

	request.DedupeKey = strings.TrimSpace(request.DedupeKey)

	request.Category = strings.ToLower(
		strings.TrimSpace(request.Category),
	)

	request.EventType = strings.ToLower(
		strings.TrimSpace(request.EventType),
	)

	request.Priority = strings.ToLower(
		strings.TrimSpace(request.Priority),
	)

	request.Title = strings.TrimSpace(request.Title)
	request.Message = strings.TrimSpace(request.Message)

	request.ActionURL = normalizeActionURL(
		request.ActionURL,
	)

	request.EntityType = strings.ToLower(
		strings.TrimSpace(request.EntityType),
	)

	request.EntityID = strings.TrimSpace(request.EntityID)

	request.RequiredPermission = strings.TrimSpace(
		request.RequiredPermission,
	)

	if request.Priority == "" {
		request.Priority = StaffPriorityInfo
	}

	if request.Metadata == nil {
		request.Metadata = map[string]any{}
	}

	if request.DedupeKey == "" ||
		len(request.DedupeKey) > 200 ||
		!validStaffCategory(request.Category) ||
		request.EventType == "" ||
		len(request.EventType) > 100 ||
		!validStaffPriority(request.Priority) ||
		request.Title == "" ||
		len(request.Title) > 160 ||
		request.Message == "" ||
		len(request.Message) > 1000 ||
		len(request.ActionURL) > 500 ||
		len(request.EntityType) > 50 ||
		len(request.EntityID) > 160 ||
		(request.EntityType == "") != (request.EntityID == "") ||
		len(request.RequiredPermission) > 120 {

		return ErrInvalidInput
	}

	return nil
}

func validStaffCategory(value string) bool {
	switch value {
	case StaffCategoryOrder,
		StaffCategoryInventory,
		StaffCategorySourcing,
		StaffCategorySupport,
		StaffCategoryReturn,
		StaffCategoryDelivery,
		StaffCategorySecurity,
		StaffCategorySystem:

		return true

	default:
		return false
	}
}

func validStaffPriority(value string) bool {
	switch value {
	case StaffPriorityInfo,
		StaffPriorityAttention,
		StaffPriorityCritical:

		return true

	default:
		return false
	}
}

func normalizeActionURL(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	if !strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "//") {

		return ""
	}

	return value
}

package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

type AuditEvent struct {
	ID string `json:"id"`

	StaffAccountID string `json:"staff_account_id,omitempty"`

	StaffCode string `json:"staff_code,omitempty"`

	AdminSessionID string `json:"admin_session_id,omitempty"`

	EventType string `json:"event_type"`

	Outcome string `json:"outcome"`

	IdentifierHash string `json:"identifier_hash,omitempty"`

	RequestID string `json:"request_id,omitempty"`

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	Details json.RawMessage `json:"details,omitempty"`

	OccurredAt time.Time `json:"occurred_at"`
}

type AuditFilter struct {
	EventType string

	Outcome string

	StaffID string
}

type AuditPage struct {
	Items []AuditEvent `json:"items"`

	Meta platformpagination.Meta `json:"meta"`
}

func (s *Service) AuditEvents(
	ctx context.Context,
	params platformpagination.Params,
	filter AuditFilter,
) (
	AuditPage,
	error,
) {
	filter.EventType =
		strings.TrimSpace(
			filter.EventType,
		)

	filter.Outcome =
		strings.TrimSpace(
			filter.Outcome,
		)

	filter.StaffID =
		strings.TrimSpace(
			filter.StaffID,
		)

	var total int64

	if err :=
		s.db.QueryRow(
			ctx,
			`
				SELECT COUNT(*)
				FROM admin_security_events e
				WHERE
					($1 = '' OR e.event_type = $1)
					AND ($2 = '' OR e.outcome = $2)
					AND (
						NULLIF($3, '') IS NULL
						OR e.staff_account_id =
							NULLIF($3, '')::uuid
					)
			`,
			filter.EventType,
			filter.Outcome,
			filter.StaffID,
		).Scan(
			&total,
		); err != nil {

		return AuditPage{},
			fmt.Errorf(
				"count Admin audit events: %w",
				err,
			)
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					e.id::text,
					COALESCE(
						e.staff_account_id::text,
						''
					),
					COALESCE(
						sa.staff_code,
						''
					),
					COALESCE(
						e.admin_session_id::text,
						''
					),
					e.event_type,
					e.outcome,
					COALESCE(
						e.identifier_hash,
						''
					),
					COALESCE(
						e.request_id,
						''
					),
					COALESCE(
						e.ip_address,
						''
					),
					COALESCE(
						e.user_agent,
						''
					),
					COALESCE(
						e.details,
						'null'::jsonb
					),
					e.occurred_at
				FROM admin_security_events e
				LEFT JOIN staff_accounts sa
					ON sa.id =
						e.staff_account_id
				WHERE
					($1 = '' OR e.event_type = $1)
					AND ($2 = '' OR e.outcome = $2)
					AND (
						NULLIF($3, '') IS NULL
						OR e.staff_account_id =
							NULLIF($3, '')::uuid
					)
				ORDER BY
					e.occurred_at DESC,
					e.id DESC
				LIMIT $4
				OFFSET $5
			`,
			filter.EventType,
			filter.Outcome,
			filter.StaffID,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return AuditPage{},
			fmt.Errorf(
				"list Admin audit events: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]AuditEvent,
			0,
			params.Limit,
		)

	for rows.Next() {
		var item AuditEvent

		if err :=
			rows.Scan(
				&item.ID,
				&item.StaffAccountID,
				&item.StaffCode,
				&item.AdminSessionID,
				&item.EventType,
				&item.Outcome,
				&item.IdentifierHash,
				&item.RequestID,
				&item.IPAddress,
				&item.UserAgent,
				&item.Details,
				&item.OccurredAt,
			); err != nil {

			return AuditPage{},
				fmt.Errorf(
					"scan Admin audit event: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return AuditPage{},
			fmt.Errorf(
				"iterate Admin audit events: %w",
				err,
			)
	}

	return AuditPage{
			Items: items,

			Meta: platformpagination.NewMeta(
				params,
				total,
			),
		},
		nil
}

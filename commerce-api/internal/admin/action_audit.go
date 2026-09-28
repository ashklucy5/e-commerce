package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/adminauth"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
)

const (
	adminEventCustomerStatusChanged = "customer_status_changed"
	adminEventReviewStatusChanged   = "review_status_changed"
)

type AdminActionMetadata struct {
	StaffAccountID string
	AdminSessionID string

	RequestID string
	IPAddress string
	UserAgent string
}

func adminActionMetadataFromContext(
	c *gin.Context,
) (
	AdminActionMetadata,
	bool,
) {
	principal, ok := adminauth.PrincipalFromContext(c)
	if !ok {
		return AdminActionMetadata{}, false
	}

	requestID, _ :=
		platformmiddleware.RequestIDFromContext(c)

	return AdminActionMetadata{
			StaffAccountID: strings.TrimSpace(
				principal.Staff.ID,
			),

			AdminSessionID: strings.TrimSpace(
				principal.SessionID,
			),

			RequestID: requestID,

			IPAddress: truncateAdminAuditValue(
				strings.TrimSpace(
					c.ClientIP(),
				),
				64,
			),

			UserAgent: truncateAdminAuditValue(
				strings.TrimSpace(
					c.Request.UserAgent(),
				),
				500,
			),
		},
		true
}

func insertAdminActionAuditTx(
	ctx context.Context,
	tx pgx.Tx,
	metadata AdminActionMetadata,
	eventType string,
	details map[string]any,
) error {
	encodedDetails, err :=
		json.Marshal(
			details,
		)
	if err != nil {
		return fmt.Errorf(
			"marshal Admin action audit details: %w",
			err,
		)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				INSERT INTO admin_security_events (
					staff_account_id,
					admin_session_id,
					event_type,
					outcome,
					identifier_hash,
					request_id,
					ip_address,
					user_agent,
					details,
					occurred_at,
					created_at
				)
				VALUES (
					NULLIF($1, '')::uuid,
					NULLIF($2, '')::uuid,
					$3,
					'success',
					NULL,
					NULLIF($4, ''),
					NULLIF($5, ''),
					NULLIF($6, ''),
					$7::jsonb,
					now(),
					now()
				)
			`,
			metadata.StaffAccountID,
			metadata.AdminSessionID,
			eventType,
			metadata.RequestID,
			metadata.IPAddress,
			metadata.UserAgent,
			encodedDetails,
		)
	if err != nil {
		return fmt.Errorf(
			"insert Admin action audit event: %w",
			err,
		)
	}

	return nil
}

func truncateAdminAuditValue(
	value string,
	maxRunes int,
) string {
	if maxRunes <= 0 {
		return ""
	}

	runes :=
		[]rune(value)

	if len(runes) <= maxRunes {
		return value
	}

	return string(
		runes[:maxRunes],
	)
}

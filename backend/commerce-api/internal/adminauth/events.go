package adminauth

import (
	"strings"
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	SecurityEventPasswordLoginSucceeded = "password_login_succeeded"

	SecurityEventPasswordLoginFailed = "password_login_failed"

	SecurityEventLoginBlocked = "login_blocked"

	SecurityEventMFAEnrollmentStarted = "mfa_enrollment_started"

	SecurityEventMFAEnrollmentCompleted = "mfa_enrollment_completed"

	SecurityEventMFAVerified = "mfa_verified"

	SecurityEventMFAFailed = "mfa_failed"

	SecurityEventMFABlocked = "mfa_blocked"

	SecurityEventRecoveryCodeUsed = "recovery_code_used"

	SecurityEventSessionCreated = "session_created"

	SecurityEventSessionRefreshed = "session_refreshed"

	SecurityEventSessionRevoked = "session_revoked"

	SecurityEventLogout = "logout"
)

const (
	SecurityOutcomeSuccess = "success"

	SecurityOutcomeFailure = "failure"

	SecurityOutcomeBlocked = "blocked"

	SecurityOutcomeInfo = "info"
)

func HashLoginIdentifier(
	identifier string,
) *string {
	identifier =
		strings.ToLower(
			strings.TrimSpace(
				identifier,
			),
		)

	if identifier == "" {
		return nil
	}

	value :=
		platformsecurity.SHA256String(
			identifier,
		)

	return &value
}

func newSecurityEvent(
	eventType string,
	outcome string,
	staffAccountID *string,
	sessionID *string,
	identifier string,
	metadata ClientMetadata,
	details map[string]any,
	at time.Time,
) SecurityEvent {
	return SecurityEvent{
		StaffAccountID: staffAccountID,

		AdminSessionID: sessionID,

		EventType: eventType,

		Outcome: outcome,

		IdentifierHash: HashLoginIdentifier(
			identifier,
		),

		RequestID: strings.TrimSpace(
			metadata.RequestID,
		),

		IPAddress: strings.TrimSpace(
			metadata.IPAddress,
		),

		UserAgent: strings.TrimSpace(
			metadata.UserAgent,
		),

		Details: details,

		OccurredAt: at.UTC(),
	}
}

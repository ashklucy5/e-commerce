package adminauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

const adminSelfPasswordChangedTestPassword = "EneDei!Admin-Self-Changed-Password-2026"

func TestAdminSelfPasswordChangeDBLifecycle(t *testing.T) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff, secret, _ := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-password-change",
		adminAuthSessionTestPassword,
	)

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	currentSession := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		secret,
	)

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	otherSession := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		secret,
	)

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	result, err := f.service.ChangeSelfPassword(
		ctx,
		currentSession.Material.AccessToken,
		currentSession.Material.CSRFToken,
		ChangeSelfPasswordRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodTOTP,
				Code:     mustAdminAuthSessionTOTP(t, secret, f.now),
			},
			NewPassword: adminSelfPasswordChangedTestPassword,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("change own Admin password: %v", err)
	}
	if result.Material.AccessToken == "" || result.Material.CSRFToken == "" {
		t.Fatal("self password change did not return rotated session material")
	}

	_, err = f.service.AuthenticateAccessToken(ctx, currentSession.Material.AccessToken)
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("old current access token after password change: expected %v, got %v", ErrInvalidAccessToken, err)
	}

	_, err = f.service.AuthenticateAccessToken(ctx, otherSession.Material.AccessToken)
	if !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("other Admin session after password change: expected %v, got %v", ErrSessionRevoked, err)
	}

	if _, err := f.service.AuthenticateAccessTokenWithCSRF(
		ctx,
		result.Material.AccessToken,
		result.Material.CSRFToken,
	); err != nil {
		t.Fatalf("authenticate rotated session after password change: %v", err)
	}

	_, err = f.service.Login(
		ctx,
		LoginRequest{Identifier: staff.Email, Password: adminAuthSessionTestPassword},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password after self password change: expected %v, got %v", ErrInvalidCredentials, err)
	}

	challenge, err := f.service.Login(
		ctx,
		LoginRequest{Identifier: staff.Email, Password: adminSelfPasswordChangedTestPassword},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("new password login after self password change: %v", err)
	}
	if challenge.MFAEnrollmentRequired {
		t.Fatal("normal self password change unexpectedly disabled MFA")
	}

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	if _, err := f.service.VerifyMFA(
		ctx,
		MFAVerifyRequest{
			ChallengeToken: challenge.ChallengeToken,
			Method:         MFAMethodTOTP,
			Code:           mustAdminAuthSessionTOTP(t, secret, f.now),
		},
		f.metadata(),
	); err != nil {
		t.Fatalf("existing authenticator after self password change: %v", err)
	}

	var events int
	if err := f.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM admin_security_events
			WHERE
				staff_account_id = $1::uuid
				AND event_type = $2
				AND outcome = $3
		`,
		staff.ID,
		SecurityEventSelfPasswordChanged,
		SecurityOutcomeSuccess,
	).Scan(&events); err != nil {
		t.Fatalf("count self password-change security events: %v", err)
	}
	if events != 1 {
		t.Fatalf("expected one self password-change event, got %d", events)
	}
}

func TestAdminSelfPasswordChangeDBRecoveryStepUpIsSingleUse(t *testing.T) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff, secret, recoveryCodes := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-password-change-recovery",
		adminAuthSessionTestPassword,
	)

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	session := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		secret,
	)

	result, err := f.service.ChangeSelfPassword(
		ctx,
		session.Material.AccessToken,
		session.Material.CSRFToken,
		ChangeSelfPasswordRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodRecoveryCode,
				Code:     recoveryCodes[0],
			},
			NewPassword: adminSelfPasswordChangedTestPassword,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("change password with recovery-code step-up: %v", err)
	}

	hash, err := HashRecoveryCode(recoveryCodes[0])
	if err != nil {
		t.Fatalf("hash consumed recovery code: %v", err)
	}
	var usedAt *time.Time
	if err := f.db.QueryRow(
		ctx,
		`
			SELECT used_at
			FROM admin_mfa_recovery_codes
			WHERE
				staff_account_id = $1::uuid
				AND code_hash = $2
		`,
		staff.ID,
		hash,
	).Scan(&usedAt); err != nil {
		t.Fatalf("read recovery code after password-change step-up: %v", err)
	}
	if usedAt == nil {
		t.Fatal("recovery code used for password-change step-up was not consumed")
	}

	_, err = f.service.ChangeSelfPassword(
		ctx,
		result.Material.AccessToken,
		result.Material.CSRFToken,
		ChangeSelfPasswordRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminSelfPasswordChangedTestPassword,
				Method:   MFAMethodRecoveryCode,
				Code:     recoveryCodes[0],
			},
			NewPassword: "EneDei!Admin-Self-Second-Password-2026",
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidMFACode) {
		t.Fatalf("replay recovery code for self password change: expected %v, got %v", ErrInvalidMFACode, err)
	}
}

func TestAdminSelfPasswordChangeDBRejectsUnchangedPassword(t *testing.T) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff, secret, recoveryCodes := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-password-change-unchanged",
		adminAuthSessionTestPassword,
	)

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	session := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		secret,
	)

	_, err := f.service.ChangeSelfPassword(
		ctx,
		session.Material.AccessToken,
		session.Material.CSRFToken,
		ChangeSelfPasswordRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodRecoveryCode,
				Code:     recoveryCodes[0],
			},
			NewPassword: adminAuthSessionTestPassword,
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrPasswordUnchanged) {
		t.Fatalf("unchanged self password: expected %v, got %v", ErrPasswordUnchanged, err)
	}

	hash, err := HashRecoveryCode(recoveryCodes[0])
	if err != nil {
		t.Fatalf("hash recovery code after unchanged-password rejection: %v", err)
	}
	var usedAt *time.Time
	if err := f.db.QueryRow(
		ctx,
		`
			SELECT used_at
			FROM admin_mfa_recovery_codes
			WHERE
				staff_account_id = $1::uuid
				AND code_hash = $2
		`,
		staff.ID,
		hash,
	).Scan(&usedAt); err != nil {
		t.Fatalf("read recovery code after unchanged-password rejection: %v", err)
	}
	if usedAt != nil {
		t.Fatal("unchanged password rejection consumed the recovery code")
	}
}

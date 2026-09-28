package adminauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

const adminRecoveryResetPassword = "EneDei!Recovered-Admin-Password-2026"

func TestAdminRecoveryPasswordResetDBLifecycle(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		_,
		recoveryCodes :=
		prepareAdminAuthSessionStaff(
			t,
			f,
			"recovery-password-reset",
			adminAuthSessionTestPassword,
		)

	if len(recoveryCodes) == 0 {
		t.Fatal("recovery fixture returned no recovery codes")
	}

	metadata := f.metadata()

	challenge, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: staff.Email,
				Password:   adminAuthSessionTestPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("password login before recovery reset: %v", err)
	}

	recoverySession, err :=
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: challenge.ChallengeToken,
				Method:         MFAMethodRecoveryCode,
				Code:           recoveryCodes[0],
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("recovery-code login before password reset: %v", err)
	}

	_, err =
		f.service.ResetPasswordFromRecoverySession(
			ctx,
			recoverySession.Material.AccessToken,
			"wrong-csrf-token",
			RecoveryPasswordResetRequest{
				NewPassword: adminRecoveryResetPassword,
			},
			metadata,
		)
	if !errors.Is(err, ErrInvalidCSRFToken) {
		t.Fatalf(
			"recovery password reset with wrong CSRF: expected %v, got %v",
			ErrInvalidCSRFToken,
			err,
		)
	}

	reset, err :=
		f.service.ResetPasswordFromRecoverySession(
			ctx,
			recoverySession.Material.AccessToken,
			recoverySession.Material.CSRFToken,
			RecoveryPasswordResetRequest{
				NewPassword: adminRecoveryResetPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("reset password from recovery session: %v", err)
	}

	if reset.ChallengeToken == "" {
		t.Fatal("recovery password reset returned an empty MFA enrollment challenge")
	}

	if !reset.MFAEnrollmentRequired {
		t.Fatal("recovery password reset did not require fresh MFA enrollment")
	}

	_, err =
		f.service.AuthenticateAccessToken(
			ctx,
			recoverySession.Material.AccessToken,
		)
	if !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf(
			"recovery session after password reset: expected %v, got %v",
			ErrSessionRevoked,
			err,
		)
	}

	var credentialStatus string
	var recoveryCodeCount int

	if err :=
		f.db.QueryRow(
			ctx,
			`
				SELECT status
				FROM admin_totp_credentials
				WHERE staff_account_id = $1::uuid
			`,
			staff.ID,
		).Scan(
			&credentialStatus,
		); err != nil {

		t.Fatalf("read MFA state after recovery password reset: %v", err)
	}

	if credentialStatus != TOTPCredentialStatusDisabled {
		t.Fatalf(
			"MFA status after recovery password reset = %q, want %q",
			credentialStatus,
			TOTPCredentialStatusDisabled,
		)
	}

	if err :=
		f.db.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM admin_mfa_recovery_codes
				WHERE staff_account_id = $1::uuid
			`,
			staff.ID,
		).Scan(
			&recoveryCodeCount,
		); err != nil {

		t.Fatalf("count recovery codes after password reset: %v", err)
	}

	if recoveryCodeCount != 0 {
		t.Fatalf(
			"expected recovery codes to be removed after password reset, got %d",
			recoveryCodeCount,
		)
	}

	_, err =
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: staff.Email,
				Password:   adminAuthSessionTestPassword,
			},
			metadata,
		)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"old password after recovery reset: expected %v, got %v",
			ErrInvalidCredentials,
			err,
		)
	}

	newPasswordChallenge, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: staff.Email,
				Password:   adminRecoveryResetPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("new password login after recovery reset: %v", err)
	}

	if !newPasswordChallenge.MFAEnrollmentRequired {
		t.Fatal("new password login did not require MFA re-enrollment")
	}

	enrollment, err :=
		f.service.BeginMFAEnrollment(
			ctx,
			MFAEnrollRequest{
				ChallengeToken: reset.ChallengeToken,
				Label:          "Recovered authenticator",
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("begin MFA re-enrollment after recovery reset: %v", err)
	}

	code :=
		mustAdminAuthSessionTOTP(
			t,
			enrollment.Enrollment.Secret,
			f.now,
		)

	completed, err :=
		f.service.ConfirmMFAEnrollment(
			ctx,
			MFAConfirmEnrollmentRequest{
				ChallengeToken: reset.ChallengeToken,
				Code:           code,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("confirm MFA re-enrollment after recovery reset: %v", err)
	}

	if len(completed.RecoveryCodes) != DefaultRecoveryCodeCount {
		t.Fatalf(
			"fresh recovery-code count = %d, want %d",
			len(completed.RecoveryCodes),
			DefaultRecoveryCodeCount,
		)
	}

	if completed.Session.Material.AccessToken == "" {
		t.Fatal("MFA re-enrollment did not create a new Admin session")
	}

	var recoveryResetEvents int

	if err :=
		f.db.QueryRow(
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
			SecurityEventRecoveryPasswordReset,
			SecurityOutcomeSuccess,
		).Scan(
			&recoveryResetEvents,
		); err != nil {

		t.Fatalf("count recovery password-reset security events: %v", err)
	}

	if recoveryResetEvents != 1 {
		t.Fatalf(
			"expected one recovery password-reset event, got %d",
			recoveryResetEvents,
		)
	}
}

func TestAdminRecoveryPasswordResetDBRequiresRecoveryAuthenticatedSession(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		secret,
		_ :=
		prepareAdminAuthSessionStaff(
			t,
			f,
			"recovery-password-totp-denied",
			adminAuthSessionTestPassword,
		)

	f.now =
		f.now.Add(
			time.Duration(DefaultTOTPPeriodSeconds) * time.Second,
		)

	session :=
		mustAdminAuthSessionLoginWithTOTP(
			t,
			f,
			staff,
			adminAuthSessionTestPassword,
			secret,
		)

	_, err :=
		f.service.ResetPasswordFromRecoverySession(
			ctx,
			session.Material.AccessToken,
			session.Material.CSRFToken,
			RecoveryPasswordResetRequest{
				NewPassword: adminRecoveryResetPassword,
			},
			f.metadata(),
		)
	if !errors.Is(err, ErrRecoverySessionRequired) {
		t.Fatalf(
			"TOTP-authenticated password reset: expected %v, got %v",
			ErrRecoverySessionRequired,
			err,
		)
	}
}

func TestAdminRecoveryPasswordResetDBRecoveryProofExpires(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		_,
		recoveryCodes :=
		prepareAdminAuthSessionStaff(
			t,
			f,
			"recovery-password-expiry",
			adminAuthSessionTestPassword,
		)

	metadata := f.metadata()

	challenge, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: staff.Email,
				Password:   adminAuthSessionTestPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("password login before recovery-proof expiry: %v", err)
	}

	session, err :=
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: challenge.ChallengeToken,
				Method:         MFAMethodRecoveryCode,
				Code:           recoveryCodes[0],
			},
			metadata,
		)
	if err != nil {
		t.Fatalf("recovery-code login before recovery-proof expiry: %v", err)
	}

	f.now =
		f.now.Add(
			AdminRecoverySessionTTL - time.Minute,
		)

	refreshed, err :=
		f.service.Refresh(
			ctx,
			session.Material.RefreshToken,
			session.Material.CSRFToken,
			metadata,
		)
	if err != nil {
		t.Fatalf("refresh recovery-authenticated session: %v", err)
	}

	f.now =
		f.now.Add(
			2 * time.Minute,
		)

	_, err =
		f.service.ResetPasswordFromRecoverySession(
			ctx,
			refreshed.Material.AccessToken,
			refreshed.Material.CSRFToken,
			RecoveryPasswordResetRequest{
				NewPassword: adminRecoveryResetPassword,
			},
			metadata,
		)
	if !errors.Is(err, ErrRecoverySessionExpired) {
		t.Fatalf(
			"expired recovery proof: expected %v, got %v",
			ErrRecoverySessionExpired,
			err,
		)
	}
}

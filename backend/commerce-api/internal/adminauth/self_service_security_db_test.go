package adminauth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAdminSelfSecurityDBMFARotationLifecycle(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		oldSecret,
		oldRecoveryCodes := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-mfa-rotation",
		adminAuthSessionTestPassword,
	)

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	currentSession := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		oldSecret,
	)

	// Step-up TOTP must be newer than the code that established the session.
	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	stepUpCode := mustAdminAuthSessionTOTP(t, oldSecret, f.now)

	begin, err := f.service.BeginSelfMFARotation(
		ctx,
		currentSession.Material.AccessToken,
		currentSession.Material.CSRFToken,
		BeginMFARotationRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodTOTP,
				Code:     stepUpCode,
			},
			Label: "Rotated authenticator",
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("begin self MFA rotation: %v", err)
	}
	if begin.RotationToken == "" {
		t.Fatal("self MFA rotation returned empty rotation token")
	}
	if begin.Enrollment.Secret == "" || begin.Enrollment.EnrollmentURI == "" {
		t.Fatal("self MFA rotation returned incomplete enrollment material")
	}
	if begin.Enrollment.Secret == oldSecret {
		t.Fatal("self MFA rotation reused the existing TOTP secret")
	}

	pendingPayload, err := f.redis.Get(
		ctx,
		selfMFARotationRedisKey(staff.ID),
	).Result()
	if err != nil {
		t.Fatalf("read pending MFA rotation from Redis: %v", err)
	}
	if strings.Contains(pendingPayload, begin.Enrollment.Secret) {
		t.Fatal("pending MFA rotation stored the new TOTP secret in plaintext")
	}

	_, err = f.service.ConfirmSelfMFARotation(
		ctx,
		currentSession.Material.AccessToken,
		currentSession.Material.CSRFToken,
		ConfirmMFARotationRequest{
			RotationToken: begin.RotationToken,
			Code:          "not-a-valid-totp",
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidMFACode) {
		t.Fatalf("invalid new authenticator code: expected %v, got %v", ErrInvalidMFACode, err)
	}

	// Starting a rotation, and even a failed confirmation attempt, must not
	// replace or disable the existing MFA.
	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	oldMFASession := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		oldSecret,
	)

	newCode := mustAdminAuthSessionTOTP(t, begin.Enrollment.Secret, f.now)
	completed, err := f.service.ConfirmSelfMFARotation(
		ctx,
		currentSession.Material.AccessToken,
		currentSession.Material.CSRFToken,
		ConfirmMFARotationRequest{
			RotationToken: begin.RotationToken,
			Code:          newCode,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("confirm self MFA rotation: %v", err)
	}
	if len(completed.RecoveryCodes) != DefaultRecoveryCodeCount {
		t.Fatalf(
			"rotated MFA recovery-code count = %d, want %d",
			len(completed.RecoveryCodes),
			DefaultRecoveryCodeCount,
		)
	}
	if completed.Session.Material.AccessToken == "" || completed.Session.Material.CSRFToken == "" {
		t.Fatal("self MFA rotation did not return rotated session material")
	}

	// The old current-session token was rotated.
	_, err = f.service.AuthenticateAccessToken(ctx, currentSession.Material.AccessToken)
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("old current access token after MFA rotation: expected %v, got %v", ErrInvalidAccessToken, err)
	}

	// Every other Admin session was revoked.
	_, err = f.service.AuthenticateAccessToken(ctx, oldMFASession.Material.AccessToken)
	if !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("other Admin session after MFA rotation: expected %v, got %v", ErrSessionRevoked, err)
	}

	if _, err := f.service.AuthenticateAccessTokenWithCSRF(
		ctx,
		completed.Session.Material.AccessToken,
		completed.Session.Material.CSRFToken,
	); err != nil {
		t.Fatalf("authenticate rotated Admin session: %v", err)
	}

	// Old recovery material must no longer exist.
	oldRecoveryHash, err := HashRecoveryCode(oldRecoveryCodes[0])
	if err != nil {
		t.Fatalf("hash old recovery code: %v", err)
	}
	var oldRecoveryRows int
	if err := f.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM admin_mfa_recovery_codes
			WHERE
				staff_account_id = $1::uuid
				AND code_hash = $2
		`,
		staff.ID,
		oldRecoveryHash,
	).Scan(&oldRecoveryRows); err != nil {
		t.Fatalf("count old recovery code after MFA rotation: %v", err)
	}
	if oldRecoveryRows != 0 {
		t.Fatalf("old recovery code remained after MFA rotation: %d rows", oldRecoveryRows)
	}

	// The old authenticator must stop working after confirmation, while the
	// replacement authenticator must work on the same pending login challenge.
	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	challenge, err := f.service.Login(
		ctx,
		LoginRequest{
			Identifier: staff.Email,
			Password:   adminAuthSessionTestPassword,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("password login after MFA rotation: %v", err)
	}

	_, err = f.service.VerifyMFA(
		ctx,
		MFAVerifyRequest{
			ChallengeToken: challenge.ChallengeToken,
			Method:         MFAMethodTOTP,
			Code:           mustAdminAuthSessionTOTP(t, oldSecret, f.now),
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidMFACode) {
		t.Fatalf("old authenticator after confirmed rotation: expected %v, got %v", ErrInvalidMFACode, err)
	}

	newLogin, err := f.service.VerifyMFA(
		ctx,
		MFAVerifyRequest{
			ChallengeToken: challenge.ChallengeToken,
			Method:         MFAMethodTOTP,
			Code:           mustAdminAuthSessionTOTP(t, begin.Enrollment.Secret, f.now),
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("new authenticator login after rotation: %v", err)
	}
	if newLogin.Principal.Staff.ID != staff.ID {
		t.Fatalf("new authenticator login staff ID = %q, want %q", newLogin.Principal.Staff.ID, staff.ID)
	}
}

func TestAdminSelfSecurityDBRotationRecoveryStepUpIsSingleUse(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		secret,
		recoveryCodes := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-mfa-rotation-recovery-step-up",
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

	_, err := f.service.BeginSelfMFARotation(
		ctx,
		session.Material.AccessToken,
		session.Material.CSRFToken,
		BeginMFARotationRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodRecoveryCode,
				Code:     recoveryCodes[0],
			},
			Label: "Recovery stepped-up rotation",
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("begin self MFA rotation with recovery-code step-up: %v", err)
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
		t.Fatalf("read recovery-code step-up state: %v", err)
	}
	if usedAt == nil {
		t.Fatal("recovery code used for self-security step-up was not consumed")
	}

	_, err = f.service.BeginSelfMFARotation(
		ctx,
		session.Material.AccessToken,
		session.Material.CSRFToken,
		BeginMFARotationRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodRecoveryCode,
				Code:     recoveryCodes[0],
			},
			Label: "Replay should fail",
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidMFACode) {
		t.Fatalf("replay recovery-code self-security step-up: expected %v, got %v", ErrInvalidMFACode, err)
	}
}

func TestAdminSelfSecurityDBRecoveryCodeRegeneration(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		secret,
		oldRecoveryCodes := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-recovery-code-regeneration",
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

	// Create another Admin session that must be revoked by the security change.
	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	otherSession := mustAdminAuthSessionLoginWithTOTP(
		t,
		f,
		staff,
		adminAuthSessionTestPassword,
		secret,
	)

	// Advance again so the TOTP used for the step-up is fresh.
	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	result, err := f.service.RegenerateSelfRecoveryCodes(
		ctx,
		currentSession.Material.AccessToken,
		currentSession.Material.CSRFToken,
		RegenerateRecoveryCodesRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodTOTP,
				Code:     mustAdminAuthSessionTOTP(t, secret, f.now),
			},
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("regenerate Admin recovery codes: %v", err)
	}
	if len(result.RecoveryCodes) != DefaultRecoveryCodeCount {
		t.Fatalf(
			"regenerated recovery-code count = %d, want %d",
			len(result.RecoveryCodes),
			DefaultRecoveryCodeCount,
		)
	}

	_, err = f.service.AuthenticateAccessToken(ctx, currentSession.Material.AccessToken)
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("old current access token after recovery-code regeneration: expected %v, got %v", ErrInvalidAccessToken, err)
	}

	_, err = f.service.AuthenticateAccessToken(ctx, otherSession.Material.AccessToken)
	if !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("other Admin session after recovery-code regeneration: expected %v, got %v", ErrSessionRevoked, err)
	}

	if _, err := f.service.AuthenticateAccessTokenWithCSRF(
		ctx,
		result.Session.Material.AccessToken,
		result.Session.Material.CSRFToken,
	); err != nil {
		t.Fatalf("authenticate session after recovery-code regeneration: %v", err)
	}

	oldHash, err := HashRecoveryCode(oldRecoveryCodes[0])
	if err != nil {
		t.Fatalf("hash old recovery code: %v", err)
	}
	var oldRows int
	if err := f.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM admin_mfa_recovery_codes
			WHERE code_hash = $1
		`,
		oldHash,
	).Scan(&oldRows); err != nil {
		t.Fatalf("count old recovery code after regeneration: %v", err)
	}
	if oldRows != 0 {
		t.Fatalf("old recovery code remained after regeneration: %d rows", oldRows)
	}

	newHash, err := HashRecoveryCode(result.RecoveryCodes[0])
	if err != nil {
		t.Fatalf("hash regenerated recovery code: %v", err)
	}
	var newRows int
	if err := f.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM admin_mfa_recovery_codes
			WHERE
				staff_account_id = $1::uuid
				AND code_hash = $2
				AND used_at IS NULL
		`,
		staff.ID,
		newHash,
	).Scan(&newRows); err != nil {
		t.Fatalf("count regenerated recovery code: %v", err)
	}
	if newRows != 1 {
		t.Fatalf("expected regenerated recovery code to be persisted once, got %d rows", newRows)
	}
}

func TestAdminSelfSecurityDBWrongPasswordDoesNotConsumeRecoveryCode(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		secret,
		recoveryCodes := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-security-wrong-password",
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

	_, err := f.service.RegenerateSelfRecoveryCodes(
		ctx,
		session.Material.AccessToken,
		session.Material.CSRFToken,
		RegenerateRecoveryCodesRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: "definitely-wrong-password",
				Method:   MFAMethodRecoveryCode,
				Code:     recoveryCodes[0],
			},
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong current password during recovery-code regeneration: expected %v, got %v", ErrInvalidCredentials, err)
	}

	hash, err := HashRecoveryCode(recoveryCodes[0])
	if err != nil {
		t.Fatalf("hash recovery code after wrong password: %v", err)
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
		t.Fatalf("read recovery code after wrong password: %v", err)
	}
	if usedAt != nil {
		t.Fatal("wrong current password consumed the recovery code")
	}
}

func TestAdminSelfSecurityDBMFARotationExpiresAfterSessionRefresh(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		secret,
		_ := prepareAdminAuthSessionStaff(
		t,
		f,
		"self-mfa-rotation-expiry",
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

	f.now = f.now.Add(time.Duration(DefaultTOTPPeriodSeconds) * time.Second)
	begin, err := f.service.BeginSelfMFARotation(
		ctx,
		session.Material.AccessToken,
		session.Material.CSRFToken,
		BeginMFARotationRequest{
			SecurityStepUpRequest: SecurityStepUpRequest{
				Password: adminAuthSessionTestPassword,
				Method:   MFAMethodTOTP,
				Code:     mustAdminAuthSessionTOTP(t, secret, f.now),
			},
			Label: "Expiring rotation",
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("begin expiring self MFA rotation: %v", err)
	}

	// Refresh the same Admin session before the access token expires. The
	// session ID survives refresh, but the MFA-rotation authorization must not.
	f.now = f.now.Add(AdminMFARotationTTL - 2*time.Minute)
	refreshed, err := f.service.Refresh(
		ctx,
		session.Material.RefreshToken,
		session.Material.CSRFToken,
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("refresh session during pending MFA rotation: %v", err)
	}

	f.now = f.now.Add(3 * time.Minute)
	_, err = f.service.ConfirmSelfMFARotation(
		ctx,
		refreshed.Material.AccessToken,
		refreshed.Material.CSRFToken,
		ConfirmMFARotationRequest{
			RotationToken: begin.RotationToken,
			Code: mustAdminAuthSessionTOTP(
				t,
				begin.Enrollment.Secret,
				f.now,
			),
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrInvalidMFARotation) {
		t.Fatalf("expired MFA rotation after session refresh: expected %v, got %v", ErrInvalidMFARotation, err)
	}
}

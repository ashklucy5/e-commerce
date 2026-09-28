package adminauth

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const adminAuthSessionTestPassword = "EneDei!Admin-Session-Security-2026"

func TestAdminAuthSessionDBLifecycle(
	t *testing.T,
) {
	f :=
		newStaffActivationDBFixture(
			t,
		)

	ctx :=
		context.Background()

	staff,
		secret,
		_ :=
		prepareAdminAuthSessionStaff(
			t,
			f,
			"session-lifecycle",
			adminAuthSessionTestPassword,
		)

	f.now =
		f.now.Add(
			time.Duration(
				DefaultTOTPPeriodSeconds,
			) * time.Second,
		)

	metadata :=
		f.metadata()

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
		t.Fatalf(
			"normal Admin password login: %v",
			err,
		)
	}

	if challenge.ChallengeToken == "" {
		t.Fatal(
			"normal Admin password login returned an empty challenge token",
		)
	}

	if challenge.MFAEnrollmentRequired {
		t.Fatal(
			"active MFA account unexpectedly requires enrollment",
		)
	}

	code :=
		mustAdminAuthSessionTOTP(
			t,
			secret,
			f.now,
		)

	session, err :=
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: challenge.ChallengeToken,
				Method:         MFAMethodTOTP,
				Code:           code,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"normal Admin TOTP verification: %v",
			err,
		)
	}

	if session.Principal.Staff.ID != staff.ID {
		t.Fatalf(
			"authenticated staff ID = %q, want %q",
			session.Principal.Staff.ID,
			staff.ID,
		)
	}

	if session.Material.AccessToken == "" ||
		session.Material.RefreshToken == "" ||
		session.Material.CSRFToken == "" {

		t.Fatal(
			"successful Admin MFA login did not return complete session material",
		)
	}

	if _, err :=
		f.service.AuthenticateAccessTokenWithCSRF(
			ctx,
			session.Material.AccessToken,
			session.Material.CSRFToken,
		); err != nil {

		t.Fatalf(
			"authenticate newly created Admin session: %v",
			err,
		)
	}

	_, err =
		f.service.AuthenticateAccessTokenWithCSRF(
			ctx,
			session.Material.AccessToken,
			"wrong-csrf-token",
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrInvalidCSRFToken,
		"Admin access with wrong CSRF token",
	)

	_, err =
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: challenge.ChallengeToken,
				Method:         MFAMethodTOTP,
				Code:           code,
			},
			metadata,
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrChallengeConsumed,
		"replay consumed Admin login challenge",
	)

	refreshed, err :=
		f.service.Refresh(
			ctx,
			session.Material.RefreshToken,
			session.Material.CSRFToken,
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"refresh Admin session: %v",
			err,
		)
	}

	if refreshed.Principal.SessionID !=
		session.Principal.SessionID {

		t.Fatalf(
			"refresh changed Admin session ID from %q to %q",
			session.Principal.SessionID,
			refreshed.Principal.SessionID,
		)
	}

	if refreshed.Material.AccessToken ==
		session.Material.AccessToken {

		t.Fatal(
			"Admin refresh did not rotate the access token",
		)
	}

	if refreshed.Material.RefreshToken ==
		session.Material.RefreshToken {

		t.Fatal(
			"Admin refresh did not rotate the refresh token",
		)
	}

	_, err =
		f.service.AuthenticateAccessToken(
			ctx,
			session.Material.AccessToken,
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrInvalidAccessToken,
		"old access token after refresh",
	)

	_, err =
		f.service.Refresh(
			ctx,
			session.Material.RefreshToken,
			session.Material.CSRFToken,
			metadata,
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrInvalidRefreshToken,
		"old refresh token after rotation",
	)

	if err :=
		f.service.Logout(
			ctx,
			refreshed.Material.AccessToken,
			"wrong-csrf-token",
			metadata,
		); !errors.Is(
		err,
		ErrInvalidCSRFToken,
	) {

		t.Fatalf(
			"logout with wrong CSRF token: expected %v, got %v",
			ErrInvalidCSRFToken,
			err,
		)
	}

	if _, err :=
		f.service.AuthenticateAccessToken(
			ctx,
			refreshed.Material.AccessToken,
		); err != nil {

		t.Fatalf(
			"wrong-CSRF logout unexpectedly revoked session: %v",
			err,
		)
	}

	if err :=
		f.service.Logout(
			ctx,
			refreshed.Material.AccessToken,
			refreshed.Material.CSRFToken,
			metadata,
		); err != nil {

		t.Fatalf(
			"logout Admin session: %v",
			err,
		)
	}

	_, err =
		f.service.AuthenticateAccessToken(
			ctx,
			refreshed.Material.AccessToken,
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrSessionRevoked,
		"access token after logout",
	)

	_, err =
		f.service.Refresh(
			ctx,
			refreshed.Material.RefreshToken,
			refreshed.Material.CSRFToken,
			metadata,
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrSessionRevoked,
		"refresh token after logout",
	)

	if err :=
		f.service.Logout(
			ctx,
			refreshed.Material.AccessToken,
			refreshed.Material.CSRFToken,
			metadata,
		); err != nil {

		t.Fatalf(
			"idempotent Admin logout: %v",
			err,
		)
	}
}

func TestAdminAuthSessionDBRecoveryCodeSingleUse(
	t *testing.T,
) {
	f :=
		newStaffActivationDBFixture(
			t,
		)

	ctx :=
		context.Background()

	staff,
		_,
		recoveryCodes :=
		prepareAdminAuthSessionStaff(
			t,
			f,
			"recovery-code",
			adminAuthSessionTestPassword,
		)

	if len(recoveryCodes) < 2 {
		t.Fatalf(
			"expected at least two recovery codes, got %d",
			len(recoveryCodes),
		)
	}

	metadata :=
		f.metadata()

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
		t.Fatalf(
			"password login for recovery-code test: %v",
			err,
		)
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
		t.Fatalf(
			"first recovery-code login: %v",
			err,
		)
	}

	if session.Material.AccessToken == "" {
		t.Fatal(
			"recovery-code login did not create an Admin session",
		)
	}

	firstHash, err :=
		HashRecoveryCode(
			recoveryCodes[0],
		)
	if err != nil {
		t.Fatalf(
			"hash used recovery code: %v",
			err,
		)
	}

	var usedAt *time.Time

	if err :=
		f.db.QueryRow(
			ctx,
			`
				SELECT used_at
				FROM admin_mfa_recovery_codes
				WHERE
					staff_account_id = $1::uuid
					AND code_hash = $2
			`,
			staff.ID,
			firstHash,
		).Scan(
			&usedAt,
		); err != nil {

		t.Fatalf(
			"load consumed recovery code: %v",
			err,
		)
	}

	if usedAt == nil {
		t.Fatal(
			"successful recovery-code login did not persist used_at",
		)
	}

	secondChallenge, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: staff.Email,
				Password:   adminAuthSessionTestPassword,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"second password login for recovery-code test: %v",
			err,
		)
	}

	_, err =
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: secondChallenge.ChallengeToken,
				Method:         MFAMethodRecoveryCode,
				Code:           recoveryCodes[0],
			},
			metadata,
		)
	assertAdminAuthSessionDBErrorIs(
		t,
		err,
		ErrInvalidMFACode,
		"reuse consumed recovery code",
	)

	if _, err :=
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: secondChallenge.ChallengeToken,
				Method:         MFAMethodRecoveryCode,
				Code:           recoveryCodes[1],
			},
			metadata,
		); err != nil {

		t.Fatalf(
			"unused recovery code after failed replay: %v",
			err,
		)
	}
}

func TestAdminAuthSessionDBPasswordAndMFALockout(
	t *testing.T,
) {
	t.Run(
		"password lockout",
		func(
			t *testing.T,
		) {
			f :=
				newStaffActivationDBFixture(
					t,
				)

			ctx :=
				context.Background()

			staff :=
				f.addStaff(
					"password-lockout",
					"active",
					true,
				)

			setAdminAuthSessionPassword(
				t,
				f,
				staff.ID,
				adminAuthSessionTestPassword,
			)

			metadata :=
				f.metadata()

			for attempt :=
				0; attempt < DefaultAdminLoginMaxFailures; attempt++ {

				_, err :=
					f.service.Login(
						ctx,
						LoginRequest{
							Identifier: staff.Email,
							Password:   "definitely-wrong-password",
						},
						metadata,
					)

				assertAdminAuthSessionDBErrorIs(
					t,
					err,
					ErrInvalidCredentials,
					fmt.Sprintf(
						"password failure %d",
						attempt+1,
					),
				)
			}

			_, err :=
				f.service.Login(
					ctx,
					LoginRequest{
						Identifier: staff.Email,
						Password:   adminAuthSessionTestPassword,
					},
					metadata,
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrLoginBlocked,
				"correct password after password lockout",
			)
		},
	)

	t.Run(
		"MFA lockout",
		func(
			t *testing.T,
		) {
			f :=
				newStaffActivationDBFixture(
					t,
				)

			ctx :=
				context.Background()

			staff,
				_,
				_ :=
				prepareAdminAuthSessionStaff(
					t,
					f,
					"mfa-lockout",
					adminAuthSessionTestPassword,
				)

			metadata :=
				f.metadata()

			for attempt :=
				0; attempt < DefaultAdminMFAMaxFailures; attempt++ {

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
					t.Fatalf(
						"password login before MFA failure %d: %v",
						attempt+1,
						err,
					)
				}

				_, err =
					f.service.VerifyMFA(
						ctx,
						MFAVerifyRequest{
							ChallengeToken: challenge.ChallengeToken,
							Method:         MFAMethodRecoveryCode,
							Code:           "definitely-invalid-recovery-code",
						},
						metadata,
					)
				assertAdminAuthSessionDBErrorIs(
					t,
					err,
					ErrInvalidMFACode,
					fmt.Sprintf(
						"MFA failure %d",
						attempt+1,
					),
				)
			}

			_, err :=
				f.service.Login(
					ctx,
					LoginRequest{
						Identifier: staff.Email,
						Password:   adminAuthSessionTestPassword,
					},
					metadata,
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrMFABlocked,
				"password login after MFA lockout",
			)
		},
	)
}

func TestAdminAuthSessionDBStatusAndChallengeEnforcement(
	t *testing.T,
) {
	t.Run(
		"non-active accounts and panel permission",
		func(
			t *testing.T,
		) {
			f :=
				newStaffActivationDBFixture(
					t,
				)

			ctx :=
				context.Background()

			statuses :=
				[]string{
					"pending_activation",
					"suspended",
					"disabled",
					"deleted",
					"banned",
				}

			for index, status := range statuses {

				staff :=
					f.addStaff(
						"status-"+status,
						status,
						true,
					)

				setAdminAuthSessionPassword(
					t,
					f,
					staff.ID,
					adminAuthSessionTestPassword,
				)

				metadata :=
					f.metadata()

				metadata.IPAddress =
					fmt.Sprintf(
						"192.0.2.%d",
						index+10,
					)

				_, err :=
					f.service.Login(
						ctx,
						LoginRequest{
							Identifier: staff.Email,
							Password:   adminAuthSessionTestPassword,
						},
						metadata,
					)

				assertAdminAuthSessionDBErrorIs(
					t,
					err,
					ErrInvalidCredentials,
					"login with status "+status,
				)
			}

			noPanel :=
				f.addStaff(
					"no-panel-permission",
					"active",
					false,
				)

			setAdminAuthSessionPassword(
				t,
				f,
				noPanel.ID,
				adminAuthSessionTestPassword,
			)

			metadata :=
				f.metadata()

			metadata.IPAddress =
				"192.0.2.99"

			_, err :=
				f.service.Login(
					ctx,
					LoginRequest{
						Identifier: noPanel.Email,
						Password:   adminAuthSessionTestPassword,
					},
					metadata,
				)

			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrInvalidCredentials,
				"active account without Admin panel permission",
			)
		},
	)

	t.Run(
		"existing session observes status and permission changes",
		func(
			t *testing.T,
		) {
			f :=
				newStaffActivationDBFixture(
					t,
				)

			ctx :=
				context.Background()

			staff,
				secret,
				_ :=
				prepareAdminAuthSessionStaff(
					t,
					f,
					"live-account-state",
					adminAuthSessionTestPassword,
				)

			f.now =
				f.now.Add(
					time.Duration(
						DefaultTOTPPeriodSeconds,
					) * time.Second,
				)

			session :=
				mustAdminAuthSessionLoginWithTOTP(
					t,
					f,
					staff,
					adminAuthSessionTestPassword,
					secret,
				)

			f.exec(
				`
					UPDATE staff_accounts
					SET
						status = 'suspended',
						updated_at = now()
					WHERE id = $1::uuid
				`,
				staff.ID,
			)

			_, err :=
				f.service.AuthenticateAccessToken(
					ctx,
					session.Material.AccessToken,
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrAdminAccountDisabled,
				"access token after staff suspension",
			)

			_, err =
				f.service.Refresh(
					ctx,
					session.Material.RefreshToken,
					session.Material.CSRFToken,
					f.metadata(),
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrAdminAccountDisabled,
				"refresh token after staff suspension",
			)

			f.exec(
				`
					UPDATE staff_accounts
					SET
						status = 'active',
						updated_at = now()
					WHERE id = $1::uuid
				`,
				staff.ID,
			)

			f.exec(
				`
					DELETE FROM staff_account_roles
					WHERE staff_account_id = $1::uuid
				`,
				staff.ID,
			)

			_, err =
				f.service.AuthenticateAccessToken(
					ctx,
					session.Material.AccessToken,
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrAdminPanelAccessRequired,
				"access token after Admin panel permission removal",
			)

			_, err =
				f.service.Refresh(
					ctx,
					session.Material.RefreshToken,
					session.Material.CSRFToken,
					f.metadata(),
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrAdminPanelAccessRequired,
				"refresh token after Admin panel permission removal",
			)
		},
	)

	t.Run(
		"locked and expired challenges",
		func(
			t *testing.T,
		) {
			f :=
				newStaffActivationDBFixture(
					t,
				)

			ctx :=
				context.Background()

			staff,
				_,
				_ :=
				prepareAdminAuthSessionStaff(
					t,
					f,
					"challenge-state",
					adminAuthSessionTestPassword,
				)

			metadata :=
				f.metadata()

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
				t.Fatalf(
					"password login for challenge-lock test: %v",
					err,
				)
			}

			for attempt :=
				1; attempt <= AdminChallengeMaxAttempts; attempt++ {

				_, err =
					f.service.VerifyMFA(
						ctx,
						MFAVerifyRequest{
							ChallengeToken: challenge.ChallengeToken,
							Method:         MFAMethodRecoveryCode,
							Code:           "definitely-invalid-recovery-code",
						},
						metadata,
					)

				expected :=
					ErrInvalidMFACode

				if attempt ==
					AdminChallengeMaxAttempts {

					expected =
						ErrChallengeLocked
				}

				assertAdminAuthSessionDBErrorIs(
					t,
					err,
					expected,
					fmt.Sprintf(
						"challenge failure %d",
						attempt,
					),
				)
			}

			_, err =
				f.service.VerifyMFA(
					ctx,
					MFAVerifyRequest{
						ChallengeToken: challenge.ChallengeToken,
						Method:         MFAMethodRecoveryCode,
						Code:           "definitely-invalid-recovery-code",
					},
					metadata,
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrChallengeLocked,
				"replay locked Admin login challenge",
			)

			/*
				Use a different source IP so the challenge-lock failures
				above cannot interfere with the expiration check through
				the Redis MFA-IP dimension.
			*/
			expiryMetadata :=
				f.metadata()

			expiryMetadata.IPAddress =
				"192.0.2.200"

			expiringChallenge, err :=
				f.service.Login(
					ctx,
					LoginRequest{
						Identifier: staff.Email,
						Password:   adminAuthSessionTestPassword,
					},
					expiryMetadata,
				)
			if err != nil {
				t.Fatalf(
					"password login for challenge-expiry test: %v",
					err,
				)
			}

			f.now =
				expiringChallenge.ChallengeExpiresAt.Add(
					time.Second,
				)

			_, err =
				f.service.VerifyMFA(
					ctx,
					MFAVerifyRequest{
						ChallengeToken: expiringChallenge.ChallengeToken,
						Method:         MFAMethodRecoveryCode,
						Code:           "definitely-invalid-recovery-code",
					},
					expiryMetadata,
				)
			assertAdminAuthSessionDBErrorIs(
				t,
				err,
				ErrChallengeExpired,
				"expired Admin login challenge",
			)

			var status string

			if err :=
				f.db.QueryRow(
					ctx,
					`
						SELECT status
						FROM admin_login_challenges
						WHERE challenge_token_hash = $1
					`,
					platformsecurity.HashToken(
						expiringChallenge.ChallengeToken,
					),
				).Scan(
					&status,
				); err != nil {

				t.Fatalf(
					"load expired Admin challenge status: %v",
					err,
				)
			}

			if status !=
				ChallengeStatusExpired {

				t.Fatalf(
					"expired Admin challenge status = %q, want %q",
					status,
					ChallengeStatusExpired,
				)
			}
		},
	)
}

func prepareAdminAuthSessionStaff(
	t *testing.T,
	f *staffActivationDBFixture,
	label string,
	password string,
) (
	staffActivationTestStaff,
	string,
	[]string,
) {
	t.Helper()

	staff :=
		f.addStaff(
			label,
			"active",
			true,
		)

	setAdminAuthSessionPassword(
		t,
		f,
		staff.ID,
		password,
	)

	enrollment, err :=
		f.service.mfa.BeginEnrollment(
			context.Background(),
			staff.ID,
			"Admin auth session DB test",
		)
	if err != nil {
		t.Fatalf(
			"begin fixture MFA enrollment: %v",
			err,
		)
	}

	code :=
		mustAdminAuthSessionTOTP(
			t,
			enrollment.Secret,
			f.now,
		)

	confirmation, err :=
		f.service.mfa.ConfirmEnrollment(
			context.Background(),
			staff.ID,
			code,
		)
	if err != nil {
		t.Fatalf(
			"confirm fixture MFA enrollment: %v",
			err,
		)
	}

	return staff,
		enrollment.Secret,
		confirmation.RecoveryCodes
}

func setAdminAuthSessionPassword(
	t *testing.T,
	f *staffActivationDBFixture,
	staffID string,
	password string,
) {
	t.Helper()

	passwordHash, err :=
		HashNewPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"hash Admin auth session fixture password: %v",
			err,
		)
	}

	f.exec(
		`
			UPDATE staff_accounts
			SET
				password_hash = $2,
				updated_at = now()
			WHERE id = $1::uuid
		`,
		staffID,
		passwordHash,
	)
}

func mustAdminAuthSessionLoginWithTOTP(
	t *testing.T,
	f *staffActivationDBFixture,
	staff staffActivationTestStaff,
	password string,
	secret string,
) SessionResult {
	t.Helper()

	ctx :=
		context.Background()

	metadata :=
		f.metadata()

	challenge, err :=
		f.service.Login(
			ctx,
			LoginRequest{
				Identifier: staff.Email,
				Password:   password,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"fixture password login: %v",
			err,
		)
	}

	code :=
		mustAdminAuthSessionTOTP(
			t,
			secret,
			f.now,
		)

	session, err :=
		f.service.VerifyMFA(
			ctx,
			MFAVerifyRequest{
				ChallengeToken: challenge.ChallengeToken,
				Method:         MFAMethodTOTP,
				Code:           code,
			},
			metadata,
		)
	if err != nil {
		t.Fatalf(
			"fixture TOTP login: %v",
			err,
		)
	}

	return session
}

func mustAdminAuthSessionTOTP(
	t *testing.T,
	secret string,
	at time.Time,
) string {
	t.Helper()

	code,
		_,
		err :=
		GenerateTOTP(
			secret,
			at,
			DefaultTOTPConfig(),
		)
	if err != nil {
		t.Fatalf(
			"generate Admin auth session TOTP: %v",
			err,
		)
	}

	return code
}

func assertAdminAuthSessionDBErrorIs(
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

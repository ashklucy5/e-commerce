package adminauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAdminSelfSecurityOverviewDBLifecycle(
	t *testing.T,
) {
	f := newStaffActivationDBFixture(t)
	ctx := context.Background()

	staff,
		secret,
		recoveryCodes :=
		prepareAdminAuthSessionStaff(
			t,
			f,
			"security-overview",
			adminAuthSessionTestPassword,
		)

	f.now =
		f.now.Add(
			time.Duration(DefaultTOTPPeriodSeconds) * time.Second,
		)

	current :=
		mustAdminAuthSessionLoginWithTOTP(
			t,
			f,
			staff,
			adminAuthSessionTestPassword,
			secret,
		)

	f.now =
		f.now.Add(
			time.Duration(DefaultTOTPPeriodSeconds) * time.Second,
		)

	other :=
		mustAdminAuthSessionLoginWithTOTP(
			t,
			f,
			staff,
			adminAuthSessionTestPassword,
			secret,
		)

	overview, err :=
		f.service.SelfSecurityOverview(
			ctx,
			current.Material.AccessToken,
		)
	if err != nil {
		t.Fatalf(
			"load self security overview: %v",
			err,
		)
	}

	if !overview.MFA.Enabled {
		t.Fatal("security overview reported MFA disabled")
	}

	if overview.MFA.Label != "Admin auth session DB test" {
		t.Fatalf(
			"security overview MFA label = %q",
			overview.MFA.Label,
		)
	}

	if overview.MFA.RecoveryCodesRemaining != int64(len(recoveryCodes)) {
		t.Fatalf(
			"security overview recovery codes = %d, want %d",
			overview.MFA.RecoveryCodesRemaining,
			len(recoveryCodes),
		)
	}

	if len(overview.Sessions) != 2 {
		t.Fatalf(
			"security overview active sessions = %d, want 2",
			len(overview.Sessions),
		)
	}

	var currentSeen bool
	var otherSeen bool

	for _, session := range overview.Sessions {
		switch session.ID {
		case current.Principal.SessionID:
			if !session.Current {
				t.Fatal("current Admin session was not marked current")
			}

			currentSeen = true

		case other.Principal.SessionID:
			if session.Current {
				t.Fatal("other Admin session was marked current")
			}

			otherSeen = true
		}
	}

	if !currentSeen || !otherSeen {
		t.Fatalf(
			"security overview session membership current=%v other=%v",
			currentSeen,
			otherSeen,
		)
	}

	err =
		f.service.RevokeSelfSecuritySession(
			ctx,
			current.Material.AccessToken,
			current.Material.CSRFToken,
			current.Principal.SessionID,
			f.metadata(),
		)
	if !errors.Is(
		err,
		ErrCannotRevokeCurrentAdminSession,
	) {
		t.Fatalf(
			"revoke current Admin session: expected %v, got %v",
			ErrCannotRevokeCurrentAdminSession,
			err,
		)
	}

	err =
		f.service.RevokeSelfSecuritySession(
			ctx,
			current.Material.AccessToken,
			"wrong-csrf-token",
			other.Principal.SessionID,
			f.metadata(),
		)
	if !errors.Is(
		err,
		ErrInvalidCSRFToken,
	) {
		t.Fatalf(
			"revoke other Admin session with wrong CSRF: expected %v, got %v",
			ErrInvalidCSRFToken,
			err,
		)
	}

	if err :=
		f.service.RevokeSelfSecuritySession(
			ctx,
			current.Material.AccessToken,
			current.Material.CSRFToken,
			other.Principal.SessionID,
			f.metadata(),
		); err != nil {

		t.Fatalf(
			"revoke other Admin session: %v",
			err,
		)
	}

	_, err =
		f.service.AuthenticateAccessToken(
			ctx,
			other.Material.AccessToken,
		)
	if !errors.Is(
		err,
		ErrSessionRevoked,
	) {
		t.Fatalf(
			"revoked other Admin session: expected %v, got %v",
			ErrSessionRevoked,
			err,
		)
	}

	if _, err :=
		f.service.AuthenticateAccessToken(
			ctx,
			current.Material.AccessToken,
		); err != nil {

		t.Fatalf(
			"current Admin session after revoking other session: %v",
			err,
		)
	}

	overview, err =
		f.service.SelfSecurityOverview(
			ctx,
			current.Material.AccessToken,
		)
	if err != nil {
		t.Fatalf(
			"reload self security overview: %v",
			err,
		)
	}

	if len(overview.Sessions) != 1 ||
		!overview.Sessions[0].Current {

		t.Fatalf(
			"security overview after revocation = %+v, want only current session",
			overview.Sessions,
		)
	}
}

func TestAdminSelfSecurityOverviewDBRevokeOtherSessions(
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
			"security-revoke-others",
			adminAuthSessionTestPassword,
		)

	login :=
		func() SessionResult {
			f.now =
				f.now.Add(
					time.Duration(DefaultTOTPPeriodSeconds) * time.Second,
				)

			return mustAdminAuthSessionLoginWithTOTP(
				t,
				f,
				staff,
				adminAuthSessionTestPassword,
				secret,
			)
		}

	current := login()
	otherOne := login()
	otherTwo := login()

	result, err :=
		f.service.RevokeOtherSelfSecuritySessions(
			ctx,
			current.Material.AccessToken,
			current.Material.CSRFToken,
			f.metadata(),
		)
	if err != nil {
		t.Fatalf(
			"revoke other Admin sessions: %v",
			err,
		)
	}

	if result.RevokedSessions != 2 {
		t.Fatalf(
			"revoked Admin sessions = %d, want 2",
			result.RevokedSessions,
		)
	}

	for _, session := range []SessionResult{
		otherOne,
		otherTwo,
	} {

		_, err :=
			f.service.AuthenticateAccessToken(
				ctx,
				session.Material.AccessToken,
			)
		if !errors.Is(
			err,
			ErrSessionRevoked,
		) {
			t.Fatalf(
				"other Admin session %s: expected %v, got %v",
				session.Principal.SessionID,
				ErrSessionRevoked,
				err,
			)
		}
	}

	if _, err :=
		f.service.AuthenticateAccessToken(
			ctx,
			current.Material.AccessToken,
		); err != nil {

		t.Fatalf(
			"current Admin session after revoke others: %v",
			err,
		)
	}

	result, err =
		f.service.RevokeOtherSelfSecuritySessions(
			ctx,
			current.Material.AccessToken,
			current.Material.CSRFToken,
			f.metadata(),
		)
	if err != nil {
		t.Fatalf(
			"repeat revoke other Admin sessions: %v",
			err,
		)
	}

	if result.RevokedSessions != 0 {
		t.Fatalf(
			"repeat revoke other Admin sessions = %d, want 0",
			result.RevokedSessions,
		)
	}
}

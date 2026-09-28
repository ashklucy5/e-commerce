package adminauth

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	auth :=
		group.Group(
			"/auth",
		)

	/*
		Public staff onboarding.

		These routes do not require an existing Admin session.
		The one-time activation token plus the bound staff
		email are the onboarding capability.

		Successful activation does not create a session.
	*/
	auth.POST(
		"/activate/password",
		handler.SetActivationPassword,
	)

	auth.POST(
		"/activate/mfa/enroll",
		handler.BeginActivationMFA,
	)

	auth.POST(
		"/activate/mfa/confirm",
		handler.ConfirmActivationMFA,
	)

	/*
		Normal Admin authentication.
	*/
	auth.POST(
		"/login",
		handler.Login,
	)

	auth.POST(
		"/mfa/enroll",
		handler.BeginMFAEnrollment,
	)

	auth.POST(
		"/mfa/enroll/confirm",
		handler.ConfirmMFAEnrollment,
	)

	auth.POST(
		"/mfa/verify",
		handler.VerifyMFA,
	)

	/*
		Recovery-session password reset.

		This is not an anonymous forgot-password endpoint. The
		handler requires the existing Admin access cookie, the
		matching CSRF token and proof that this exact Admin session
		was established with password + a one-time recovery code.
	*/
	auth.POST(
		"/recovery/password-reset",
		handler.ResetPasswordFromRecoverySession,
	)

	/*
		Authenticated self-service security operations.

		MFA rotation preserves the current authenticator until the
		new authenticator has been confirmed successfully. Recovery
		codes are returned only when freshly regenerated.
	*/
	auth.GET(
		"/security/overview",
		handler.SelfSecurityOverview,
	)

	auth.DELETE(
		"/security/sessions/:session_id",
		handler.RevokeSelfSecuritySession,
	)

	auth.POST(
		"/security/sessions/revoke-others",
		handler.RevokeOtherSelfSecuritySessions,
	)

	auth.POST(
		"/security/mfa/rotation/begin",
		handler.BeginSelfMFARotation,
	)

	auth.POST(
		"/security/mfa/rotation/confirm",
		handler.ConfirmSelfMFARotation,
	)

	auth.POST(
		"/security/recovery-codes/regenerate",
		handler.RegenerateSelfRecoveryCodes,
	)

	auth.POST(
		"/security/password/change",
		handler.ChangeSelfPassword,
	)

	auth.POST(
		"/refresh",
		handler.Refresh,
	)

	auth.POST(
		"/logout",
		handler.Logout,
	)

	auth.GET(
		"/me",
		handler.Me,
	)
}

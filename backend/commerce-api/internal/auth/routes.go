package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	middleware ...gin.HandlerFunc,
) {
	authGroup :=
		group.Group(
			"/auth",
		)

	if len(
		middleware,
	) > 0 {

		authGroup.Use(
			middleware...,
		)
	}

	authGroup.POST(
		"/register",
		handler.Register,
	)

	authGroup.POST(
		"/register/verify",
		handler.VerifyRegistration,
	)

	authGroup.POST(
		"/register/resend",
		handler.ResendRegistration,
	)

	authGroup.POST(
		"/login",
		handler.Login,
	)

	authGroup.POST(
		"/refresh",
		handler.Refresh,
	)

	authGroup.POST(
		"/logout",
		handler.Logout,
	)

	authGroup.POST(
		"/password/forgot",
		handler.ForgotPassword,
	)

	authGroup.POST(
		"/password/forgot/verify",
		handler.VerifyForgotPassword,
	)

	authGroup.POST(
		"/password/forgot/resend",
		handler.ResendForgotPassword,
	)

	authGroup.POST(
		"/password/reset",
		handler.ResetPassword,
	)

	authGroup.POST(
		"/password/change",
		handler.ChangePassword,
	)

	customerSecurity :=
		group.Group(
			"/customers/me",
		)

	customerSecurity.Use(
		RequireAuth(
			handler.service,
		),
	)

	customerSecurity.GET(
		"/security",
		handler.GetCustomerSecurityOverview,
	)

	customerSecurity.GET(
		"/sessions",
		handler.ListCustomerSessions,
	)

	customerSecurity.POST(
		"/sessions/revoke-others",
		handler.RevokeOtherCustomerSessions,
	)

	customerSecurity.DELETE(
		"/sessions/:session_id",
		handler.RevokeCustomerSession,
	)
}

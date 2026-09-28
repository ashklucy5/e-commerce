package adminauth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const adminPrincipalContextKey = "adminauth.principal"

func SetPrincipal(
	c *gin.Context,
	principal AdminPrincipal,
) {
	if c == nil {
		return
	}

	c.Set(
		adminPrincipalContextKey,
		principal,
	)
}

func PrincipalFromContext(
	c *gin.Context,
) (
	AdminPrincipal,
	bool,
) {
	if c == nil {
		return AdminPrincipal{}, false
	}

	value, exists :=
		c.Get(
			adminPrincipalContextKey,
		)
	if !exists {
		return AdminPrincipal{}, false
	}

	principal, ok :=
		value.(AdminPrincipal)
	if !ok ||
		strings.TrimSpace(
			principal.Staff.ID,
		) == "" {
		return AdminPrincipal{}, false
	}

	return principal, true
}

func ActorIDFromContext(
	c *gin.Context,
) (
	string,
	bool,
) {
	principal, ok :=
		PrincipalFromContext(
			c,
		)
	if !ok {
		return "", false
	}

	actorID :=
		strings.TrimSpace(
			principal.Staff.StaffCode,
		)

	if actorID == "" {
		return "", false
	}

	return actorID, true
}

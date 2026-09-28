package middleware

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

type AdminNetworkGuard struct {
	networks []*net.IPNet
}

func NewAdminNetworkGuard(
	allowedCIDRs []string,
) (
	gin.HandlerFunc,
	error,
) {
	guard :=
		AdminNetworkGuard{
			networks: make(
				[]*net.IPNet,
				0,
				len(allowedCIDRs),
			),
		}

	for _, raw := range allowedCIDRs {

		value :=
			strings.TrimSpace(
				raw,
			)
		if value == "" {
			continue
		}

		if !strings.Contains(
			value,
			"/",
		) {
			ip :=
				net.ParseIP(
					value,
				)
			if ip == nil {
				return nil,
					fmt.Errorf(
						"invalid Admin allowed IP %q",
						value,
					)
			}

			if ip.To4() != nil {
				value += "/32"
			} else {
				value += "/128"
			}
		}

		_, network, err :=
			net.ParseCIDR(
				value,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"invalid Admin allowed CIDR %q: %w",
					value,
					err,
				)
		}

		guard.networks =
			append(
				guard.networks,
				network,
			)
	}

	return guard.Handler(), nil
}

func (g AdminNetworkGuard) Handler() gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		if len(g.networks) == 0 {
			c.Next()

			return
		}

		clientIP :=
			net.ParseIP(
				strings.TrimSpace(
					c.ClientIP(),
				),
			)
		if clientIP == nil {
			platformerrors.Abort(
				c,
				platformerrors.Forbidden(
					"ADMIN_NETWORK_DENIED",
					"Admin access is not allowed from this network",
				),
			)

			return
		}

		for _, network := range g.networks {

			if network.Contains(
				clientIP,
			) {
				c.Next()

				return
			}
		}

		platformerrors.Abort(
			c,
			platformerrors.Forbidden(
				"ADMIN_NETWORK_DENIED",
				"Admin access is not allowed from this network",
			),
		)
	}
}

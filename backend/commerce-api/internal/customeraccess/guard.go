package customeraccess

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"project.local/commerce-api/internal/auth"
)

const GuestCheckoutKeyHeader = "X-Checkout-Key"

const maxCheckoutKeyLength = 160

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

const ownershipQuery = `
	WITH target_order AS (
		SELECT o.id AS order_id
		FROM orders o
		WHERE $1 = 'order' AND o.id = $2::uuid

		UNION ALL

		SELECT r.order_id
		FROM returns r
		WHERE $1 = 'return' AND r.id = $2::uuid

		UNION ALL

		SELECT r.order_id
		FROM refunds r
		WHERE $1 = 'refund' AND r.id = $2::uuid
	)
	SELECT EXISTS (
		SELECT 1
		FROM target_order target
		JOIN orders o
			ON o.id = target.order_id
		LEFT JOIN checkout_sessions cs
			ON cs.id = o.checkout_id
		WHERE
			(
				o.customer_id IS NOT NULL
				AND o.customer_id = NULLIF($3, '')::uuid
			)
			OR (
				o.customer_id IS NULL
				AND $4 <> ''
				AND cs.checkout_key = $4
			)
	)
`

type queryRower interface {
	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

type Guard struct {
	db queryRower
}

func NewGuard(
	db *pgxpool.Pool,
) *Guard {
	return &Guard{
		db: db,
	}
}

type accessTarget struct {
	kind       string
	param      string
	errorCode  string
	errorLabel string
}

var protectedRoutes = map[string]accessTarget{
	http.MethodGet + " /api/v1/orders/:order_id": {
		kind:       "order",
		param:      "order_id",
		errorCode:  "ORDER_NOT_FOUND",
		errorLabel: "Order",
	},

	http.MethodPost + " /api/v1/orders/:order_id/cancel": {
		kind:       "order",
		param:      "order_id",
		errorCode:  "ORDER_NOT_FOUND",
		errorLabel: "Order",
	},

	http.MethodGet + " /api/v1/orders/:order_id/timeline": {
		kind:       "order",
		param:      "order_id",
		errorCode:  "ORDER_NOT_FOUND",
		errorLabel: "Order",
	},

	http.MethodGet + " /api/v1/orders/:order_id/invoice": {
		kind:       "order",
		param:      "order_id",
		errorCode:  "ORDER_NOT_FOUND",
		errorLabel: "Order",
	},

	http.MethodPost + " /api/v1/orders/:order_id/returns": {
		kind:       "order",
		param:      "order_id",
		errorCode:  "RETURN_NOT_FOUND",
		errorLabel: "Return",
	},

	http.MethodGet + " /api/v1/returns/:return_id": {
		kind:       "return",
		param:      "return_id",
		errorCode:  "RETURN_NOT_FOUND",
		errorLabel: "Return",
	},

	http.MethodGet + " /api/v1/returns/:return_id/timeline": {
		kind:       "return",
		param:      "return_id",
		errorCode:  "RETURN_NOT_FOUND",
		errorLabel: "Return",
	},

	http.MethodPost + " /api/v1/returns/:return_id/cancel": {
		kind:       "return",
		param:      "return_id",
		errorCode:  "RETURN_NOT_FOUND",
		errorLabel: "Return",
	},

	http.MethodGet + " /api/v1/refunds/:refund_id": {
		kind:       "refund",
		param:      "refund_id",
		errorCode:  "REFUND_NOT_FOUND",
		errorLabel: "Refund",
	},

	http.MethodGet + " /api/v1/refunds/:refund_id/timeline": {
		kind:       "refund",
		param:      "refund_id",
		errorCode:  "REFUND_NOT_FOUND",
		errorLabel: "Refund",
	},
}

func (g *Guard) Middleware() gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		target, ok :=
			protectedRoutes[c.Request.Method+" "+c.FullPath()]

		if !ok {
			c.Next()

			return
		}

		resourceID :=
			strings.TrimSpace(
				c.Param(
					target.param,
				),
			)

		// Keep malformed UUID handling in the existing handlers.
		// This middleware only decides ownership for valid resource IDs.
		if !uuidPattern.MatchString(
			resourceID,
		) {
			c.Next()

			return
		}

		customerID, _ :=
			auth.CustomerIDFromContext(
				c,
			)

		checkoutKey :=
			normalizeCheckoutKey(
				c.GetHeader(
					GuestCheckoutKeyHeader,
				),
			)

		authorized, err :=
			g.authorized(
				c.Request.Context(),
				target.kind,
				resourceID,
				customerID,
				checkoutKey,
			)
		if err != nil {
			_ =
				c.Error(
					fmt.Errorf(
						"customer access authorization failed: %w",
						err,
					),
				)

			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code": "INTERNAL_ERROR",

						"message": "Unable to authorize request",
					},
				},
			)

			return
		}

		if !authorized {
			c.AbortWithStatusJSON(
				http.StatusNotFound,
				gin.H{
					"error": gin.H{
						"code": target.errorCode,

						"message": target.errorLabel +
							" not found",
					},
				},
			)

			return
		}

		c.Next()
	}
}

func normalizeCheckoutKey(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" ||
		len(value) > maxCheckoutKeyLength ||
		!strings.HasPrefix(
			value,
			"chk_",
		) {

		return ""
	}

	return value
}

func (g *Guard) authorized(
	ctx context.Context,
	kind string,
	resourceID string,
	customerID string,
	checkoutKey string,
) (bool, error) {
	var authorized bool

	if err :=
		g.db.QueryRow(
			ctx,
			ownershipQuery,
			kind,
			resourceID,
			strings.TrimSpace(
				customerID,
			),
			checkoutKey,
		).Scan(
			&authorized,
		); err != nil {

		return false,
			fmt.Errorf(
				"query %s ownership: %w",
				kind,
				err,
			)
	}

	return authorized, nil
}

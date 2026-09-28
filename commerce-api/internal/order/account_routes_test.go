package order

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccountOrderHistoryRouteRequiresMiddleware(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	engine :=
		gin.New()

	api :=
		engine.Group(
			"/api/v1",
		)

	RegisterAccountRoutes(
		api,
		&Handler{},
		func(
			c *gin.Context,
		) {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": gin.H{
						"code": "TEST_AUTH_REQUIRED",
					},
				},
			)
		},
	)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/orders",
			nil,
		)

	response :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		response,
		request,
	)

	if response.Code !=
		http.StatusUnauthorized {

		t.Fatalf(
			"expected protected order history route to return 401, got %d",
			response.Code,
		)
	}
}

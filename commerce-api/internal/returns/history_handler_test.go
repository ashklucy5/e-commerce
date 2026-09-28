package returns

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReturnHistoryRequiresAuthentication(
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

	RegisterRoutes(
		api,
		&Handler{},
	)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/returns",
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
			"expected return history to require authentication, got HTTP %d",
			response.Code,
		)
	}
}

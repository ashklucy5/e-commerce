package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCommerceCORSScope(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	corsMiddleware, err :=
		newCommerceCORS(
			[]string{
				"https://shop.example.com",
			},
			[]string{
				"https://admin.example.com",
			},
		)
	if err != nil {
		t.Fatalf(
			"create CORS middleware: %v",
			err,
		)
	}

	engine :=
		gin.New()

	engine.Use(
		corsMiddleware,
	)

	for _, route := range []string{
		"/api/v1/products",
		"/api/v1/admin/dashboard",
		"/api/v1/support/me",
	} {

		engine.OPTIONS(
			route,
			func(
				c *gin.Context,
			) {
				c.Status(
					http.StatusNoContent,
				)
			},
		)
	}

	t.Run(
		"storefront customer API with checkout key",
		func(
			t *testing.T,
		) {
			response :=
				preflight(
					engine,
					"/api/v1/products",
					"https://shop.example.com",
					"GET",
					"Authorization, X-Checkout-Key",
				)

			if response.Code !=
				http.StatusNoContent {

				t.Fatalf(
					"expected 204, got %d",
					response.Code,
				)
			}

			if got :=
				response.Header().
					Get(
						"Access-Control-Allow-Origin",
					); got !=
				"https://shop.example.com" {

				t.Fatalf(
					"unexpected allow origin %q",
					got,
				)
			}

			if got :=
				response.Header().
					Get(
						"Access-Control-Allow-Credentials",
					); got != "" {

				t.Fatalf(
					"customer CORS must not allow credentials, got %q",
					got,
				)
			}
		},
	)

	t.Run(
		"storefront cannot preflight Admin API",
		func(
			t *testing.T,
		) {
			response :=
				preflight(
					engine,
					"/api/v1/admin/dashboard",
					"https://shop.example.com",
					"GET",
					"X-CSRF-Token",
				)

			if response.Code !=
				http.StatusForbidden {

				t.Fatalf(
					"expected 403, got %d",
					response.Code,
				)
			}
		},
	)

	t.Run(
		"Admin origin can use Admin API with credentials",
		func(
			t *testing.T,
		) {
			response :=
				preflight(
					engine,
					"/api/v1/admin/dashboard",
					"https://admin.example.com",
					"PATCH",
					"Content-Type, X-CSRF-Token",
				)

			if response.Code !=
				http.StatusNoContent {

				t.Fatalf(
					"expected 204, got %d",
					response.Code,
				)
			}

			if got :=
				response.Header().
					Get(
						"Access-Control-Allow-Credentials",
					); got != "true" {

				t.Fatalf(
					"expected credentialed Admin CORS, got %q",
					got,
				)
			}
		},
	)

	t.Run(
		"support uses Admin origin boundary",
		func(
			t *testing.T,
		) {
			response :=
				preflight(
					engine,
					"/api/v1/support/me",
					"https://admin.example.com",
					"GET",
					"Authorization",
				)

			if response.Code !=
				http.StatusNoContent {

				t.Fatalf(
					"expected 204, got %d",
					response.Code,
				)
			}
		},
	)

	t.Run(
		"Admin origin cannot preflight customer API",
		func(
			t *testing.T,
		) {
			response :=
				preflight(
					engine,
					"/api/v1/products",
					"https://admin.example.com",
					"GET",
					"Authorization",
				)

			if response.Code !=
				http.StatusForbidden {

				t.Fatalf(
					"expected 403, got %d",
					response.Code,
				)
			}
		},
	)
}

func preflight(
	engine http.Handler,
	path string,
	origin string,
	method string,
	headers string,
) *httptest.ResponseRecorder {
	request :=
		httptest.NewRequest(
			http.MethodOptions,
			path,
			nil,
		)

	request.Header.Set(
		"Origin",
		origin,
	)

	request.Header.Set(
		"Access-Control-Request-Method",
		method,
	)

	if headers != "" {
		request.Header.Set(
			"Access-Control-Request-Headers",
			headers,
		)
	}

	response :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		response,
		request,
	)

	return response
}

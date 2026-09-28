package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	cfg :=
		DefaultSecurityHeadersConfig()

	cfg.EnableHSTS =
		true

	cfg.HSTSMaxAge =
		24 * time.Hour

	cfg.HSTSIncludeSubDomains =
		true

	engine :=
		gin.New()

	engine.Use(
		SecurityHeaders(
			cfg,
		),
	)

	engine.GET(
		"/ping",
		func(
			c *gin.Context,
		) {
			c.JSON(
				http.StatusOK,
				gin.H{
					"data": "ok",
				},
			)
		},
	)

	response :=
		httptest.NewRecorder()

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/ping",
			nil,
		)

	engine.ServeHTTP(
		response,
		request,
	)

	if actual :=
		response.Header().
			Get(
				"X-Content-Type-Options",
			); actual !=
		"nosniff" {

		t.Fatalf(
			"unexpected X-Content-Type-Options %q",
			actual,
		)
	}

	if actual :=
		response.Header().
			Get(
				"X-Frame-Options",
			); actual !=
		"DENY" {

		t.Fatalf(
			"unexpected X-Frame-Options %q",
			actual,
		)
	}

	if actual :=
		response.Header().
			Get(
				"Content-Security-Policy",
			); actual == "" {

		t.Fatal(
			"expected Content-Security-Policy",
		)
	}

	hsts :=
		response.Header().
			Get(
				"Strict-Transport-Security",
			)

	if !strings.Contains(
		hsts,
		"max-age=86400",
	) {
		t.Fatalf(
			"unexpected HSTS header %q",
			hsts,
		)
	}

	if !strings.Contains(
		hsts,
		"includeSubDomains",
	) {
		t.Fatalf(
			"expected includeSubDomains, got %q",
			hsts,
		)
	}
}

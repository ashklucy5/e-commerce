package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSPreflightAllowed(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	cfg :=
		DefaultCORSConfig()

	cfg.AllowedOrigins =
		[]string{
			"https://admin.example.com",
		}

	cfg.AllowCredentials =
		true

	handler, err :=
		NewCORS(
			cfg,
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
		handler,
	)

	engine.POST(
		"/admin",
		func(
			c *gin.Context,
		) {
			c.Status(
				http.StatusNoContent,
			)
		},
	)

	request :=
		httptest.NewRequest(
			http.MethodOptions,
			"/admin",
			nil,
		)

	request.Header.Set(
		"Origin",
		"https://admin.example.com",
	)

	request.Header.Set(
		"Access-Control-Request-Method",
		http.MethodPost,
	)

	request.Header.Set(
		"Access-Control-Request-Headers",
		"Content-Type, X-CSRF-Token",
	)

	response :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		response,
		request,
	)

	if response.Code !=
		http.StatusNoContent {
		t.Fatalf(
			"expected status 204, got %d",
			response.Code,
		)
	}

	if actual :=
		response.Header().
			Get(
				"Access-Control-Allow-Origin",
			); actual !=
		"https://admin.example.com" {

		t.Fatalf(
			"unexpected allowed origin %q",
			actual,
		)
	}

	if actual :=
		response.Header().
			Get(
				"Access-Control-Allow-Credentials",
			); actual !=
		"true" {

		t.Fatalf(
			"expected credential header, got %q",
			actual,
		)
	}
}

func TestCORSRejectsUnknownPreflightOrigin(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	cfg :=
		DefaultCORSConfig()

	cfg.AllowedOrigins =
		[]string{
			"https://admin.example.com",
		}

	handler, err :=
		NewCORS(
			cfg,
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
		handler,
	)

	engine.POST(
		"/admin",
		func(
			c *gin.Context,
		) {
			c.Status(
				http.StatusNoContent,
			)
		},
	)

	request :=
		httptest.NewRequest(
			http.MethodOptions,
			"/admin",
			nil,
		)

	request.Header.Set(
		"Origin",
		"https://evil.example",
	)

	request.Header.Set(
		"Access-Control-Request-Method",
		http.MethodPost,
	)

	response :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		response,
		request,
	)

	if response.Code !=
		http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			response.Code,
		)
	}

	if !strings.Contains(
		response.Body.String(),
		`"code":"CORS_ORIGIN_DENIED"`,
	) {
		t.Fatalf(
			"unexpected body %s",
			response.Body.String(),
		)
	}
}

func TestCORSRejectsWildcardWithCredentials(
	t *testing.T,
) {
	cfg :=
		DefaultCORSConfig()

	cfg.AllowedOrigins =
		[]string{
			"*",
		}

	cfg.AllowCredentials =
		true

	_, err :=
		NewCORS(
			cfg,
		)

	if err == nil {
		t.Fatal(
			"expected invalid wildcard credential configuration",
		)
	}
}

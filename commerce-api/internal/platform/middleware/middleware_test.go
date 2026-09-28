package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	platformlogger "project.local/commerce-api/internal/platform/logger"
)

func TestRequestIDPreservesValidIncomingID(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	engine :=
		gin.New()

	engine.Use(
		RequestID(),
	)

	engine.GET(
		"/ping",
		func(
			c *gin.Context,
		) {
			requestID, ok :=
				RequestIDFromContext(
					c,
				)
			if !ok {
				t.Fatal(
					"expected request ID in Gin context",
				)
			}

			contextRequestID, ok :=
				RequestIDFromRequestContext(
					c.Request.Context(),
				)
			if !ok {
				t.Fatal(
					"expected request ID in request context",
				)
			}

			if requestID !=
				contextRequestID {
				t.Fatalf(
					"request ID mismatch: gin=%q request=%q",
					requestID,
					contextRequestID,
				)
			}

			c.Status(
				http.StatusNoContent,
			)
		},
	)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/ping",
			nil,
		)

	request.Header.Set(
		RequestIDHeader,
		"client-request-123",
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
				RequestIDHeader,
			); actual !=
		"client-request-123" {

		t.Fatalf(
			"expected preserved request ID, got %q",
			actual,
		)
	}
}

func TestRequestIDReplacesUnsafeIncomingID(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	engine :=
		gin.New()

	engine.Use(
		RequestID(),
	)

	engine.GET(
		"/ping",
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
			http.MethodGet,
			"/ping",
			nil,
		)

	request.Header.Set(
		RequestIDHeader,
		"unsafe request id",
	)

	response :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		response,
		request,
	)

	actual :=
		response.Header().
			Get(
				RequestIDHeader,
			)

	if actual == "" {
		t.Fatal(
			"expected generated request ID",
		)
	}

	if actual ==
		"unsafe request id" {
		t.Fatal(
			"expected unsafe incoming request ID to be replaced",
		)
	}
}

func TestRecoveryReturnsSafeInternalError(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	engine :=
		gin.New()

	engine.Use(
		RequestID(),
	)

	engine.Use(
		Recovery(),
	)

	engine.GET(
		"/panic",
		func(
			c *gin.Context,
		) {
			panic(
				"sensitive panic detail",
			)
		},
	)

	response :=
		httptest.NewRecorder()

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/panic",
			nil,
		)

	engine.ServeHTTP(
		response,
		request,
	)

	if response.Code !=
		http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			response.Code,
		)
	}

	body :=
		response.Body.String()

	if !strings.Contains(
		body,
		`"code":"INTERNAL_ERROR"`,
	) {
		t.Fatalf(
			"expected internal error response, got %s",
			body,
		)
	}

	if strings.Contains(
		body,
		"sensitive panic detail",
	) {
		t.Fatalf(
			"panic detail leaked to client: %s",
			body,
		)
	}
}

func TestLoggingWritesStructuredRequestRecord(
	t *testing.T,
) {
	gin.SetMode(
		gin.TestMode,
	)

	var output bytes.Buffer

	value :=
		platformlogger.New(
			platformlogger.Config{
				Level: "debug",

				Format: "json",

				Writer: &output,
			},
		)

	engine :=
		gin.New()

	engine.Use(
		RequestID(),
	)

	engine.Use(
		Logging(
			value,
		),
	)

	engine.GET(
		"/created",
		func(
			c *gin.Context,
		) {
			c.JSON(
				http.StatusCreated,
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
			"/created",
			nil,
		)

	request.Header.Set(
		RequestIDHeader,
		"request-log-123",
	)

	engine.ServeHTTP(
		response,
		request,
	)

	logOutput :=
		output.String()

	for _, expected := range []string{
		`"msg":"http request completed"`,
		`"request_id":"request-log-123"`,
		`"method":"GET"`,
		`"route":"/created"`,
		`"status":201`,
	} {

		if !strings.Contains(
			logOutput,
			expected,
		) {
			t.Fatalf(
				"expected log output to contain %q, got %s",
				expected,
				logOutput,
			)
		}
	}
}

package customeraccess

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
)

// -----------------------------------------------------------------------------
// Fake database row/query implementation.
// No PostgreSQL is required for these tests.
// -----------------------------------------------------------------------------

type fakeRow struct {
	authorized bool
	err        error
}

func (r fakeRow) Scan(
	dest ...any,
) error {
	if r.err != nil {
		return r.err
	}

	if len(dest) != 1 {
		panic("fakeRow expected exactly one Scan destination")
	}

	value, ok :=
		dest[0].(*bool)
	if !ok {
		panic("fakeRow expected *bool destination")
	}

	*value =
		r.authorized

	return nil
}

type fakeQueryRower struct {
	allowedKind        string
	allowedResourceID  string
	allowedCustomerID  string
	allowedCheckoutKey string
}

func (f *fakeQueryRower) QueryRow(
	_ context.Context,
	_ string,
	args ...any,
) pgx.Row {
	if len(args) != 4 {
		return fakeRow{
			authorized: false,
		}
	}

	kind, _ :=
		args[0].(string)

	resourceID, _ :=
		args[1].(string)

	customerID, _ :=
		args[2].(string)

	checkoutKey, _ :=
		args[3].(string)

	if kind != f.allowedKind ||
		resourceID != f.allowedResourceID {

		return fakeRow{
			authorized: false,
		}
	}

	// Registered order:
	// only matching authenticated customer may access it.
	if f.allowedCustomerID != "" {
		return fakeRow{
			authorized: customerID ==
				f.allowedCustomerID,
		}
	}

	// Guest order:
	// possession of the correct checkout capability is required.
	return fakeRow{
		authorized: f.allowedCheckoutKey != "" &&
			checkoutKey ==
				f.allowedCheckoutKey,
	}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func projectFile(
	t *testing.T,
	parts ...string,
) string {
	t.Helper()

	_, currentFile, _, ok :=
		runtime.Caller(
			0,
		)
	if !ok {
		t.Fatal(
			"unable to determine test file path",
		)
	}

	base :=
		filepath.Dir(
			currentFile,
		)

	allParts :=
		append(
			[]string{
				base,
			},
			parts...,
		)

	return filepath.Join(
		allParts...,
	)
}

func parseProjectGoFile(
	t *testing.T,
	parts ...string,
) *ast.File {
	t.Helper()

	filename :=
		projectFile(
			t,
			parts...,
		)

	file, err :=
		parser.ParseFile(
			token.NewFileSet(),
			filename,
			nil,
			0,
		)
	if err != nil {
		t.Fatalf(
			"parse %s: %v",
			filename,
			err,
		)
	}

	return file
}

func assertRegisterRoutesGroup(
	t *testing.T,
	packageName string,
	groupName string,
) {
	t.Helper()

	file :=
		parseProjectGoFile(
			t,
			"..",
			"platform",
			"router",
			"public.go",
		)

	found :=
		false

	ast.Inspect(
		file,
		func(
			node ast.Node,
		) bool {
			call, ok :=
				node.(*ast.CallExpr)
			if !ok {
				return true
			}

			selector, ok :=
				call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			receiver, ok :=
				selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			if receiver.Name != packageName ||
				selector.Sel.Name !=
					"RegisterRoutes" {

				return true
			}

			found =
				true

			if len(
				call.Args,
			) == 0 {

				t.Fatalf(
					"%s.RegisterRoutes has no router-group argument",
					packageName,
				)
			}

			group, ok :=
				call.Args[0].(*ast.Ident)
			if !ok {
				t.Fatalf(
					"%s.RegisterRoutes first argument is not an identifier",
					packageName,
				)
			}

			if group.Name !=
				groupName {

				t.Fatalf(
					"%s.RegisterRoutes uses %q; expected %q",
					packageName,
					group.Name,
					groupName,
				)
			}

			return true
		},
	)

	if !found {
		t.Fatalf(
			"%s.RegisterRoutes call not found",
			packageName,
		)
	}
}

func assertRouterUsesTrustedProxies(
	t *testing.T,
) {
	t.Helper()

	file :=
		parseProjectGoFile(
			t,
			"..",
			"platform",
			"router",
			"router.go",
		)

	hasTrustedProxyField :=
		false

	hasSetTrustedProxiesCall :=
		false

	ast.Inspect(
		file,
		func(
			node ast.Node,
		) bool {
			switch value :=
				node.(type) {

			case *ast.TypeSpec:
				if value.Name.Name !=
					"Dependencies" {

					return true
				}

				structType, ok :=
					value.Type.(*ast.StructType)
				if !ok {
					return true
				}

				for _, field := range structType.Fields.List {

					for _, name := range field.Names {

						if name.Name ==
							"TrustedProxies" {

							hasTrustedProxyField =
								true
						}
					}
				}

			case *ast.CallExpr:
				selector, ok :=
					value.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				if selector.Sel.Name ==
					"SetTrustedProxies" {

					hasSetTrustedProxiesCall =
						true
				}
			}

			return true
		},
	)

	if !hasTrustedProxyField {
		t.Fatal(
			"router.Dependencies does not contain TrustedProxies",
		)
	}

	if !hasSetTrustedProxiesCall {
		t.Fatal(
			"router.New does not call SetTrustedProxies",
		)
	}
}

func requestThroughGuard(
	t *testing.T,
	method string,
	routeTemplate string,
	actualPath string,
	customerID string,
	checkoutKey string,
	database *fakeQueryRower,
) int {
	t.Helper()

	gin.SetMode(
		gin.TestMode,
	)

	engine :=
		gin.New()

	if customerID != "" {
		engine.Use(
			func(
				c *gin.Context,
			) {
				// auth.CustomerIDFromContext uses this Gin key.
				c.Set(
					"auth_customer_id",
					customerID,
				)

				c.Next()
			},
		)
	}

	guard :=
		&Guard{
			db: database,
		}

	engine.Use(
		guard.Middleware(),
	)

	engine.Handle(
		method,
		routeTemplate,
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
			method,
			actualPath,
			nil,
		)

	if checkoutKey != "" {
		request.Header.Set(
			GuestCheckoutKeyHeader,
			checkoutKey,
		)
	}

	response :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		response,
		request,
	)

	return response.Code
}

// -----------------------------------------------------------------------------
// Main security regression test.
// -----------------------------------------------------------------------------

func TestSecurityFixes(
	t *testing.T,
) {
	const (
		orderID = "11111111-1111-4111-8111-111111111111"

		returnID = "22222222-2222-4222-8222-222222222222"

		refundID = "33333333-3333-4333-8333-333333333333"

		customerA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

		customerB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

		guestCheckoutKey = "chk_security_test_guest_key"

		wrongCheckoutKey = "chk_security_test_wrong_key"
	)

	// -----------------------------------------------------------------
	// FIX #1
	// Trusted proxies
	// -----------------------------------------------------------------

	t.Run(
		"01 trusted proxy wiring exists",
		func(
			t *testing.T,
		) {
			assertRouterUsesTrustedProxies(
				t,
			)
		},
	)

	t.Run(
		"01 spoofed forwarded IP ignored when proxies disabled",
		func(
			t *testing.T,
		) {
			gin.SetMode(
				gin.TestMode,
			)

			engine :=
				gin.New()

			if err :=
				engine.SetTrustedProxies(
					nil,
				); err != nil {

				t.Fatalf(
					"SetTrustedProxies(nil): %v",
					err,
				)
			}

			engine.GET(
				"/client-ip",
				func(
					c *gin.Context,
				) {
					c.String(
						http.StatusOK,
						c.ClientIP(),
					)
				},
			)

			request :=
				httptest.NewRequest(
					http.MethodGet,
					"/client-ip",
					nil,
				)

			request.RemoteAddr =
				"127.0.0.1:54321"

			request.Header.Set(
				"X-Forwarded-For",
				"203.0.113.77",
			)

			response :=
				httptest.NewRecorder()

			engine.ServeHTTP(
				response,
				request,
			)

			if response.Code !=
				http.StatusOK {

				t.Fatalf(
					"unexpected response status: %d",
					response.Code,
				)
			}

			clientIP :=
				strings.TrimSpace(
					response.Body.String(),
				)

			if clientIP ==
				"203.0.113.77" {

				t.Fatal(
					"spoofed X-Forwarded-For was trusted",
				)
			}

			if clientIP !=
				"127.0.0.1" {

				t.Fatalf(
					"expected real client IP 127.0.0.1, got %q",
					clientIP,
				)
			}
		},
	)

	// -----------------------------------------------------------------
	// FIX #2
	// Order ownership
	// -----------------------------------------------------------------

	t.Run(
		"02 registered order accepts owner",
		func(
			t *testing.T,
		) {
			status :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/orders/:order_id",
					"/api/v1/orders/"+orderID,
					customerA,
					"",
					&fakeQueryRower{
						allowedKind: "order",

						allowedResourceID: orderID,

						allowedCustomerID: customerA,
					},
				)

			if status !=
				http.StatusNoContent {

				t.Fatalf(
					"owner should be allowed; got HTTP %d",
					status,
				)
			}
		},
	)

	t.Run(
		"02 registered order rejects another customer",
		func(
			t *testing.T,
		) {
			status :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/orders/:order_id",
					"/api/v1/orders/"+orderID,
					customerB,
					"",
					&fakeQueryRower{
						allowedKind: "order",

						allowedResourceID: orderID,

						allowedCustomerID: customerA,
					},
				)

			if status !=
				http.StatusNotFound {

				t.Fatalf(
					"other customer should receive 404; got HTTP %d",
					status,
				)
			}
		},
	)

	t.Run(
		"02 checkout key cannot bypass registered ownership",
		func(
			t *testing.T,
		) {
			status :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/orders/:order_id",
					"/api/v1/orders/"+orderID,
					"",
					guestCheckoutKey,
					&fakeQueryRower{
						allowedKind: "order",

						allowedResourceID: orderID,

						allowedCustomerID: customerA,
					},
				)

			if status !=
				http.StatusNotFound {

				t.Fatalf(
					"checkout key bypassed registered ownership; got HTTP %d",
					status,
				)
			}
		},
	)

	t.Run(
		"02 guest order requires checkout key",
		func(
			t *testing.T,
		) {
			database :=
				&fakeQueryRower{
					allowedKind: "order",

					allowedResourceID: orderID,

					allowedCheckoutKey: guestCheckoutKey,
				}

			noKey :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/orders/:order_id",
					"/api/v1/orders/"+orderID,
					"",
					"",
					database,
				)

			if noKey !=
				http.StatusNotFound {

				t.Fatalf(
					"guest order without key should return 404; got %d",
					noKey,
				)
			}

			wrongKey :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/orders/:order_id",
					"/api/v1/orders/"+orderID,
					"",
					wrongCheckoutKey,
					database,
				)

			if wrongKey !=
				http.StatusNotFound {

				t.Fatalf(
					"guest order with wrong key should return 404; got %d",
					wrongKey,
				)
			}

			correctKey :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/orders/:order_id",
					"/api/v1/orders/"+orderID,
					"",
					guestCheckoutKey,
					database,
				)

			if correctKey !=
				http.StatusNoContent {

				t.Fatalf(
					"guest order with correct key should be allowed; got %d",
					correctKey,
				)
			}
		},
	)

	// -----------------------------------------------------------------
	// FIX #3 and #4
	// Ensure Returns and Refunds are actually mounted under
	// customerAware, otherwise the Guard middleware would never run.
	// -----------------------------------------------------------------

	t.Run(
		"03 returns use customer aware router group",
		func(
			t *testing.T,
		) {
			assertRegisterRoutesGroup(
				t,
				"returns",
				"customerAware",
			)
		},
	)

	t.Run(
		"04 refunds use customer aware router group",
		func(
			t *testing.T,
		) {
			assertRegisterRoutesGroup(
				t,
				"refund",
				"customerAware",
			)
		},
	)

	t.Run(
		"03 return access requires ownership",
		func(
			t *testing.T,
		) {
			database :=
				&fakeQueryRower{
					allowedKind: "return",

					allowedResourceID: returnID,

					allowedCheckoutKey: guestCheckoutKey,
				}

			wrong :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/returns/:return_id",
					"/api/v1/returns/"+returnID,
					"",
					wrongCheckoutKey,
					database,
				)

			if wrong !=
				http.StatusNotFound {

				t.Fatalf(
					"wrong return credential should receive 404; got %d",
					wrong,
				)
			}

			correct :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/returns/:return_id",
					"/api/v1/returns/"+returnID,
					"",
					guestCheckoutKey,
					database,
				)

			if correct !=
				http.StatusNoContent {

				t.Fatalf(
					"correct return credential should be allowed; got %d",
					correct,
				)
			}
		},
	)

	t.Run(
		"04 refund access requires ownership",
		func(
			t *testing.T,
		) {
			database :=
				&fakeQueryRower{
					allowedKind: "refund",

					allowedResourceID: refundID,

					allowedCheckoutKey: guestCheckoutKey,
				}

			noKey :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/refunds/:refund_id",
					"/api/v1/refunds/"+refundID,
					"",
					"",
					database,
				)

			if noKey !=
				http.StatusNotFound {

				t.Fatalf(
					"refund without credential should receive 404; got %d",
					noKey,
				)
			}

			wrong :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/refunds/:refund_id",
					"/api/v1/refunds/"+refundID,
					"",
					wrongCheckoutKey,
					database,
				)

			if wrong !=
				http.StatusNotFound {

				t.Fatalf(
					"wrong refund credential should receive 404; got %d",
					wrong,
				)
			}

			correct :=
				requestThroughGuard(
					t,
					http.MethodGet,
					"/api/v1/refunds/:refund_id",
					"/api/v1/refunds/"+refundID,
					"",
					guestCheckoutKey,
					database,
				)

			if correct !=
				http.StatusNoContent {

				t.Fatalf(
					"correct refund credential should be allowed; got %d",
					correct,
				)
			}
		},
	)

	// -----------------------------------------------------------------
	// Make sure EVERY sensitive route is covered by the guard.
	// -----------------------------------------------------------------

	t.Run(
		"02 03 04 all sensitive routes are protected",
		func(
			t *testing.T,
		) {
			expected :=
				[]string{
					http.MethodGet +
						" /api/v1/orders/:order_id",

					http.MethodPost +
						" /api/v1/orders/:order_id/cancel",

					http.MethodGet +
						" /api/v1/orders/:order_id/timeline",

					http.MethodGet +
						" /api/v1/orders/:order_id/invoice",

					http.MethodPost +
						" /api/v1/orders/:order_id/returns",

					http.MethodGet +
						" /api/v1/returns/:return_id",

					http.MethodGet +
						" /api/v1/returns/:return_id/timeline",

					http.MethodPost +
						" /api/v1/returns/:return_id/cancel",

					http.MethodGet +
						" /api/v1/refunds/:refund_id",

					http.MethodGet +
						" /api/v1/refunds/:refund_id/timeline",
				}

			for _, route := range expected {

				if _, exists :=
					protectedRoutes[route]; !exists {

					t.Errorf(
						"security guard missing route: %s",
						route,
					)
				}
			}
		},
	)

	// -----------------------------------------------------------------
	// Verify the SQL itself keeps registered and guest authorization
	// logically separate.
	// -----------------------------------------------------------------

	t.Run(
		"02 ownership SQL prevents checkout key bypass",
		func(
			t *testing.T,
		) {
			required :=
				[]string{
					"o.customer_id IS NOT NULL",
					"o.customer_id = NULLIF($3, '')::uuid",
					"o.customer_id IS NULL",
					"cs.checkout_key = $4",
				}

			for _, fragment := range required {

				if !strings.Contains(
					ownershipQuery,
					fragment,
				) {

					t.Errorf(
						"ownership query missing security condition %q",
						fragment,
					)
				}
			}
		},
	)

	// -----------------------------------------------------------------
	// FIX #5
	// Raw cart/checkout secrets must not appear in logs.
	// -----------------------------------------------------------------

	t.Run(
		"05 matched route hides checkout secret from logs",
		func(
			t *testing.T,
		) {
			var logs bytes.Buffer

			logger :=
				slog.New(
					slog.NewJSONHandler(
						&logs,
						nil,
					),
				)

			gin.SetMode(
				gin.TestMode,
			)

			engine :=
				gin.New()

			engine.Use(
				platformmiddleware.RequestID(),
				platformmiddleware.Logging(
					logger,
				),
			)

			engine.GET(
				"/api/v1/checkouts/:checkout_key",
				func(
					c *gin.Context,
				) {
					c.Status(
						http.StatusNoContent,
					)
				},
			)

			const secret = "chk_NEVER_LOG_THIS_SECRET_123456"

			request :=
				httptest.NewRequest(
					http.MethodGet,
					"/api/v1/checkouts/"+secret,
					nil,
				)

			response :=
				httptest.NewRecorder()

			engine.ServeHTTP(
				response,
				request,
			)

			output :=
				logs.String()

			if strings.Contains(
				output,
				secret,
			) {

				t.Fatalf(
					"checkout secret leaked into logs:\n%s",
					output,
				)
			}

			if !strings.Contains(
				output,
				"/api/v1/checkouts/:checkout_key",
			) {

				t.Fatalf(
					"route template missing from log:\n%s",
					output,
				)
			}
		},
	)

	t.Run(
		"05 unmatched route does not log raw secret path",
		func(
			t *testing.T,
		) {
			var logs bytes.Buffer

			logger :=
				slog.New(
					slog.NewJSONHandler(
						&logs,
						nil,
					),
				)

			gin.SetMode(
				gin.TestMode,
			)

			engine :=
				gin.New()

			engine.Use(
				platformmiddleware.RequestID(),
				platformmiddleware.Logging(
					logger,
				),
			)

			const secret = "chk_UNMATCHED_SECRET_987654"

			request :=
				httptest.NewRequest(
					http.MethodGet,
					"/api/v1/does-not-exist/"+secret,
					nil,
				)

			response :=
				httptest.NewRecorder()

			engine.ServeHTTP(
				response,
				request,
			)

			output :=
				logs.String()

			if strings.Contains(
				output,
				secret,
			) {

				t.Fatalf(
					"unmatched URL secret leaked into logs:\n%s",
					output,
				)
			}

			if !strings.Contains(
				output,
				`"route":"unmatched"`,
			) {

				t.Fatalf(
					"expected unmatched route marker:\n%s",
					output,
				)
			}
		},
	)
}

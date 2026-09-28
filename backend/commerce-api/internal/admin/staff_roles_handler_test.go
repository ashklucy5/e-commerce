package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateStaffStatusRejectsDeletedAndBanned(
	t *testing.T,
) {
	t.Parallel()

	gin.SetMode(
		gin.TestMode,
	)

	tests :=
		[]struct {
			name   string
			status string
		}{
			{
				name:   "deleted",
				status: "deleted",
			},
			{
				name:   "banned",
				status: "banned",
			},
		}

	for _, test := range tests {

		test :=
			test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				recorder :=
					httptest.NewRecorder()

				context,
					_ :=
					gin.CreateTestContext(
						recorder,
					)

				context.Params =
					gin.Params{
						{
							Key: "staff_id",

							Value: "00000000-0000-0000-0000-000000000002",
						},
					}

				request :=
					httptest.NewRequest(
						http.MethodPatch,
						"/staff/00000000-0000-0000-0000-000000000002/status",
						strings.NewReader(
							`{"status":"`+
								test.status+
								`"}`,
						),
					)

				request.Header.Set(
					"Content-Type",
					"application/json",
				)

				context.Request =
					request

				handler :=
					&Handler{
						service: &Service{},
					}

				handler.UpdateStaffStatus(
					context,
				)

				if recorder.Code !=
					http.StatusBadRequest {

					t.Fatalf(
						"UpdateStaffStatus(%q) status = %d, want %d; body=%s",
						test.status,
						recorder.Code,
						http.StatusBadRequest,
						recorder.Body.String(),
					)
				}

				assertAdminErrorCode(
					t,
					recorder,
					"INVALID_STAFF_STATUS",
				)
			},
		)
	}
}

func TestUpdateStaffStatusRejectsUnknownState(
	t *testing.T,
) {
	t.Parallel()

	gin.SetMode(
		gin.TestMode,
	)

	recorder :=
		httptest.NewRecorder()

	context,
		_ :=
		gin.CreateTestContext(
			recorder,
		)

	context.Params =
		gin.Params{
			{
				Key: "staff_id",

				Value: "00000000-0000-0000-0000-000000000002",
			},
		}

	request :=
		httptest.NewRequest(
			http.MethodPatch,
			"/staff/00000000-0000-0000-0000-000000000002/status",
			strings.NewReader(
				`{"status":"unknown"}`,
			),
		)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	context.Request =
		request

	handler :=
		&Handler{
			service: &Service{},
		}

	handler.UpdateStaffStatus(
		context,
	)

	if recorder.Code !=
		http.StatusBadRequest {

		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusBadRequest,
			recorder.Body.String(),
		)
	}

	assertAdminErrorCode(
		t,
		recorder,
		"INVALID_STAFF_STATUS",
	)
}

func TestWriteAdminStaffRoleErrorSecurityMappings(
	t *testing.T,
) {
	t.Parallel()

	gin.SetMode(
		gin.TestMode,
	)

	tests :=
		[]struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{
				name: "self password reset",

				err: ErrAdminSelfPasswordReset,

				wantStatus: http.StatusConflict,

				wantCode: "SELF_PASSWORD_RESET_NOT_ALLOWED",
			},
			{
				name: "self MFA reset",

				err: ErrAdminSelfMFAReset,

				wantStatus: http.StatusConflict,

				wantCode: "SELF_MFA_RESET_NOT_ALLOWED",
			},
			{
				name: "self deletion",

				err: ErrAdminSelfDelete,

				wantStatus: http.StatusConflict,

				wantCode: "SELF_DELETE_NOT_ALLOWED",
			},
			{
				name: "deleted staff mutation",

				err: ErrAdminStaffDeleted,

				wantStatus: http.StatusConflict,

				wantCode: "STAFF_ACCOUNT_DELETED",
			},
			{
				name: "banned staff generic mutation",

				err: ErrAdminStaffBanManagedSeparately,

				wantStatus: http.StatusConflict,

				wantCode: "STAFF_ACCOUNT_BANNED",
			},
			{
				name: "protected staff",

				err: ErrAdminProtectedStaffMutation,

				wantStatus: http.StatusForbidden,

				wantCode: "PROTECTED_STAFF_REQUIRES_SUPER_ADMIN",
			},
			{
				name: "protected role assignment",

				err: ErrAdminProtectedRoleAssignment,

				wantStatus: http.StatusForbidden,

				wantCode: "PROTECTED_ROLE_REQUIRES_SUPER_ADMIN",
			},
			{
				name: "disallowed role assignment",

				err: ErrAdminRoleAssignmentNotAllowed,

				wantStatus: http.StatusForbidden,

				wantCode: "ROLE_ASSIGNMENT_NOT_ALLOWED",
			},
			{
				name: "last Super Admin",

				err: ErrAdminLastSuperAdmin,

				wantStatus: http.StatusConflict,

				wantCode: "LAST_SUPER_ADMIN_REQUIRED",
			},
		}

	for _, test := range tests {

		test :=
			test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				recorder :=
					httptest.NewRecorder()

				context,
					_ :=
					gin.CreateTestContext(
						recorder,
					)

				handled :=
					writeAdminStaffRoleError(
						context,
						test.err,
					)

				if !handled {
					t.Fatal(
						"writeAdminStaffRoleError returned false",
					)
				}

				if recorder.Code !=
					test.wantStatus {

					t.Fatalf(
						"status = %d, want %d; body=%s",
						recorder.Code,
						test.wantStatus,
						recorder.Body.String(),
					)
				}

				assertAdminErrorCode(
					t,
					recorder,
					test.wantCode,
				)
			},
		)
	}
}

func TestWriteAdminStaffRoleErrorNil(
	t *testing.T,
) {
	t.Parallel()

	gin.SetMode(
		gin.TestMode,
	)

	recorder :=
		httptest.NewRecorder()

	context,
		_ :=
		gin.CreateTestContext(
			recorder,
		)

	if writeAdminStaffRoleError(
		context,
		nil,
	) {
		t.Fatal(
			"nil error should not be handled",
		)
	}
}

func TestDeleteStaffRouteRequiresDeletePermission(
	t *testing.T,
) {
	t.Parallel()

	gin.SetMode(
		gin.TestMode,
	)

	engine :=
		gin.New()

	handler :=
		&Handler{
			service: &Service{},
		}

	var executedPermissions []string

	requirePermission :=
		func(
			permission string,
		) gin.HandlerFunc {

			return func(
				context *gin.Context,
			) {
				executedPermissions =
					append(
						executedPermissions,
						permission,
					)

				/*
					Stop before the real handler.

					This test verifies route-level permission
					wiring only. Service behavior is tested
					separately.
				*/
				context.AbortWithStatus(
					http.StatusNoContent,
				)
			}
		}

	group :=
		engine.Group(
			"/api/v1/admin",
		)

	RegisterRoutes(
		group,
		handler,
		requirePermission,
	)

	request :=
		httptest.NewRequest(
			http.MethodDelete,
			"/api/v1/admin/staff/00000000-0000-0000-0000-000000000002",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	engine.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusNoContent {

		t.Fatalf(
			"DELETE staff status = %d, want %d",
			recorder.Code,
			http.StatusNoContent,
		)
	}

	if len(
		executedPermissions,
	) != 1 {

		t.Fatalf(
			"executed permissions = %#v, want exactly one permission",
			executedPermissions,
		)
	}

	if executedPermissions[0] !=
		PermissionStaffDelete {

		t.Fatalf(
			"DELETE staff permission = %q, want %q",
			executedPermissions[0],
			PermissionStaffDelete,
		)
	}
}

func assertAdminErrorCode(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantCode string,
) {
	t.Helper()

	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}

	if err :=
		json.Unmarshal(
			recorder.Body.Bytes(),
			&response,
		); err != nil {

		t.Fatalf(
			"decode error response: %v; body=%s",
			err,
			recorder.Body.String(),
		)
	}

	if response.Error.Code !=
		wantCode {

		t.Fatalf(
			"error code = %q, want %q; body=%s",
			response.Error.Code,
			wantCode,
			recorder.Body.String(),
		)
	}
}

func TestSecurityErrorsRemainComparableWithErrorsIs(
	t *testing.T,
) {
	t.Parallel()

	tests :=
		[]error{
			ErrAdminSelfDelete,
			ErrAdminStaffDeleted,
			ErrAdminStaffBanManagedSeparately,
			ErrAdminProtectedStaffMutation,
			ErrAdminProtectedRoleAssignment,
			ErrAdminRoleAssignmentNotAllowed,
			ErrAdminLastSuperAdmin,
		}

	for _, expected := range tests {

		wrapped :=
			errors.Join(
				errors.New(
					"wrapper",
				),
				expected,
			)

		if !errors.Is(
			wrapped,
			expected,
		) {
			t.Fatalf(
				"errors.Is did not recognize %v",
				expected,
			)
		}
	}
}

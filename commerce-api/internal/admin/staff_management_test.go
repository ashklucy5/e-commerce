package admin

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestNormalizeAdminCodes(t *testing.T) {
	t.Parallel()

	got :=
		normalizeAdminCodes(
			[]string{
				" admin_catalog ",
				"ADMIN_INVENTORY",
				"admin_catalog",
				"",
				"   ",
				"admin_finance",
				"ADMIN_INVENTORY",
			},
		)

	want :=
		[]string{
			"admin_catalog",
			"admin_finance",
			"admin_inventory",
		}

	if !reflect.DeepEqual(
		got,
		want,
	) {
		t.Fatalf(
			"normalizeAdminCodes() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestContainsAdminRoleCode(t *testing.T) {
	t.Parallel()

	roleCodes :=
		[]string{
			RoleCatalog,
			RoleInventory,
			RoleFinance,
		}

	if !containsAdminRoleCode(
		roleCodes,
		RoleInventory,
	) {
		t.Fatal(
			"expected inventory role to be found",
		)
	}

	if containsAdminRoleCode(
		roleCodes,
		RoleSuperAdmin,
	) {
		t.Fatal(
			"did not expect Super Admin role to be found",
		)
	}
}

func TestContainsProtectedAdminRoleCode(t *testing.T) {
	t.Parallel()

	tests :=
		[]struct {
			name      string
			roleCodes []string
			want      bool
		}{
			{
				name: "Super Admin is protected",

				roleCodes: []string{
					RoleCatalog,
					RoleSuperAdmin,
				},

				want: true,
			},
			{
				name: "Administrator is protected",

				roleCodes: []string{
					RoleAdministrator,
				},

				want: true,
			},
			{
				name: "Security is protected",

				roleCodes: []string{
					RoleSecurity,
				},

				want: true,
			},
			{
				name: "ordinary operational roles are not protected",

				roleCodes: []string{
					RoleCatalog,
					RoleInventory,
					RoleWarehouse,
					RoleFulfillment,
				},

				want: false,
			},
			{
				name:      "empty roles are not protected",
				roleCodes: nil,
				want:      false,
			},
		}

	for _, test := range tests {

		test :=
			test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				got :=
					containsProtectedAdminRoleCode(
						test.roleCodes,
					)

				if got !=
					test.want {

					t.Fatalf(
						"containsProtectedAdminRoleCode(%v) = %v, want %v",
						test.roleCodes,
						got,
						test.want,
					)
				}
			},
		)
	}
}

func TestProtectedRoleDefinitions(t *testing.T) {
	t.Parallel()

	tests :=
		[]struct {
			roleCode string
			want     bool
		}{
			{
				roleCode: RoleSuperAdmin,
				want:     true,
			},
			{
				roleCode: RoleAdministrator,
				want:     true,
			},
			{
				roleCode: RoleSecurity,
				want:     true,
			},
			{
				roleCode: RoleSourcing,
				want:     false,
			},
			{
				roleCode: RoleCatalog,
				want:     false,
			},
			{
				roleCode: RoleInventory,
				want:     false,
			},
			{
				roleCode: RoleWarehouse,
				want:     false,
			},
			{
				roleCode: RoleFulfillment,
				want:     false,
			},
			{
				roleCode: RoleFinance,
				want:     false,
			},
			{
				roleCode: RoleReturns,
				want:     false,
			},
			{
				roleCode: RoleAnalytics,
				want:     false,
			},
		}

	for _, test := range tests {

		test :=
			test

		t.Run(
			test.roleCode,
			func(t *testing.T) {
				t.Parallel()

				got :=
					IsProtectedRoleCode(
						test.roleCode,
					)

				if got !=
					test.want {

					t.Fatalf(
						"IsProtectedRoleCode(%q) = %v, want %v",
						test.roleCode,
						got,
						test.want,
					)
				}
			},
		)
	}
}

func TestPermissionCodesContainStaffSecurityPermissions(
	t *testing.T,
) {
	t.Parallel()

	codes :=
		PermissionCodes()

	required :=
		[]string{
			PermissionStaffRead,
			PermissionStaffManage,
			PermissionStaffRoleAssign,
			PermissionStaffSecurityManage,
			PermissionStaffDelete,
			PermissionStaffBan,
		}

	for _, permission := range required {

		if !containsString(
			codes,
			permission,
		) {
			t.Fatalf(
				"PermissionCodes() does not contain required permission %q",
				permission,
			)
		}
	}
}

func TestSetStaffStatusRejectsDedicatedSecurityStates(
	t *testing.T,
) {
	t.Parallel()

	service :=
		&Service{}

	metadata :=
		AdminActionMetadata{
			StaffAccountID: "00000000-0000-0000-0000-000000000001",
		}

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
			{
				name:   "unknown",
				status: "something_else",
			},
		}

	for _, test := range tests {

		test :=
			test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				_,
					err :=
					service.SetStaffStatus(
						context.Background(),
						"00000000-0000-0000-0000-000000000002",
						test.status,
						metadata,
					)

				if !errors.Is(
					err,
					ErrInvalidAdminStaffStatus,
				) {
					t.Fatalf(
						"SetStaffStatus(%q) error = %v, want %v",
						test.status,
						err,
						ErrInvalidAdminStaffStatus,
					)
				}
			},
		)
	}
}

func TestSetStaffStatusRejectsSelfDisable(
	t *testing.T,
) {
	t.Parallel()

	const staffID = "00000000-0000-0000-0000-000000000001"

	service :=
		&Service{}

	tests :=
		[]string{
			"suspended",
			"disabled",
		}

	for _, status := range tests {

		status :=
			status

		t.Run(
			status,
			func(t *testing.T) {
				t.Parallel()

				_,
					err :=
					service.SetStaffStatus(
						context.Background(),
						staffID,
						status,
						AdminActionMetadata{
							StaffAccountID: staffID,
						},
					)

				if !errors.Is(
					err,
					ErrAdminSelfDisable,
				) {
					t.Fatalf(
						"SetStaffStatus(self, %q) error = %v, want %v",
						status,
						err,
						ErrAdminSelfDisable,
					)
				}
			},
		)
	}
}

func TestReplaceStaffRolesRejectsSelfMutation(
	t *testing.T,
) {
	t.Parallel()

	const staffID = "00000000-0000-0000-0000-000000000001"

	service :=
		&Service{}

	_,
		err :=
		service.ReplaceStaffRoles(
			context.Background(),
			staffID,
			[]string{
				RoleCatalog,
			},
			AdminActionMetadata{
				StaffAccountID: staffID,
			},
		)

	if !errors.Is(
		err,
		ErrAdminSelfRoleMutation,
	) {
		t.Fatalf(
			"ReplaceStaffRoles(self) error = %v, want %v",
			err,
			ErrAdminSelfRoleMutation,
		)
	}
}

func TestResetStaffPasswordRejectsSelfReset(
	t *testing.T,
) {
	t.Parallel()

	const staffID = "00000000-0000-0000-0000-000000000001"

	service :=
		&Service{}

	_,
		err :=
		service.ResetStaffPassword(
			context.Background(),
			staffID,
			"this-password-is-never-used",
			AdminActionMetadata{
				StaffAccountID: staffID,
			},
		)

	if !errors.Is(
		err,
		ErrAdminSelfPasswordReset,
	) {
		t.Fatalf(
			"ResetStaffPassword(self) error = %v, want %v",
			err,
			ErrAdminSelfPasswordReset,
		)
	}
}

func TestResetStaffAdminMFARejectsSelfReset(
	t *testing.T,
) {
	t.Parallel()

	const staffID = "00000000-0000-0000-0000-000000000001"

	service :=
		&Service{}

	_,
		err :=
		service.ResetStaffAdminMFA(
			context.Background(),
			staffID,
			AdminActionMetadata{
				StaffAccountID: staffID,
			},
		)

	if !errors.Is(
		err,
		ErrAdminSelfMFAReset,
	) {
		t.Fatalf(
			"ResetStaffAdminMFA(self) error = %v, want %v",
			err,
			ErrAdminSelfMFAReset,
		)
	}
}

func TestDeleteStaffRejectsSelfDelete(
	t *testing.T,
) {
	t.Parallel()

	const staffID = "00000000-0000-0000-0000-000000000001"

	service :=
		&Service{}

	_,
		err :=
		service.DeleteStaff(
			context.Background(),
			staffID,
			AdminActionMetadata{
				StaffAccountID: staffID,
			},
		)

	if !errors.Is(
		err,
		ErrAdminSelfDelete,
	) {
		t.Fatalf(
			"DeleteStaff(self) error = %v, want %v",
			err,
			ErrAdminSelfDelete,
		)
	}
}

func TestStaffLifecycleErrorIdentity(
	t *testing.T,
) {
	t.Parallel()

	errorsToCheck :=
		[]error{
			ErrAdminSelfPasswordReset,
			ErrAdminSelfDelete,
			ErrAdminStaffDeleted,
			ErrAdminStaffBanManagedSeparately,
			ErrAdminProtectedStaffMutation,
			ErrAdminProtectedRoleAssignment,
			ErrAdminRoleAssignmentNotAllowed,
			ErrAdminLastSuperAdmin,
		}

	for index, left := range errorsToCheck {

		for otherIndex, right := range errorsToCheck {

			if index ==
				otherIndex {

				continue
			}

			if errors.Is(
				left,
				right,
			) {
				t.Fatalf(
					"security errors must have distinct identities: %v unexpectedly matches %v",
					left,
					right,
				)
			}
		}
	}
}

func containsString(
	values []string,
	target string,
) bool {
	for _, value := range values {

		if value ==
			target {

			return true
		}
	}

	return false
}

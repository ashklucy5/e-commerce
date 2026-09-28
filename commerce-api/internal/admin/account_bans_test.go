package admin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const testAccountBanUUID = "11111111-1111-4111-8111-111111111111"

func TestNormalizeAndValidateAccountBanInputCustomerTemporaryScope(t *testing.T) {
	t.Parallel()

	now := time.Date(
		2026,
		time.September,
		15,
		1,
		0,
		0,
		0,
		time.UTC,
	)

	expiresAt := now.Add(2 * time.Hour)

	input := CreateAccountBanInput{
		Scope: "  PURCHASING  ",

		BanType: "  TEMPORARY  ",

		Reason: "  repeated checkout abuse  ",

		ExpiresAt: &expiresAt,
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		now,
	)
	if err != nil {
		t.Fatalf(
			"normalizeAndValidateAccountBanInput() error = %v",
			err,
		)
	}

	if input.Scope != AccountBanScopePurchasing {
		t.Fatalf(
			"Scope = %q, want %q",
			input.Scope,
			AccountBanScopePurchasing,
		)
	}

	if input.BanType != AccountBanTypeTemporary {
		t.Fatalf(
			"BanType = %q, want %q",
			input.BanType,
			AccountBanTypeTemporary,
		)
	}

	if input.Reason != "repeated checkout abuse" {
		t.Fatalf(
			"Reason = %q",
			input.Reason,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputPermanentFullAccount(t *testing.T) {
	t.Parallel()

	input := CreateAccountBanInput{
		Scope: AccountBanScopeFullAccount,

		BanType: AccountBanTypePermanent,

		Reason: "security enforcement",
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		time.Now(),
	)
	if err != nil {
		t.Fatalf(
			"normalizeAndValidateAccountBanInput() error = %v",
			err,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputRejectsTemporaryFullAccount(t *testing.T) {
	t.Parallel()

	now := time.Now()

	expiresAt := now.Add(time.Hour)

	input := CreateAccountBanInput{
		Scope: AccountBanScopeFullAccount,

		BanType: AccountBanTypeTemporary,

		Reason: "temporary lockout",

		ExpiresAt: &expiresAt,
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		now,
	)

	if !errors.Is(
		err,
		ErrAdminTemporaryFullAccountBanUnsupported,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminTemporaryFullAccountBanUnsupported",
			err,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputRejectsStaffScopedBan(t *testing.T) {
	t.Parallel()

	input := CreateAccountBanInput{
		Scope: AccountBanScopePurchasing,

		BanType: AccountBanTypePermanent,

		Reason: "invalid staff scope",
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetStaff,
		&input,
		time.Now(),
	)

	if !errors.Is(
		err,
		ErrAdminBanInvalidInput,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminBanInvalidInput",
			err,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputRejectsExpiredTemporaryBan(t *testing.T) {
	t.Parallel()

	now := time.Now()

	expiresAt := now.Add(-time.Minute)

	input := CreateAccountBanInput{
		Scope: AccountBanScopeReviews,

		BanType: AccountBanTypeTemporary,

		Reason: "expired request",

		ExpiresAt: &expiresAt,
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		now,
	)

	if !errors.Is(
		err,
		ErrAdminBanInvalidInput,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminBanInvalidInput",
			err,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputRejectsPermanentExpiry(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().Add(time.Hour)

	input := CreateAccountBanInput{
		Scope: AccountBanScopeReviews,

		BanType: AccountBanTypePermanent,

		Reason: "permanent restriction",

		ExpiresAt: &expiresAt,
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		time.Now(),
	)

	if !errors.Is(
		err,
		ErrAdminBanInvalidInput,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminBanInvalidInput",
			err,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputRejectsBlankReason(t *testing.T) {
	t.Parallel()

	input := CreateAccountBanInput{
		Scope: AccountBanScopeReviews,

		BanType: AccountBanTypePermanent,

		Reason: "   ",
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		time.Now(),
	)

	if !errors.Is(
		err,
		ErrAdminBanInvalidInput,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminBanInvalidInput",
			err,
		)
	}
}

func TestNormalizeAndValidateAccountBanInputRejectsOversizedReason(t *testing.T) {
	t.Parallel()

	input := CreateAccountBanInput{
		Scope: AccountBanScopeReviews,

		BanType: AccountBanTypePermanent,

		Reason: strings.Repeat(
			"x",
			1001,
		),
	}

	err := normalizeAndValidateAccountBanInput(
		AccountBanTargetCustomer,
		&input,
		time.Now(),
	)

	if !errors.Is(
		err,
		ErrAdminBanInvalidInput,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminBanInvalidInput",
			err,
		)
	}
}

func TestValidAccountBanTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		targetType string
		targetID   string

		want bool
	}{
		{
			name: "staff valid",

			targetType: AccountBanTargetStaff,

			targetID: testAccountBanUUID,

			want: true,
		},
		{
			name: "customer valid",

			targetType: AccountBanTargetCustomer,

			targetID: testAccountBanUUID,

			want: true,
		},
		{
			name: "unknown target",

			targetType: "vendor",

			targetID: testAccountBanUUID,

			want: false,
		},
		{
			name: "invalid uuid",

			targetType: AccountBanTargetCustomer,

			targetID: "not-a-uuid",

			want: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				got := validAccountBanTarget(
					testCase.targetType,
					testCase.targetID,
				)

				if got != testCase.want {
					t.Fatalf(
						"validAccountBanTarget() = %v, want %v",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

func TestValidateFullAccountBanStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		targetType string
		status     string

		wantErr error
	}{
		{
			name: "active staff",

			targetType: AccountBanTargetStaff,

			status: "active",
		},
		{
			name: "suspended staff",

			targetType: AccountBanTargetStaff,

			status: "suspended",
		},
		{
			name: "disabled staff",

			targetType: AccountBanTargetStaff,

			status: "disabled",
		},
		{
			name: "deleted staff",

			targetType: AccountBanTargetStaff,

			status: "deleted",

			wantErr: ErrAdminStaffDeleted,
		},
		{
			name: "already banned staff",

			targetType: AccountBanTargetStaff,

			status: "banned",

			wantErr: ErrAdminBanStateConflict,
		},
		{
			name: "active customer",

			targetType: AccountBanTargetCustomer,

			status: "active",
		},
		{
			name: "disabled customer",

			targetType: AccountBanTargetCustomer,

			status: "disabled",
		},
		{
			name: "already banned customer",

			targetType: AccountBanTargetCustomer,

			status: "banned",

			wantErr: ErrAdminBanStateConflict,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				err := validateFullAccountBanStatus(
					testCase.targetType,
					testCase.status,
				)

				if testCase.wantErr == nil {
					if err != nil {
						t.Fatalf(
							"error = %v, want nil",
							err,
						)
					}

					return
				}

				if !errors.Is(
					err,
					testCase.wantErr,
				) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						testCase.wantErr,
					)
				}
			},
		)
	}
}

func TestValidateAccountBanRestoreStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		targetType string
		status     string

		valid bool
	}{
		{
			name: "staff active",

			targetType: AccountBanTargetStaff,

			status: "active",

			valid: true,
		},
		{
			name: "staff suspended",

			targetType: AccountBanTargetStaff,

			status: "suspended",

			valid: true,
		},
		{
			name: "staff disabled",

			targetType: AccountBanTargetStaff,

			status: "disabled",

			valid: true,
		},
		{
			name: "staff banned invalid",

			targetType: AccountBanTargetStaff,

			status: "banned",
		},
		{
			name: "customer active",

			targetType: AccountBanTargetCustomer,

			status: "active",

			valid: true,
		},
		{
			name: "customer disabled",

			targetType: AccountBanTargetCustomer,

			status: "disabled",

			valid: true,
		},
		{
			name: "customer banned invalid",

			targetType: AccountBanTargetCustomer,

			status: "banned",
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				err := validateAccountBanRestoreStatus(
					testCase.targetType,
					testCase.status,
				)

				if testCase.valid {
					if err != nil {
						t.Fatalf(
							"error = %v, want nil",
							err,
						)
					}

					return
				}

				if !errors.Is(
					err,
					ErrAdminBanStateConflict,
				) {
					t.Fatalf(
						"error = %v, want ErrAdminBanStateConflict",
						err,
					)
				}
			},
		)
	}
}

func TestCreateStaffBanRejectsSelfBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	service := &Service{}

	_, err := service.CreateStaffBan(
		context.Background(),
		testAccountBanUUID,
		CreateAccountBanInput{},
		AdminActionMetadata{
			StaffAccountID: testAccountBanUUID,
		},
	)

	if !errors.Is(
		err,
		ErrAdminSelfBan,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminSelfBan",
			err,
		)
	}
}

func TestRevokeStaffBanRejectsSelfBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	service := &Service{}

	_, err := service.RevokeStaffBan(
		context.Background(),
		testAccountBanUUID,
		"22222222-2222-4222-8222-222222222222",
		"restore access",
		AdminActionMetadata{
			StaffAccountID: testAccountBanUUID,
		},
	)

	if !errors.Is(
		err,
		ErrAdminSelfBan,
	) {
		t.Fatalf(
			"error = %v, want ErrAdminSelfBan",
			err,
		)
	}
}

func TestAccountBanContainsRole(t *testing.T) {
	t.Parallel()

	roles := []string{
		RoleCatalog,
		RoleInventory,
		RoleSuperAdmin,
	}

	if !accountBanContainsRole(
		roles,
		RoleSuperAdmin,
	) {
		t.Fatal(
			"accountBanContainsRole() did not find existing role",
		)
	}

	if accountBanContainsRole(
		roles,
		RoleSecurity,
	) {
		t.Fatal(
			"accountBanContainsRole() found role that is not present",
		)
	}
}

func TestAccountBanAuditEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		targetType string
		revoke     bool

		want string
	}{
		{
			targetType: AccountBanTargetStaff,

			want: adminEventStaffBanCreated,
		},
		{
			targetType: AccountBanTargetStaff,

			revoke: true,

			want: adminEventStaffBanRevoked,
		},
		{
			targetType: AccountBanTargetCustomer,

			want: adminEventCustomerBanCreated,
		},
		{
			targetType: AccountBanTargetCustomer,

			revoke: true,

			want: adminEventCustomerBanRevoked,
		},
	}

	for _, testCase := range tests {
		got := accountBanAuditEventType(
			testCase.targetType,
			testCase.revoke,
		)

		if got != testCase.want {
			t.Fatalf(
				"accountBanAuditEventType(%q, %v) = %q, want %q",
				testCase.targetType,
				testCase.revoke,
				got,
				testCase.want,
			)
		}
	}
}

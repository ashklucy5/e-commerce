package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"project.local/commerce-api/internal/adminauth"
	platformdatabase "project.local/commerce-api/internal/platform/database"
)

const adminEventSuperAdminBreakGlassRecovery = "super_admin_break_glass_recovery"

var (
	ErrAdminBreakGlassInvalidInput          = errors.New("invalid Super Admin break-glass recovery input")
	ErrAdminBreakGlassTargetNotFound        = errors.New("Super Admin break-glass target not found")
	ErrAdminBreakGlassTargetNotActive       = errors.New("Super Admin break-glass target must be active")
	ErrAdminBreakGlassTargetNotSuperAdmin   = errors.New("Super Admin break-glass target does not have the Super Admin role")
	ErrAdminBreakGlassOtherActiveSuperAdmin = errors.New("another active Super Admin exists; use the normal privileged recovery flow")
)

type SuperAdminBreakGlassInput struct {
	Identifier  string
	NewPassword string
	ResetMFA    bool
	Operator    string
}

type SuperAdminBreakGlassResult struct {
	StaffAccountID string
	StaffCode      string
	Email          string
	MFAReset       bool
}

// RecoverLastSuperAdmin performs an offline/server-side credential recovery for
// the sole active Super Admin. It is intentionally not exposed through HTTP.
//
// If another active Super Admin exists, recovery must happen through the normal
// privileged staff-security flow instead of this break-glass path.
func RecoverLastSuperAdmin(
	ctx context.Context,
	db *pgxpool.Pool,
	input SuperAdminBreakGlassInput,
) (SuperAdminBreakGlassResult, error) {
	if db == nil {
		return SuperAdminBreakGlassResult{}, ErrAdminBreakGlassInvalidInput
	}

	input.Identifier = strings.TrimSpace(input.Identifier)
	input.Operator = strings.TrimSpace(input.Operator)

	if input.Identifier == "" || input.Operator == "" {
		return SuperAdminBreakGlassResult{}, ErrAdminBreakGlassInvalidInput
	}

	passwordHash, err := adminauth.HashNewPassword(input.NewPassword)
	if err != nil {
		return SuperAdminBreakGlassResult{}, err
	}

	var result SuperAdminBreakGlassResult

	err = platformdatabase.WithinTxOptions(
		ctx,
		db,
		pgx.TxOptions{},
		func(ctx context.Context, tx pgx.Tx) error {
			if _, err := tx.Exec(
				ctx,
				`SELECT pg_advisory_xact_lock(hashtext('commerce-admin-break-glass-recovery-v1'))`,
			); err != nil {
				return fmt.Errorf("lock Super Admin break-glass recovery: %w", err)
			}

			target, err := lockBreakGlassTargetTx(ctx, tx, input.Identifier)
			if err != nil {
				return err
			}

			if target.Status != "active" {
				return ErrAdminBreakGlassTargetNotActive
			}

			isSuperAdmin, err := adminStaffHasRoleTx(
				ctx,
				tx,
				target.ID,
				RoleSuperAdmin,
			)
			if err != nil {
				return err
			}
			if !isSuperAdmin {
				return ErrAdminBreakGlassTargetNotSuperAdmin
			}

			var otherActiveSuperAdmins int
			if err := tx.QueryRow(
				ctx,
				`
					SELECT COUNT(DISTINCT sa.id)::integer
					FROM staff_accounts sa
					JOIN staff_account_roles sar
						ON sar.staff_account_id = sa.id
					JOIN staff_roles sr
						ON sr.id = sar.role_id
					WHERE
						sa.status = 'active'
						AND sr.code = $1
						AND sa.id <> $2::uuid
				`,
				RoleSuperAdmin,
				target.ID,
			).Scan(&otherActiveSuperAdmins); err != nil {
				return fmt.Errorf("count other active Super Admins: %w", err)
			}

			if otherActiveSuperAdmins > 0 {
				return ErrAdminBreakGlassOtherActiveSuperAdmin
			}

			if _, err := tx.Exec(
				ctx,
				`
					UPDATE staff_accounts
					SET
						password_hash = $2,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				target.ID,
				passwordHash,
			); err != nil {
				return fmt.Errorf("reset last Super Admin password: %w", err)
			}

			if input.ResetMFA {
				if _, err := tx.Exec(
					ctx,
					`
						UPDATE admin_totp_credentials
						SET
							status = 'disabled',
							last_accepted_step = NULL,
							disabled_at = now(),
							updated_at = now()
						WHERE staff_account_id = $1::uuid
					`,
					target.ID,
				); err != nil {
					return fmt.Errorf("disable last Super Admin MFA credential: %w", err)
				}

				if _, err := tx.Exec(
					ctx,
					`DELETE FROM admin_mfa_recovery_codes WHERE staff_account_id = $1::uuid`,
					target.ID,
				); err != nil {
					return fmt.Errorf("delete last Super Admin MFA recovery codes: %w", err)
				}
			}

			if err := revokeStaffSecuritySessionsTx(
				ctx,
				tx,
				target.ID,
				"super_admin_break_glass_recovery",
			); err != nil {
				return err
			}

			if err := insertAdminActionAuditTx(
				ctx,
				tx,
				AdminActionMetadata{
					StaffAccountID: target.ID,
					UserAgent:      "admin-recovery-cli",
				},
				adminEventSuperAdminBreakGlassRecovery,
				map[string]any{
					"operator":                input.Operator,
					"source":                  "offline_break_glass_cli",
					"target_staff_account_id": target.ID,
					"target_staff_code":       target.StaffCode,
					"password_reset":          true,
					"mfa_reset":               input.ResetMFA,
				},
			); err != nil {
				return err
			}

			result = SuperAdminBreakGlassResult{
				StaffAccountID: target.ID,
				StaffCode:      target.StaffCode,
				Email:          target.Email,
				MFAReset:       input.ResetMFA,
			}

			return nil
		},
	)
	if err != nil {
		return SuperAdminBreakGlassResult{}, err
	}

	return result, nil
}

type breakGlassTarget struct {
	ID        string
	StaffCode string
	Email     string
	Status    string
}

func lockBreakGlassTargetTx(
	ctx context.Context,
	tx pgx.Tx,
	identifier string,
) (breakGlassTarget, error) {
	var target breakGlassTarget

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				staff_code,
				email,
				status
			FROM staff_accounts
			WHERE
				id::text = $1
				OR upper(staff_code) = upper($1)
				OR lower(email) = lower($1)
			LIMIT 1
			FOR UPDATE
		`,
		identifier,
	).Scan(
		&target.ID,
		&target.StaffCode,
		&target.Email,
		&target.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return breakGlassTarget{}, ErrAdminBreakGlassTargetNotFound
	}
	if err != nil {
		return breakGlassTarget{}, fmt.Errorf("lock Super Admin break-glass target: %w", err)
	}

	return target, nil
}

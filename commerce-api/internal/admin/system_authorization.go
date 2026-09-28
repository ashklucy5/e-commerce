package admin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/support"
)

func SyncSystemAuthorization(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	if db == nil {
		return fmt.Errorf(
			"sync Admin system authorization: database is required",
		)
	}

	return platformdatabase.WithinTx(
		ctx,
		db,
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			if _, err :=
				tx.Exec(
					ctx,
					`
						SELECT pg_advisory_xact_lock(
							hashtext(
								'commerce-admin-system-authorization-v2'
							)
						)
					`,
				); err != nil {

				return fmt.Errorf(
					"lock Admin system authorization sync: %w",
					err,
				)
			}

			if err :=
				syncAdminPermissionsTx(
					ctx,
					tx,
				); err != nil {

				return err
			}

			for _, role := range SystemRoles() {

				if err :=
					syncAdminSystemRoleTx(
						ctx,
						tx,
						role,
					); err != nil {

					return err
				}
			}

			if err :=
				syncSupportAdminBridgeTx(
					ctx,
					tx,
				); err != nil {

				return err
			}

			return nil
		},
	)
}

func syncAdminPermissionsTx(
	ctx context.Context,
	tx pgx.Tx,
) error {
	for _, permission := range Permissions {

		if _, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO staff_permissions (
						code,
						description,
						created_at
					)
					VALUES (
						$1,
						$2,
						now()
					)
					ON CONFLICT (code)
					DO UPDATE SET
						description = EXCLUDED.description
				`,
				permission.Code,
				permission.Description,
			); err != nil {

			return fmt.Errorf(
				"sync Admin permission %s: %w",
				permission.Code,
				err,
			)
		}
	}

	return nil
}

func syncAdminSystemRoleTx(
	ctx context.Context,
	tx pgx.Tx,
	role RoleDefinition,
) error {
	var roleID string

	if err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO staff_roles (
					code,
					name,
					description,
					is_system_role,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					true,
					now(),
					now()
				)
				ON CONFLICT (code)
				DO UPDATE SET
					name = EXCLUDED.name,
					description = EXCLUDED.description,
					is_system_role = true,
					updated_at = now()
				RETURNING id::text
			`,
			role.Code,
			role.Name,
			role.Description,
		).Scan(
			&roleID,
		); err != nil {

		return fmt.Errorf(
			"sync Admin system role %s: %w",
			role.Code,
			err,
		)
	}

	if _, err :=
		tx.Exec(
			ctx,
			`
				DELETE FROM staff_role_permissions
				WHERE role_id = $1::uuid
			`,
			roleID,
		); err != nil {

		return fmt.Errorf(
			"clear Admin system role %s permissions: %w",
			role.Code,
			err,
		)
	}

	for _, permissionCode := range role.Permissions {

		tag, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO staff_role_permissions (
						role_id,
						permission_id,
						created_at
					)
					SELECT
						$1::uuid,
						p.id,
						now()
					FROM staff_permissions p
					WHERE p.code = $2
					ON CONFLICT DO NOTHING
				`,
				roleID,
				permissionCode,
			)
		if err != nil {
			return fmt.Errorf(
				"assign Admin role %s permission %s: %w",
				role.Code,
				permissionCode,
				err,
			)
		}

		if tag.RowsAffected() == 1 {
			continue
		}

		var exists bool

		if err :=
			tx.QueryRow(
				ctx,
				`
					SELECT EXISTS (
						SELECT 1
						FROM staff_role_permissions rp
						JOIN staff_permissions p
							ON p.id = rp.permission_id
						WHERE
							rp.role_id = $1::uuid
							AND p.code = $2
					)
				`,
				roleID,
				permissionCode,
			).Scan(
				&exists,
			); err != nil {

			return fmt.Errorf(
				"verify Admin role %s permission %s: %w",
				role.Code,
				permissionCode,
				err,
			)
		}

		if !exists {
			return fmt.Errorf(
				"Admin permission %s does not exist while synchronizing role %s",
				permissionCode,
				role.Code,
			)
		}
	}

	return nil
}

func syncSupportAdminBridgeTx(
	ctx context.Context,
	tx pgx.Tx,
) error {
	type supportRoleBridge struct {
		RoleCode string

		Permissions []string
	}

	bridges := []supportRoleBridge{
		{
			RoleCode: support.RoleAgent,

			Permissions: []string{
				PermissionPanelAccess,

				PermissionDashboardRead,

				PermissionCustomerRead,

				PermissionOrderRead,

				PermissionCRMRead,
				PermissionCRMManage,

				PermissionSourcingRead,
				PermissionSourcingReview,
				PermissionSourcingOfferManage,

				PermissionNotificationRead,
			},
		},
		{
			RoleCode: support.RoleSupervisor,

			Permissions: []string{
				PermissionPanelAccess,

				PermissionDashboardRead,

				PermissionCustomerRead,

				PermissionOrderRead,

				PermissionCRMRead,
				PermissionCRMManage,

				PermissionSourcingRead,
				PermissionSourcingReview,
				PermissionSourcingOfferManage,
				PermissionSourcingFinalize,

				PermissionNotificationRead,
			},
		},
	}

	for _, bridge := range bridges {

		var roleID string

		err :=
			tx.QueryRow(
				ctx,
				`
					SELECT id::text
					FROM staff_roles
					WHERE code = $1
				`,
				bridge.RoleCode,
			).Scan(
				&roleID,
			)

		if err == pgx.ErrNoRows {
			/*
				The support bootstrap may not have been run
				in this environment yet.

				Do not create an incomplete support role here.
				The bridge will be applied automatically on the
				next API startup after that role exists.
			*/
			continue
		}

		if err != nil {
			return fmt.Errorf(
				"load support role %s for Admin bridge: %w",
				bridge.RoleCode,
				err,
			)
		}

		for _, permissionCode := range bridge.Permissions {

			tag, err :=
				tx.Exec(
					ctx,
					`
						INSERT INTO staff_role_permissions (
							role_id,
							permission_id,
							created_at
						)
						SELECT
							$1::uuid,
							p.id,
							now()
						FROM staff_permissions p
						WHERE p.code = $2
						ON CONFLICT DO NOTHING
					`,
					roleID,
					permissionCode,
				)

			if err != nil {
				return fmt.Errorf(
					"bridge support role %s to Admin permission %s: %w",
					bridge.RoleCode,
					permissionCode,
					err,
				)
			}

			if tag.RowsAffected() == 1 {
				continue
			}

			var exists bool

			if err :=
				tx.QueryRow(
					ctx,
					`
						SELECT EXISTS (
							SELECT 1
							FROM staff_role_permissions rp
							JOIN staff_permissions p
								ON p.id = rp.permission_id
							WHERE
								rp.role_id = $1::uuid
								AND p.code = $2
						)
					`,
					roleID,
					permissionCode,
				).Scan(
					&exists,
				); err != nil {

				return fmt.Errorf(
					"verify support role %s Admin permission %s: %w",
					bridge.RoleCode,
					permissionCode,
					err,
				)
			}

			if !exists {
				return fmt.Errorf(
					"Admin permission %s does not exist while bridging support role %s",
					permissionCode,
					bridge.RoleCode,
				)
			}
		}
	}

	return nil
}

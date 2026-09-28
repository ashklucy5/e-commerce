package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
)

const (
	adminEventRoleCreated            = "staff_role_created"
	adminEventRoleUpdated            = "staff_role_updated"
	adminEventRolePermissionsChanged = "staff_role_permissions_changed"
)

var (
	ErrAdminRoleNotFound = errors.New("Admin staff role not found")

	ErrAdminRoleConflict = errors.New("Admin staff role already exists")

	ErrAdminSystemRoleImmutable = errors.New("system staff roles cannot be modified")

	ErrAdminPermissionCodesInvalid = errors.New("one or more staff permissions do not exist")
)

type AdminPermission struct {
	ID string `json:"id"`

	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

type AdminRoleListItem struct {
	ID string `json:"id"`

	Code string `json:"code"`
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	IsSystemRole bool `json:"is_system_role"`

	PermissionCount int64 `json:"permission_count"`
	AssignedStaff   int64 `json:"assigned_staff"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminRoleDetail struct {
	AdminRoleListItem

	Permissions []AdminPermission `json:"permissions"`
}

type CreateRoleInput struct {
	Code string
	Name string

	Description string

	PermissionCodes []string
}

type RoleMetadataUpdate struct {
	Name *string

	Description *string
}

func (s *Service) ListRoles(
	ctx context.Context,
) (
	[]AdminRoleListItem,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				SELECT
					sr.id::text,
					sr.code,
					sr.name,
					COALESCE(sr.description, ''),
					sr.is_system_role,
					(
						SELECT COUNT(*)::bigint
						FROM staff_role_permissions srp
						WHERE srp.role_id = sr.id
					),
					(
						SELECT COUNT(*)::bigint
						FROM staff_account_roles sar
						WHERE sar.role_id = sr.id
					),
					sr.created_at,
					sr.updated_at
				FROM staff_roles sr
				ORDER BY
					sr.is_system_role DESC,
					sr.code
			`,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin roles: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]AdminRoleListItem,
			0,
		)

	for rows.Next() {
		var item AdminRoleListItem

		if err :=
			rows.Scan(
				&item.ID,
				&item.Code,
				&item.Name,
				&item.Description,
				&item.IsSystemRole,
				&item.PermissionCount,
				&item.AssignedStaff,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan Admin role: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate Admin roles: %w",
				err,
			)
	}

	return result, nil
}

func (s *Service) ListPermissions(
	ctx context.Context,
) (
	[]AdminPermission,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				SELECT
					id::text,
					code,
					COALESCE(description, '')
				FROM staff_permissions
				ORDER BY code
			`,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin permissions: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]AdminPermission,
			0,
		)

	for rows.Next() {
		var item AdminPermission

		if err :=
			rows.Scan(
				&item.ID,
				&item.Code,
				&item.Description,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan Admin permission: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate Admin permissions: %w",
				err,
			)
	}

	return result, nil
}

func (s *Service) GetRole(
	ctx context.Context,
	roleID string,
) (
	AdminRoleDetail,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	return getAdminRoleDetail(
		queryCtx,
		s.db,
		roleID,
	)
}

func (s *Service) CreateRole(
	ctx context.Context,
	input CreateRoleInput,
	metadata AdminActionMetadata,
) (
	AdminRoleDetail,
	error,
) {
	permissionCodes :=
		normalizeAdminCodes(
			input.PermissionCodes,
		)

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminRoleDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				if err :=
					validatePermissionCodesTx(
						ctx,
						tx,
						permissionCodes,
					); err != nil {

					return err
				}

				var roleID string

				err :=
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
								NULLIF($3, ''),
								false,
								now(),
								now()
							)
							ON CONFLICT (code)
							DO NOTHING
							RETURNING id::text
						`,
						strings.ToLower(
							strings.TrimSpace(
								input.Code,
							),
						),
						strings.TrimSpace(
							input.Name,
						),
						strings.TrimSpace(
							input.Description,
						),
					).Scan(
						&roleID,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminRoleConflict
				}

				if err != nil {
					return fmt.Errorf(
						"create Admin role: %w",
						err,
					)
				}

				if err :=
					replaceRolePermissionsTx(
						ctx,
						tx,
						roleID,
						permissionCodes,
					); err != nil {

					return err
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventRoleCreated,
						map[string]any{
							"role_id": roleID,

							"role_code": strings.ToLower(
								strings.TrimSpace(
									input.Code,
								),
							),

							"permission_codes": permissionCodes,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminRoleDetail(
						ctx,
						tx,
						roleID,
					)
				if err != nil {
					return err
				}

				result =
					item

				return nil
			},
		)
	if err != nil {
		return AdminRoleDetail{},
			err
	}

	return result, nil
}

func (s *Service) UpdateRole(
	ctx context.Context,
	roleID string,
	update RoleMetadataUpdate,
	metadata AdminActionMetadata,
) (
	AdminRoleDetail,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminRoleDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var currentName string
				var currentDescription string
				var isSystemRole bool

				err :=
					tx.QueryRow(
						ctx,
						`
							SELECT
								name,
								COALESCE(description, ''),
								is_system_role
							FROM staff_roles
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						roleID,
					).Scan(
						&currentName,
						&currentDescription,
						&isSystemRole,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminRoleNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin role: %w",
						err,
					)
				}

				if isSystemRole {
					return ErrAdminSystemRoleImmutable
				}

				newName :=
					currentName

				newDescription :=
					currentDescription

				if update.Name != nil {
					newName =
						strings.TrimSpace(
							*update.Name,
						)
				}

				if update.Description != nil {
					newDescription =
						strings.TrimSpace(
							*update.Description,
						)
				}

				_, err =
					tx.Exec(
						ctx,
						`
							UPDATE staff_roles
							SET
								name = $2,
								description = NULLIF($3, ''),
								updated_at = now()
							WHERE id = $1::uuid
						`,
						roleID,
						newName,
						newDescription,
					)
				if err != nil {
					return fmt.Errorf(
						"update Admin role: %w",
						err,
					)
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventRoleUpdated,
						map[string]any{
							"role_id": roleID,

							"previous_name": currentName,

							"new_name": newName,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminRoleDetail(
						ctx,
						tx,
						roleID,
					)
				if err != nil {
					return err
				}

				result =
					item

				return nil
			},
		)
	if err != nil {
		return AdminRoleDetail{},
			err
	}

	return result, nil
}

func (s *Service) ReplaceRolePermissions(
	ctx context.Context,
	roleID string,
	permissionCodes []string,
	metadata AdminActionMetadata,
) (
	AdminRoleDetail,
	error,
) {
	permissionCodes =
		normalizeAdminCodes(
			permissionCodes,
		)

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminRoleDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var isSystemRole bool

				err :=
					tx.QueryRow(
						ctx,
						`
							SELECT is_system_role
							FROM staff_roles
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						roleID,
					).Scan(
						&isSystemRole,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminRoleNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin role: %w",
						err,
					)
				}

				if isSystemRole {
					return ErrAdminSystemRoleImmutable
				}

				if err :=
					validatePermissionCodesTx(
						ctx,
						tx,
						permissionCodes,
					); err != nil {

					return err
				}

				previousPermissions, err :=
					adminRolePermissionCodesTx(
						ctx,
						tx,
						roleID,
					)
				if err != nil {
					return err
				}

				if err :=
					replaceRolePermissionsTx(
						ctx,
						tx,
						roleID,
						permissionCodes,
					); err != nil {

					return err
				}

				// Authorization is read from the DB on every Admin
				// request, so the new permissions take effect
				// immediately without a cache invalidation step.

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventRolePermissionsChanged,
						map[string]any{
							"role_id": roleID,

							"previous_permission_codes": previousPermissions,

							"new_permission_codes": permissionCodes,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminRoleDetail(
						ctx,
						tx,
						roleID,
					)
				if err != nil {
					return err
				}

				result =
					item

				return nil
			},
		)
	if err != nil {
		return AdminRoleDetail{},
			err
	}

	return result, nil
}

func getAdminRoleDetail(
	ctx context.Context,
	querier adminReadQuerier,
	roleID string,
) (
	AdminRoleDetail,
	error,
) {
	var result AdminRoleDetail

	err :=
		querier.QueryRow(
			ctx,
			`
				SELECT
					sr.id::text,
					sr.code,
					sr.name,
					COALESCE(sr.description, ''),
					sr.is_system_role,
					(
						SELECT COUNT(*)::bigint
						FROM staff_role_permissions srp
						WHERE srp.role_id = sr.id
					),
					(
						SELECT COUNT(*)::bigint
						FROM staff_account_roles sar
						WHERE sar.role_id = sr.id
					),
					sr.created_at,
					sr.updated_at
				FROM staff_roles sr
				WHERE sr.id = $1::uuid
			`,
			roleID,
		).Scan(
			&result.ID,
			&result.Code,
			&result.Name,
			&result.Description,
			&result.IsSystemRole,
			&result.PermissionCount,
			&result.AssignedStaff,
			&result.CreatedAt,
			&result.UpdatedAt,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AdminRoleDetail{},
			ErrAdminRoleNotFound
	}

	if err != nil {
		return AdminRoleDetail{},
			fmt.Errorf(
				"get Admin role: %w",
				err,
			)
	}

	rows, err :=
		querier.Query(
			ctx,
			`
				SELECT
					sp.id::text,
					sp.code,
					COALESCE(sp.description, '')
				FROM staff_role_permissions srp
				JOIN staff_permissions sp
					ON sp.id = srp.permission_id
				WHERE srp.role_id = $1::uuid
				ORDER BY sp.code
			`,
			roleID,
		)
	if err != nil {
		return AdminRoleDetail{},
			fmt.Errorf(
				"load Admin role permissions: %w",
				err,
			)
	}
	defer rows.Close()

	result.Permissions =
		make(
			[]AdminPermission,
			0,
		)

	for rows.Next() {
		var permission AdminPermission

		if err :=
			rows.Scan(
				&permission.ID,
				&permission.Code,
				&permission.Description,
			); err != nil {

			return AdminRoleDetail{},
				fmt.Errorf(
					"scan Admin role permission: %w",
					err,
				)
		}

		result.Permissions =
			append(
				result.Permissions,
				permission,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return AdminRoleDetail{},
			fmt.Errorf(
				"iterate Admin role permissions: %w",
				err,
			)
	}

	return result, nil
}

func validatePermissionCodesTx(
	ctx context.Context,
	tx pgx.Tx,
	permissionCodes []string,
) error {
	if len(permissionCodes) == 0 {
		return nil
	}

	var count int

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::integer
				FROM staff_permissions
				WHERE code = ANY($1::text[])
			`,
			permissionCodes,
		).Scan(
			&count,
		)
	if err != nil {
		return fmt.Errorf(
			"validate staff permissions: %w",
			err,
		)
	}

	if count !=
		len(permissionCodes) {

		return ErrAdminPermissionCodesInvalid
	}

	return nil
}

func replaceRolePermissionsTx(
	ctx context.Context,
	tx pgx.Tx,
	roleID string,
	permissionCodes []string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				DELETE FROM staff_role_permissions
				WHERE role_id = $1::uuid
			`,
			roleID,
		)
	if err != nil {
		return fmt.Errorf(
			"clear role permissions: %w",
			err,
		)
	}

	if len(permissionCodes) == 0 {
		return nil
	}

	_, err =
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
					sp.id,
					now()
				FROM staff_permissions sp
				WHERE sp.code = ANY($2::text[])
			`,
			roleID,
			permissionCodes,
		)
	if err != nil {
		return fmt.Errorf(
			"assign role permissions: %w",
			err,
		)
	}

	return nil
}

func adminRolePermissionCodesTx(
	ctx context.Context,
	tx pgx.Tx,
	roleID string,
) (
	[]string,
	error,
) {
	rows, err :=
		tx.Query(
			ctx,
			`
				SELECT sp.code
				FROM staff_role_permissions srp
				JOIN staff_permissions sp
					ON sp.id = srp.permission_id
				WHERE srp.role_id = $1::uuid
				ORDER BY sp.code
			`,
			roleID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load role permission codes: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]string,
			0,
		)

	for rows.Next() {
		var code string

		if err :=
			rows.Scan(
				&code,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan role permission code: %w",
					err,
				)
		}

		result =
			append(
				result,
				code,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate role permission codes: %w",
				err,
			)
	}

	return result, nil
}

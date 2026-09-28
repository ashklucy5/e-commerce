package admin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"project.local/commerce-api/internal/adminauth"
	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

const (
	adminEventStaffCreated       = "staff_created"
	adminEventStaffStatusChanged = "staff_status_changed"
	adminEventStaffRolesChanged  = "staff_roles_changed"
	adminEventStaffPasswordReset = "staff_password_reset"
	adminEventStaffMFAReset      = "staff_admin_mfa_reset"
	adminEventStaffDeleted       = "staff_deleted"
)

var (
	ErrAdminStaffNotFound = errors.New("Admin staff account not found")

	ErrAdminStaffConflict = errors.New("Admin staff account already exists")

	ErrInvalidAdminStaffStatus = errors.New("invalid Admin staff status")

	ErrAdminStaffRolesInvalid = errors.New("one or more staff roles do not exist")

	ErrAdminSelfRoleMutation = errors.New("cannot change your own role assignments")

	ErrAdminSelfDisable = errors.New("cannot suspend or disable your own account")

	ErrAdminSelfPasswordReset = errors.New("cannot reset your own password through staff administration")

	ErrAdminSelfMFAReset = errors.New("cannot reset your own Admin MFA")

	ErrAdminSelfDelete = errors.New("cannot delete your own staff account")

	ErrAdminStaffDeleted = errors.New("deleted staff accounts cannot be reactivated or modified")

	ErrAdminStaffBanManagedSeparately = errors.New("banned staff accounts must be managed through the ban system")

	ErrAdminProtectedStaffMutation = errors.New("only a Super Admin can manage staff with protected roles")

	ErrAdminProtectedRoleAssignment = errors.New("only a Super Admin can assign protected staff roles")

	ErrAdminRoleAssignmentNotAllowed = errors.New("this role cannot be assigned by a non-Super-Admin")

	ErrAdminLastSuperAdmin = errors.New("cannot remove access from the last active Super Admin")
)

type StaffReadFilter struct {
	Status string

	RoleCode string

	Query   string
	QueryID string
}

type CreateStaffInput struct {
	FullName string
	Email    string
	Phone    string

	DeliveryMode string

	RoleCodes []string
}

type AdminStaffListItem struct {
	ID        string `json:"id"`
	StaffCode string `json:"staff_code"`

	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone,omitempty"`

	Status string `json:"status"`

	Roles []string `json:"roles"`

	HasAdminPanelAccess bool   `json:"has_admin_panel_access"`
	AdminMFAStatus      string `json:"admin_mfa_status"`

	ActiveStaffSessions int64 `json:"active_staff_sessions"`
	ActiveAdminSessions int64 `json:"active_admin_sessions"`

	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminStaffSupportActor struct {
	ID        string `json:"id"`
	ActorCode string `json:"actor_code"`

	Status   string `json:"status"`
	Presence string `json:"presence"`
}

type AdminStaffDetail struct {
	AdminStaffListItem

	Permissions []string `json:"permissions"`

	SupportActor *AdminStaffSupportActor `json:"support_actor,omitempty"`
}

type StaffListResult struct {
	Items []AdminStaffListItem
	Meta  platformpagination.Meta
}

func (s *Service) ListStaff(
	ctx context.Context,
	params platformpagination.Params,
	filter StaffReadFilter,
) (
	StaffListResult,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				WITH selected AS (
					SELECT
						sa.id,
						sa.staff_code,
						sa.full_name,
						sa.email,
						sa.phone,
						sa.status,
						sa.last_login_at,
						sa.created_at,
						sa.updated_at,
						COUNT(*) OVER()::bigint AS total_count
					FROM staff_accounts sa
					WHERE
						($1 = '' OR sa.status = $1)
						AND (
							$2 = ''
							OR EXISTS (
								SELECT 1
								FROM staff_account_roles sar
								JOIN staff_roles sr
									ON sr.id = sar.role_id
								WHERE
									sar.staff_account_id = sa.id
									AND sr.code = $2
							)
						)
						AND (
							$3 = ''
							OR sa.staff_code = upper($3)
							OR sa.email = lower($3)
							OR COALESCE(sa.phone, '') = $3
							OR sa.id = NULLIF($4, '')::uuid
						)
					ORDER BY
						sa.created_at DESC,
						sa.id DESC
					LIMIT $5
					OFFSET $6
				)
				SELECT
					s.id::text,
					s.staff_code,
					s.full_name,
					s.email,
					COALESCE(s.phone, ''),
					s.status,
					ARRAY(
						SELECT sr.code
						FROM staff_account_roles sar
						JOIN staff_roles sr
							ON sr.id = sar.role_id
						WHERE sar.staff_account_id = s.id
						ORDER BY sr.code
					),
					EXISTS (
						SELECT 1
						FROM staff_account_roles sar
						JOIN staff_role_permissions srp
							ON srp.role_id = sar.role_id
						JOIN staff_permissions sp
							ON sp.id = srp.permission_id
						WHERE
							sar.staff_account_id = s.id
							AND sp.code = 'admin.panel.access'
					),
					COALESCE(
						(
							SELECT atc.status
							FROM admin_totp_credentials atc
							WHERE atc.staff_account_id = s.id
						),
						'not_enrolled'
					),
					(
						SELECT COUNT(*)::bigint
						FROM staff_sessions ss
						WHERE
							ss.staff_account_id = s.id
							AND ss.revoked_at IS NULL
							AND ss.refresh_expires_at > now()
					),
					(
						SELECT COUNT(*)::bigint
						FROM admin_sessions ads
						WHERE
							ads.staff_account_id = s.id
							AND ads.revoked_at IS NULL
							AND ads.refresh_expires_at > now()
					),
					s.last_login_at,
					s.created_at,
					s.updated_at,
					s.total_count
				FROM selected s
				ORDER BY
					s.created_at DESC,
					s.id DESC
			`,
			filter.Status,
			filter.RoleCode,
			filter.Query,
			filter.QueryID,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return StaffListResult{},
			fmt.Errorf(
				"list Admin staff: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]AdminStaffListItem,
			0,
			params.Limit,
		)

	var total int64

	for rows.Next() {
		var item AdminStaffListItem
		var lastLoginAt pgtype.Timestamptz

		if err :=
			rows.Scan(
				&item.ID,
				&item.StaffCode,
				&item.FullName,
				&item.Email,
				&item.Phone,
				&item.Status,
				&item.Roles,
				&item.HasAdminPanelAccess,
				&item.AdminMFAStatus,
				&item.ActiveStaffSessions,
				&item.ActiveAdminSessions,
				&lastLoginAt,
				&item.CreatedAt,
				&item.UpdatedAt,
				&total,
			); err != nil {

			return StaffListResult{},
				fmt.Errorf(
					"scan Admin staff: %w",
					err,
				)
		}

		item.LastLoginAt =
			adminTimePointer(
				lastLoginAt,
			)

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return StaffListResult{},
			fmt.Errorf(
				"iterate Admin staff: %w",
				err,
			)
	}

	return StaffListResult{
		Items: items,

		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetStaff(
	ctx context.Context,
	staffID string,
) (
	AdminStaffDetail,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	return getAdminStaffDetail(
		queryCtx,
		s.db,
		staffID,
	)
}

func (s *Service) CreateStaff(
	ctx context.Context,
	input CreateStaffInput,
	metadata AdminActionMetadata,
) (
	CreateStaffResult,
	error,
) {
	return s.createPendingStaff(
		ctx,
		input,
		metadata,
	)
}
func (s *Service) SetStaffStatus(
	ctx context.Context,
	staffID string,
	status string,
	metadata AdminActionMetadata,
) (
	AdminStaffDetail,
	error,
) {
	status =
		strings.ToLower(
			strings.TrimSpace(
				status,
			),
		)

	switch status {
	case "active",
		"suspended",
		"disabled":

	default:
		return AdminStaffDetail{},
			ErrInvalidAdminStaffStatus
	}

	if staffID ==
		metadata.StaffAccountID &&
		status != "active" {

		return AdminStaffDetail{},
			ErrAdminSelfDisable
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminStaffDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var previousStatus string

				err :=
					tx.QueryRow(
						ctx,
						`
							SELECT status
							FROM staff_accounts
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						staffID,
					).Scan(
						&previousStatus,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminStaffNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin staff account: %w",
						err,
					)
				}

				if previousStatus == "deleted" {
					return ErrAdminStaffDeleted
				}

				if previousStatus == "banned" {
					return ErrAdminStaffBanManagedSeparately
				}

				if previousStatus == "pending_activation" {
					return ErrAdminStaffActivationManagedSeparately
				}

				targetRoles, err :=
					adminStaffRoleCodesTx(
						ctx,
						tx,
						staffID,
					)
				if err != nil {
					return err
				}

				if err :=
					validateProtectedStaffMutationTx(
						ctx,
						tx,
						metadata.StaffAccountID,
						targetRoles,
					); err != nil {

					return err
				}

				if status != "active" &&
					containsAdminRoleCode(
						targetRoles,
						RoleSuperAdmin,
					) {

					if err :=
						ensureAnotherActiveSuperAdminTx(
							ctx,
							tx,
							staffID,
						); err != nil {

						return err
					}
				}

				if previousStatus !=
					status {

					_, err =
						tx.Exec(
							ctx,
							`
								UPDATE staff_accounts
								SET
									status = $2,
									updated_at = now()
								WHERE id = $1::uuid
							`,
							staffID,
							status,
						)
					if err != nil {
						return fmt.Errorf(
							"update Admin staff status: %w",
							err,
						)
					}
				}

				if status != "active" {
					if err :=
						revokeStaffSecuritySessionsTx(
							ctx,
							tx,
							staffID,
							"staff_status_changed",
						); err != nil {

						return err
					}
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventStaffStatusChanged,
						map[string]any{
							"staff_account_id": staffID,

							"previous_status": previousStatus,

							"new_status": status,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminStaffDetail(
						ctx,
						tx,
						staffID,
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
		return AdminStaffDetail{},
			err
	}

	return result, nil
}

func (s *Service) ReplaceStaffRoles(
	ctx context.Context,
	staffID string,
	roleCodes []string,
	metadata AdminActionMetadata,
) (
	AdminStaffDetail,
	error,
) {
	if staffID ==
		metadata.StaffAccountID {

		return AdminStaffDetail{},
			ErrAdminSelfRoleMutation
	}

	roleCodes =
		normalizeAdminCodes(
			roleCodes,
		)

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminStaffDetail

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
					lockAdminStaffTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					ensureStaffNotDeletedTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					validateRoleCodesTx(
						ctx,
						tx,
						roleCodes,
					); err != nil {

					return err
				}

				previousRoles, err :=
					adminStaffRoleCodesTx(
						ctx,
						tx,
						staffID,
					)
				if err != nil {
					return err
				}

				if err :=
					validateStaffRoleReplacementTx(
						ctx,
						tx,
						metadata.StaffAccountID,
						staffID,
						previousRoles,
						roleCodes,
					); err != nil {

					return err
				}

				if err :=
					replaceStaffRolesTx(
						ctx,
						tx,
						staffID,
						roleCodes,
						metadata.StaffAccountID,
					); err != nil {

					return err
				}

				if err :=
					revokeStaffSecuritySessionsTx(
						ctx,
						tx,
						staffID,
						"staff_roles_changed",
					); err != nil {

					return err
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventStaffRolesChanged,
						map[string]any{
							"staff_account_id": staffID,

							"previous_role_codes": previousRoles,

							"new_role_codes": roleCodes,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminStaffDetail(
						ctx,
						tx,
						staffID,
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
		return AdminStaffDetail{},
			err
	}

	return result, nil
}

func (s *Service) ResetStaffPassword(
	ctx context.Context,
	staffID string,
	password string,
	metadata AdminActionMetadata,
) (
	AdminStaffDetail,
	error,
) {
	if staffID ==
		metadata.StaffAccountID {

		return AdminStaffDetail{},
			ErrAdminSelfPasswordReset
	}

	passwordHash, err :=
		adminauth.HashNewPassword(
			password,
		)
	if err != nil {
		return AdminStaffDetail{},
			err
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminStaffDetail

	err =
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				if err :=
					lockAdminStaffTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					ensureStaffNotDeletedTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					ensureStaffOnboardingSecurityMutationAllowedTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					validateStaffSecurityMutationTx(
						ctx,
						tx,
						metadata.StaffAccountID,
						staffID,
					); err != nil {

					return err
				}

				_, err :=
					tx.Exec(
						ctx,
						`
							UPDATE staff_accounts
							SET
								password_hash = $2,
								updated_at = now()
							WHERE id = $1::uuid
						`,
						staffID,
						passwordHash,
					)
				if err != nil {
					return fmt.Errorf(
						"reset staff password: %w",
						err,
					)
				}

				if err :=
					revokeStaffSecuritySessionsTx(
						ctx,
						tx,
						staffID,
						"staff_password_reset",
					); err != nil {

					return err
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventStaffPasswordReset,
						map[string]any{
							"staff_account_id": staffID,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminStaffDetail(
						ctx,
						tx,
						staffID,
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
		return AdminStaffDetail{},
			err
	}

	return result, nil
}

func (s *Service) ResetStaffAdminMFA(
	ctx context.Context,
	staffID string,
	metadata AdminActionMetadata,
) (
	AdminStaffDetail,
	error,
) {
	if staffID ==
		metadata.StaffAccountID {

		return AdminStaffDetail{},
			ErrAdminSelfMFAReset
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminStaffDetail

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
					lockAdminStaffTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					ensureStaffNotDeletedTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					ensureStaffOnboardingSecurityMutationAllowedTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					validateStaffSecurityMutationTx(
						ctx,
						tx,
						metadata.StaffAccountID,
						staffID,
					); err != nil {

					return err
				}

				_, err :=
					tx.Exec(
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
						staffID,
					)
				if err != nil {
					return fmt.Errorf(
						"disable Admin MFA credential: %w",
						err,
					)
				}

				_, err =
					tx.Exec(
						ctx,
						`
							DELETE FROM admin_mfa_recovery_codes
							WHERE staff_account_id = $1::uuid
						`,
						staffID,
					)
				if err != nil {
					return fmt.Errorf(
						"delete Admin MFA recovery codes: %w",
						err,
					)
				}

				if err :=
					revokeStaffSecuritySessionsTx(
						ctx,
						tx,
						staffID,
						"admin_mfa_reset",
					); err != nil {

					return err
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventStaffMFAReset,
						map[string]any{
							"staff_account_id": staffID,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminStaffDetail(
						ctx,
						tx,
						staffID,
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
		return AdminStaffDetail{},
			err
	}

	return result, nil
}

func (s *Service) DeleteStaff(
	ctx context.Context,
	staffID string,
	metadata AdminActionMetadata,
) (
	AdminStaffDetail,
	error,
) {
	if staffID ==
		metadata.StaffAccountID {

		return AdminStaffDetail{},
			ErrAdminSelfDelete
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminStaffDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var previousStatus string

				err :=
					tx.QueryRow(
						ctx,
						`
							SELECT status
							FROM staff_accounts
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						staffID,
					).Scan(
						&previousStatus,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminStaffNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin staff account for deletion: %w",
						err,
					)
				}

				targetRoles, err :=
					adminStaffRoleCodesTx(
						ctx,
						tx,
						staffID,
					)
				if err != nil {
					return err
				}

				if err :=
					validateProtectedStaffMutationTx(
						ctx,
						tx,
						metadata.StaffAccountID,
						targetRoles,
					); err != nil {

					return err
				}

				if previousStatus == "deleted" {
					item, err :=
						getAdminStaffDetail(
							ctx,
							tx,
							staffID,
						)
					if err != nil {
						return err
					}

					result =
						item

					return nil
				}

				if previousStatus == "banned" {
					return ErrAdminStaffBanManagedSeparately
				}

				if containsAdminRoleCode(
					targetRoles,
					RoleSuperAdmin,
				) {
					if err :=
						ensureAnotherActiveSuperAdminTx(
							ctx,
							tx,
							staffID,
						); err != nil {

						return err
					}
				}

				if err :=
					cancelPendingStaffInvitationsTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				_, err =
					tx.Exec(
						ctx,
						`
							UPDATE staff_accounts
							SET
								status = 'deleted',
								updated_at = now()
							WHERE id = $1::uuid
						`,
						staffID,
					)
				if err != nil {
					return fmt.Errorf(
						"delete staff access: %w",
						err,
					)
				}

				if err :=
					revokeStaffSecuritySessionsTx(
						ctx,
						tx,
						staffID,
						"staff_deleted",
					); err != nil {

					return err
				}

				if err :=
					invalidateDeletedStaffMFAForStaffTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					disableSupportActorForStaffTx(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventStaffDeleted,
						map[string]any{
							"staff_account_id": staffID,

							"previous_status": previousStatus,

							"new_status": "deleted",

							"role_codes": targetRoles,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminStaffDetail(
						ctx,
						tx,
						staffID,
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
		return AdminStaffDetail{},
			err
	}

	return result, nil
}

func getAdminStaffDetail(
	ctx context.Context,
	querier adminReadQuerier,
	staffID string,
) (
	AdminStaffDetail,
	error,
) {
	var result AdminStaffDetail

	var lastLoginAt pgtype.Timestamptz

	var supportActorID string
	var supportActorCode string
	var supportActorStatus string
	var supportActorPresence string

	err :=
		querier.QueryRow(
			ctx,
			`
				SELECT
					sa.id::text,
					sa.staff_code,
					sa.full_name,
					sa.email,
					COALESCE(sa.phone, ''),
					sa.status,
					ARRAY(
						SELECT sr.code
						FROM staff_account_roles sar
						JOIN staff_roles sr
							ON sr.id = sar.role_id
						WHERE sar.staff_account_id = sa.id
						ORDER BY sr.code
					),
					ARRAY(
						SELECT DISTINCT sp.code
						FROM staff_account_roles sar
						JOIN staff_role_permissions srp
							ON srp.role_id = sar.role_id
						JOIN staff_permissions sp
							ON sp.id = srp.permission_id
						WHERE sar.staff_account_id = sa.id
						ORDER BY sp.code
					),
					EXISTS (
						SELECT 1
						FROM staff_account_roles sar
						JOIN staff_role_permissions srp
							ON srp.role_id = sar.role_id
						JOIN staff_permissions sp
							ON sp.id = srp.permission_id
						WHERE
							sar.staff_account_id = sa.id
							AND sp.code = 'admin.panel.access'
					),
					COALESCE(
						(
							SELECT atc.status
							FROM admin_totp_credentials atc
							WHERE atc.staff_account_id = sa.id
						),
						'not_enrolled'
					),
					(
						SELECT COUNT(*)::bigint
						FROM staff_sessions ss
						WHERE
							ss.staff_account_id = sa.id
							AND ss.revoked_at IS NULL
							AND ss.refresh_expires_at > now()
					),
					(
						SELECT COUNT(*)::bigint
						FROM admin_sessions ads
						WHERE
							ads.staff_account_id = sa.id
							AND ads.revoked_at IS NULL
							AND ads.refresh_expires_at > now()
					),
					sa.last_login_at,
					sa.created_at,
					sa.updated_at,
					COALESCE(sup.id::text, ''),
					COALESCE(sup.actor_code, ''),
					COALESCE(sup.status, ''),
					COALESCE(sup.presence, '')
				FROM staff_accounts sa
				LEFT JOIN support_actors sup
					ON sup.staff_account_id = sa.id
				WHERE sa.id = $1::uuid
			`,
			staffID,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.FullName,
			&result.Email,
			&result.Phone,
			&result.Status,
			&result.Roles,
			&result.Permissions,
			&result.HasAdminPanelAccess,
			&result.AdminMFAStatus,
			&result.ActiveStaffSessions,
			&result.ActiveAdminSessions,
			&lastLoginAt,
			&result.CreatedAt,
			&result.UpdatedAt,
			&supportActorID,
			&supportActorCode,
			&supportActorStatus,
			&supportActorPresence,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AdminStaffDetail{},
			ErrAdminStaffNotFound
	}

	if err != nil {
		return AdminStaffDetail{},
			fmt.Errorf(
				"get Admin staff account: %w",
				err,
			)
	}

	result.LastLoginAt =
		adminTimePointer(
			lastLoginAt,
		)

	if supportActorID != "" {
		result.SupportActor =
			&AdminStaffSupportActor{
				ID: supportActorID,

				ActorCode: supportActorCode,

				Status: supportActorStatus,

				Presence: supportActorPresence,
			}
	}

	return result, nil
}

func nextAdminStaffCodeTx(
	ctx context.Context,
	tx pgx.Tx,
) (
	string,
	error,
) {
	var number int

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(
						MAX(
							substring(
								staff_code
								FROM 5
							)::integer
						),
						0
					) + 1
				FROM staff_accounts
				WHERE staff_code ~ '^EMP-[0-9]{6}$'
			`,
		).Scan(
			&number,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"generate staff code: %w",
				err,
			)
	}

	return fmt.Sprintf(
			"EMP-%06d",
			number,
		),
		nil
}

func lockAdminStaffTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	var lockedID string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM staff_accounts
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			staffID,
		).Scan(
			&lockedID,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrAdminStaffNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"lock Admin staff account: %w",
			err,
		)
	}

	return nil
}

func validateRoleCodesTx(
	ctx context.Context,
	tx pgx.Tx,
	roleCodes []string,
) error {
	if len(roleCodes) == 0 {
		return nil
	}

	var count int

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::integer
				FROM staff_roles
				WHERE code = ANY($1::text[])
			`,
			roleCodes,
		).Scan(
			&count,
		)
	if err != nil {
		return fmt.Errorf(
			"validate staff roles: %w",
			err,
		)
	}

	if count !=
		len(roleCodes) {

		return ErrAdminStaffRolesInvalid
	}

	return nil
}

func replaceStaffRolesTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
	roleCodes []string,
	assignedByStaffID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				DELETE FROM staff_account_roles
				WHERE staff_account_id = $1::uuid
			`,
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"clear staff roles: %w",
			err,
		)
	}

	if len(roleCodes) == 0 {
		return nil
	}

	_, err =
		tx.Exec(
			ctx,
			`
				INSERT INTO staff_account_roles (
					staff_account_id,
					role_id,
					assigned_by_staff_id,
					created_at
				)
				SELECT
					$1::uuid,
					sr.id,
					NULLIF($3, '')::uuid,
					now()
				FROM staff_roles sr
				WHERE sr.code = ANY($2::text[])
			`,
			staffID,
			roleCodes,
			assignedByStaffID,
		)
	if err != nil {
		return fmt.Errorf(
			"assign staff roles: %w",
			err,
		)
	}

	return nil
}

func adminStaffRoleCodesTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) (
	[]string,
	error,
) {
	rows, err :=
		tx.Query(
			ctx,
			`
				SELECT sr.code
				FROM staff_account_roles sar
				JOIN staff_roles sr
					ON sr.id = sar.role_id
				WHERE sar.staff_account_id = $1::uuid
				ORDER BY sr.code
			`,
			staffID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load staff role codes: %w",
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
					"scan staff role code: %w",
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
				"iterate staff role codes: %w",
				err,
			)
	}

	return result, nil
}

func revokeStaffSecuritySessionsTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
	reason string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_sessions
				SET
					revoked_at = now(),
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke staff sessions: %w",
			err,
		)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE admin_sessions
				SET
					revoked_at = now(),
					revoke_reason = $2,
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffID,
			reason,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke Admin sessions: %w",
			err,
		)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE admin_login_challenges
				SET
					status = 'cancelled',
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND status = 'pending'
			`,
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel Admin login challenges: %w",
			err,
		)
	}

	return nil
}

/* =========================================================
   STAFF PRIVILEGE BOUNDARIES
   ========================================================= */

func validateCreateStaffRoleAssignmentTx(
	ctx context.Context,
	tx pgx.Tx,
	actorStaffID string,
	roleCodes []string,
) error {
	actorIsSuperAdmin, err :=
		adminStaffHasRoleTx(
			ctx,
			tx,
			actorStaffID,
			RoleSuperAdmin,
		)
	if err != nil {
		return err
	}

	if actorIsSuperAdmin {
		return nil
	}

	if containsProtectedAdminRoleCode(
		roleCodes,
	) {
		return ErrAdminProtectedRoleAssignment
	}

	return validateNonSuperAdminAssignableRolesTx(
		ctx,
		tx,
		roleCodes,
	)
}

func validateStaffRoleReplacementTx(
	ctx context.Context,
	tx pgx.Tx,
	actorStaffID string,
	targetStaffID string,
	previousRoleCodes []string,
	newRoleCodes []string,
) error {
	actorIsSuperAdmin, err :=
		adminStaffHasRoleTx(
			ctx,
			tx,
			actorStaffID,
			RoleSuperAdmin,
		)
	if err != nil {
		return err
	}

	if !actorIsSuperAdmin {
		if containsProtectedAdminRoleCode(
			previousRoleCodes,
		) {
			return ErrAdminProtectedStaffMutation
		}

		if containsProtectedAdminRoleCode(
			newRoleCodes,
		) {
			return ErrAdminProtectedRoleAssignment
		}

		if err :=
			validateNonSuperAdminAssignableRolesTx(
				ctx,
				tx,
				newRoleCodes,
			); err != nil {

			return err
		}
	}

	previouslySuperAdmin :=
		containsAdminRoleCode(
			previousRoleCodes,
			RoleSuperAdmin,
		)

	willRemainSuperAdmin :=
		containsAdminRoleCode(
			newRoleCodes,
			RoleSuperAdmin,
		)

	if previouslySuperAdmin &&
		!willRemainSuperAdmin {

		if err :=
			ensureAnotherActiveSuperAdminTx(
				ctx,
				tx,
				targetStaffID,
			); err != nil {

			return err
		}
	}

	return nil
}

func validateProtectedStaffMutationTx(
	ctx context.Context,
	tx pgx.Tx,
	actorStaffID string,
	targetRoleCodes []string,
) error {
	if !containsProtectedAdminRoleCode(
		targetRoleCodes,
	) {
		return nil
	}

	actorIsSuperAdmin, err :=
		adminStaffHasRoleTx(
			ctx,
			tx,
			actorStaffID,
			RoleSuperAdmin,
		)
	if err != nil {
		return err
	}

	if !actorIsSuperAdmin {
		return ErrAdminProtectedStaffMutation
	}

	return nil
}

func validateStaffSecurityMutationTx(
	ctx context.Context,
	tx pgx.Tx,
	actorStaffID string,
	targetStaffID string,
) error {
	targetRoleCodes, err :=
		adminStaffRoleCodesTx(
			ctx,
			tx,
			targetStaffID,
		)
	if err != nil {
		return err
	}

	return validateProtectedStaffMutationTx(
		ctx,
		tx,
		actorStaffID,
		targetRoleCodes,
	)
}

func validateNonSuperAdminAssignableRolesTx(
	ctx context.Context,
	tx pgx.Tx,
	roleCodes []string,
) error {
	if len(roleCodes) == 0 {
		return nil
	}

	rows, err :=
		tx.Query(
			ctx,
			`
				SELECT
					code,
					is_system_role
				FROM staff_roles
				WHERE code = ANY($1::text[])
			`,
			roleCodes,
		)
	if err != nil {
		return fmt.Errorf(
			"load assignable staff roles: %w",
			err,
		)
	}
	defer rows.Close()

	for rows.Next() {
		var roleCode string
		var systemRole bool

		if err :=
			rows.Scan(
				&roleCode,
				&systemRole,
			); err != nil {

			return fmt.Errorf(
				"scan assignable staff role: %w",
				err,
			)
		}

		if !systemRole {
			return ErrAdminRoleAssignmentNotAllowed
		}

		if IsProtectedRoleCode(
			roleCode,
		) {
			return ErrAdminProtectedRoleAssignment
		}
	}

	if err :=
		rows.Err(); err != nil {

		return fmt.Errorf(
			"iterate assignable staff roles: %w",
			err,
		)
	}

	return nil
}

func ensureAnotherActiveSuperAdminTx(
	ctx context.Context,
	tx pgx.Tx,
	targetStaffID string,
) error {
	if _, err :=
		tx.Exec(
			ctx,
			`
				SELECT pg_advisory_xact_lock(
					hashtext(
						'commerce-admin-last-super-admin-v1'
					)
				)
			`,
		); err != nil {

		return fmt.Errorf(
			"lock last Super Admin guard: %w",
			err,
		)
	}

	var targetIsActiveSuperAdmin bool

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM staff_accounts sa
					JOIN staff_account_roles sar
						ON sar.staff_account_id = sa.id
					JOIN staff_roles sr
						ON sr.id = sar.role_id
					WHERE
						sa.id = $1::uuid
						AND sa.status = 'active'
						AND sr.code = $2
				)
			`,
			targetStaffID,
			RoleSuperAdmin,
		).Scan(
			&targetIsActiveSuperAdmin,
		); err != nil {

		return fmt.Errorf(
			"check target Super Admin state: %w",
			err,
		)
	}

	if !targetIsActiveSuperAdmin {
		return nil
	}

	var remaining int

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(
					DISTINCT sa.id
				)::integer
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
			targetStaffID,
		).Scan(
			&remaining,
		); err != nil {

		return fmt.Errorf(
			"count remaining active Super Admins: %w",
			err,
		)
	}

	if remaining < 1 {
		return ErrAdminLastSuperAdmin
	}

	return nil
}

func adminStaffHasRoleTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
	roleCode string,
) (
	bool,
	error,
) {
	var exists bool

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM staff_account_roles sar
					JOIN staff_roles sr
						ON sr.id = sar.role_id
					WHERE
						sar.staff_account_id = $1::uuid
						AND sr.code = $2
				)
			`,
			staffID,
			roleCode,
		).Scan(
			&exists,
		); err != nil {

		return false,
			fmt.Errorf(
				"check staff role %s: %w",
				roleCode,
				err,
			)
	}

	return exists, nil
}

func containsProtectedAdminRoleCode(
	roleCodes []string,
) bool {
	for _, roleCode := range roleCodes {
		if IsProtectedRoleCode(
			roleCode,
		) {
			return true
		}
	}

	return false
}

func containsAdminRoleCode(
	roleCodes []string,
	roleCode string,
) bool {
	for _, current := range roleCodes {
		if current ==
			roleCode {

			return true
		}
	}

	return false
}

func ensureStaffNotDeletedTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	var status string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT status
				FROM staff_accounts
				WHERE id = $1::uuid
			`,
			staffID,
		).Scan(
			&status,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrAdminStaffNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"check staff lifecycle state: %w",
			err,
		)
	}

	switch status {
	case "deleted":
		return ErrAdminStaffDeleted

	case "banned":
		return ErrAdminStaffBanManagedSeparately
	}

	return nil
}
func invalidateDeletedStaffMFAForStaffTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	_, err :=
		tx.Exec(
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
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"disable deleted staff Admin MFA credential: %w",
			err,
		)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				DELETE FROM admin_mfa_recovery_codes
				WHERE staff_account_id = $1::uuid
			`,
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"delete deleted staff Admin MFA recovery codes: %w",
			err,
		)
	}

	return nil
}

func disableSupportActorForStaffTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE support_actors
				SET
					status = 'disabled',
					presence = 'offline',
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND actor_type = 'human'
			`,
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"disable support actor for deleted staff: %w",
			err,
		)
	}

	return nil
}

func normalizeAdminCodes(
	values []string,
) []string {
	seen :=
		make(
			map[string]struct{},
			len(values),
		)

	result :=
		make(
			[]string,
			0,
			len(values),
		)

	for _, value := range values {

		value =
			strings.ToLower(
				strings.TrimSpace(
					value,
				),
			)

		if value == "" {
			continue
		}

		if _, exists :=
			seen[value]; exists {

			continue
		}

		seen[value] =
			struct{}{}

		result =
			append(
				result,
				value,
			)
	}

	sort.Strings(
		result,
	)

	return result
}

func isAdminUniqueViolation(
	err error,
) bool {
	var postgresError *pgconn.PgError

	return errors.As(
		err,
		&postgresError,
	) &&
		postgresError.Code ==
			"23505"
}

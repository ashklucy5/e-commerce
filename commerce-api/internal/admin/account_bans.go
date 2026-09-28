package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
)

const (
	AccountBanTargetStaff    = "staff"
	AccountBanTargetCustomer = "customer"

	AccountBanScopeFullAccount     = "full_account"
	AccountBanScopePurchasing      = "purchasing"
	AccountBanScopeProductRequests = "product_requests"
	AccountBanScopeSupportMessages = "support_messages"
	AccountBanScopeReviews         = "reviews"
	AccountBanScopeReturns         = "returns"
	AccountBanScopePromotions      = "promotions"

	AccountBanTypeTemporary = "temporary"
	AccountBanTypePermanent = "permanent"

	adminEventStaffBanCreated    = "staff_ban_created"
	adminEventStaffBanRevoked    = "staff_ban_revoked"
	adminEventCustomerBanCreated = "customer_ban_created"
	adminEventCustomerBanRevoked = "customer_ban_revoked"
)

var (
	ErrAdminBanInvalidInput = errors.New(
		"invalid account ban input",
	)

	ErrAdminBanTargetNotFound = errors.New(
		"account ban target not found",
	)

	ErrAdminBanNotFound = errors.New(
		"account ban not found",
	)

	ErrAdminBanConflict = errors.New(
		"an active account ban already exists for this scope",
	)

	ErrAdminBanAlreadyRevoked = errors.New(
		"account ban has already been revoked",
	)

	ErrAdminBanNotActive = errors.New(
		"account ban is no longer active",
	)

	ErrAdminSelfBan = errors.New(
		"cannot ban or unban your own staff account",
	)

	ErrAdminTemporaryFullAccountBanUnsupported = errors.New(
		"temporary full-account bans are not available until automatic expiry restoration is enabled",
	)

	ErrAdminBanStateConflict = errors.New(
		"account state conflicts with account ban state",
	)
)

var accountBanUUIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

var accountBanScopes = map[string]struct{}{
	AccountBanScopeFullAccount:     {},
	AccountBanScopePurchasing:      {},
	AccountBanScopeProductRequests: {},
	AccountBanScopeSupportMessages: {},
	AccountBanScopeReviews:         {},
	AccountBanScopeReturns:         {},
	AccountBanScopePromotions:      {},
}

type CreateAccountBanInput struct {
	Scope   string
	BanType string
	Reason  string

	ExpiresAt *time.Time
}

type AccountBan struct {
	ID string `json:"id"`

	TargetType string `json:"target_type"`

	StaffAccountID string `json:"staff_account_id,omitempty"`
	CustomerID     string `json:"customer_id,omitempty"`

	Scope   string `json:"scope"`
	BanType string `json:"ban_type"`

	Reason string `json:"reason"`

	PreviousAccountStatus string `json:"previous_account_status,omitempty"`

	StartsAt  time.Time  `json:"starts_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	IssuedByStaffID string `json:"issued_by_staff_id"`

	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	RevokedByStaffID string `json:"revoked_by_staff_id,omitempty"`
	RevocationReason string `json:"revocation_reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Active bool `json:"active"`
}

type accountBanScanner interface {
	Scan(dest ...any) error
}

const accountBanSelectColumns = `
	b.id::text,
	b.target_type,
	COALESCE(b.staff_account_id::text, ''),
	COALESCE(b.customer_id::text, ''),
	b.scope,
	b.ban_type,
	b.reason,
	COALESCE(b.previous_account_status, ''),
	b.starts_at,
	b.expires_at,
	b.issued_by_staff_id::text,
	b.revoked_at,
	COALESCE(b.revoked_by_staff_id::text, ''),
	COALESCE(b.revocation_reason, ''),
	b.created_at,
	b.updated_at,
	(
		b.revoked_at IS NULL
		AND b.starts_at <= now()
		AND (
			b.expires_at IS NULL
			OR b.expires_at > now()
		)
	)
`

func (s *Service) ListCustomerBans(
	ctx context.Context,
	customerID string,
) ([]AccountBan, error) {
	return s.listAccountBans(
		ctx,
		AccountBanTargetCustomer,
		customerID,
	)
}

func (s *Service) ListStaffBans(
	ctx context.Context,
	staffID string,
) ([]AccountBan, error) {
	return s.listAccountBans(
		ctx,
		AccountBanTargetStaff,
		staffID,
	)
}

func (s *Service) CreateCustomerBan(
	ctx context.Context,
	customerID string,
	input CreateAccountBanInput,
	metadata AdminActionMetadata,
) (AccountBan, error) {
	return s.createAccountBan(
		ctx,
		AccountBanTargetCustomer,
		customerID,
		input,
		metadata,
	)
}

func (s *Service) CreateStaffBan(
	ctx context.Context,
	staffID string,
	input CreateAccountBanInput,
	metadata AdminActionMetadata,
) (AccountBan, error) {
	staffID = strings.TrimSpace(staffID)

	if staffID != "" &&
		staffID == strings.TrimSpace(metadata.StaffAccountID) {

		return AccountBan{},
			ErrAdminSelfBan
	}

	return s.createAccountBan(
		ctx,
		AccountBanTargetStaff,
		staffID,
		input,
		metadata,
	)
}

func (s *Service) RevokeCustomerBan(
	ctx context.Context,
	customerID string,
	banID string,
	reason string,
	metadata AdminActionMetadata,
) (AccountBan, error) {
	return s.revokeAccountBan(
		ctx,
		AccountBanTargetCustomer,
		customerID,
		banID,
		reason,
		metadata,
	)
}

func (s *Service) RevokeStaffBan(
	ctx context.Context,
	staffID string,
	banID string,
	reason string,
	metadata AdminActionMetadata,
) (AccountBan, error) {
	staffID = strings.TrimSpace(staffID)

	if staffID != "" &&
		staffID == strings.TrimSpace(metadata.StaffAccountID) {

		return AccountBan{},
			ErrAdminSelfBan
	}

	return s.revokeAccountBan(
		ctx,
		AccountBanTargetStaff,
		staffID,
		banID,
		reason,
		metadata,
	)
}

func (s *Service) listAccountBans(
	ctx context.Context,
	targetType string,
	targetID string,
) ([]AccountBan, error) {
	targetType = strings.ToLower(
		strings.TrimSpace(targetType),
	)

	targetID = strings.TrimSpace(targetID)

	if !validAccountBanTarget(
		targetType,
		targetID,
	) {
		return nil,
			ErrAdminBanInvalidInput
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	exists, err :=
		accountBanTargetExists(
			queryCtx,
			s.db,
			targetType,
			targetID,
		)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil,
			ErrAdminBanTargetNotFound
	}

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				SELECT
			`+accountBanSelectColumns+`
				FROM account_bans b
				WHERE
					b.target_type = $1
					AND (
						(
							$1 = 'staff'
							AND b.staff_account_id = $2::uuid
						)
						OR
						(
							$1 = 'customer'
							AND b.customer_id = $2::uuid
						)
					)
				ORDER BY
					b.created_at DESC,
					b.id DESC
			`,
			targetType,
			targetID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list account bans: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]AccountBan,
			0,
		)

	for rows.Next() {
		item, err :=
			scanAccountBan(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan account ban: %w",
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
				"iterate account bans: %w",
				err,
			)
	}

	return result, nil
}

func (s *Service) createAccountBan(
	ctx context.Context,
	targetType string,
	targetID string,
	input CreateAccountBanInput,
	metadata AdminActionMetadata,
) (AccountBan, error) {
	targetType =
		strings.ToLower(
			strings.TrimSpace(
				targetType,
			),
		)

	targetID =
		strings.TrimSpace(
			targetID,
		)

	if !validAccountBanTarget(
		targetType,
		targetID,
	) {
		return AccountBan{},
			ErrAdminBanInvalidInput
	}

	if err :=
		normalizeAndValidateAccountBanInput(
			targetType,
			&input,
			time.Now(),
		); err != nil {

		return AccountBan{}, err
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AccountBan

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				currentStatus, err :=
					lockAccountBanTargetTx(
						ctx,
						tx,
						targetType,
						targetID,
					)
				if err != nil {
					return err
				}

				if targetType ==
					AccountBanTargetStaff {

					if currentStatus ==
						"deleted" {

						return ErrAdminStaffDeleted
					}

					roles, err :=
						adminStaffRoleCodesTx(
							ctx,
							tx,
							targetID,
						)
					if err != nil {
						return err
					}

					if err :=
						validateProtectedStaffMutationTx(
							ctx,
							tx,
							metadata.StaffAccountID,
							roles,
						); err != nil {

						return err
					}

					if currentStatus == "active" &&
						accountBanContainsRole(
							roles,
							RoleSuperAdmin,
						) {

						if err :=
							ensureAnotherActiveSuperAdminTx(
								ctx,
								tx,
								targetID,
							); err != nil {

							return err
						}
					}
				}

				active, err :=
					activeAccountBanExistsTx(
						ctx,
						tx,
						targetType,
						targetID,
						input.Scope,
					)
				if err != nil {
					return err
				}

				if active {
					return ErrAdminBanConflict
				}

				previousStatus := ""

				if input.Scope ==
					AccountBanScopeFullAccount {

					if err :=
						validateFullAccountBanStatus(
							targetType,
							currentStatus,
						); err != nil {

						return err
					}

					previousStatus =
						currentStatus
				}

				banID, err :=
					insertAccountBanTx(
						ctx,
						tx,
						targetType,
						targetID,
						input,
						previousStatus,
						metadata.StaffAccountID,
					)
				if err != nil {
					return err
				}

				if input.Scope ==
					AccountBanScopeFullAccount {

					if err :=
						setAccountBanTargetStatusTx(
							ctx,
							tx,
							targetType,
							targetID,
							"banned",
						); err != nil {

						return err
					}

					switch targetType {
					case AccountBanTargetStaff:
						if err :=
							revokeStaffSecuritySessionsTx(
								ctx,
								tx,
								targetID,
								"staff_account_banned",
							); err != nil {

							return err
						}

					case AccountBanTargetCustomer:
						if err :=
							revokeCustomerSessionsForBanTx(
								ctx,
								tx,
								targetID,
							); err != nil {

							return err
						}
					}
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						accountBanAuditEventType(
							targetType,
							false,
						),
						map[string]any{
							"ban_id": banID,

							"target_type": targetType,

							"target_id": targetID,

							"scope": input.Scope,

							"ban_type": input.BanType,

							"reason": input.Reason,

							"previous_account_status": previousStatus,

							"expires_at": input.ExpiresAt,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAccountBanTx(
						ctx,
						tx,
						banID,
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
		return AccountBan{}, err
	}

	return result, nil
}

func (s *Service) revokeAccountBan(
	ctx context.Context,
	targetType string,
	targetID string,
	banID string,
	reason string,
	metadata AdminActionMetadata,
) (AccountBan, error) {
	targetType =
		strings.ToLower(
			strings.TrimSpace(
				targetType,
			),
		)

	targetID =
		strings.TrimSpace(
			targetID,
		)

	banID =
		strings.TrimSpace(
			banID,
		)

	reason =
		strings.TrimSpace(
			reason,
		)

	if !validAccountBanTarget(
		targetType,
		targetID,
	) ||
		!accountBanUUIDPattern.MatchString(
			banID,
		) ||
		reason == "" ||
		utf8.RuneCountInString(
			reason,
		) > 1000 {

		return AccountBan{},
			ErrAdminBanInvalidInput
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AccountBan

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				currentStatus, err :=
					lockAccountBanTargetTx(
						ctx,
						tx,
						targetType,
						targetID,
					)
				if err != nil {
					return err
				}

				if targetType ==
					AccountBanTargetStaff {

					if currentStatus ==
						"deleted" {

						return ErrAdminStaffDeleted
					}

					roles, err :=
						adminStaffRoleCodesTx(
							ctx,
							tx,
							targetID,
						)
					if err != nil {
						return err
					}

					if err :=
						validateProtectedStaffMutationTx(
							ctx,
							tx,
							metadata.StaffAccountID,
							roles,
						); err != nil {

						return err
					}
				}

				currentBan, err :=
					lockAccountBanTx(
						ctx,
						tx,
						targetType,
						targetID,
						banID,
					)
				if err != nil {
					return err
				}

				if currentBan.RevokedAt !=
					nil {

					return ErrAdminBanAlreadyRevoked
				}

				if !currentBan.Active {
					return ErrAdminBanNotActive
				}

				restoreStatus := ""

				if currentBan.Scope ==
					AccountBanScopeFullAccount {

					if currentStatus !=
						"banned" {

						return ErrAdminBanStateConflict
					}

					if err :=
						validateAccountBanRestoreStatus(
							targetType,
							currentBan.PreviousAccountStatus,
						); err != nil {

						return err
					}

					restoreStatus =
						currentBan.PreviousAccountStatus
				}

				tag, err :=
					tx.Exec(
						ctx,
						`
							UPDATE account_bans
							SET
								revoked_at = now(),
								revoked_by_staff_id = $2::uuid,
								revocation_reason = $3,
								updated_at = now()
							WHERE
								id = $1::uuid
								AND revoked_at IS NULL
						`,
						banID,
						metadata.StaffAccountID,
						reason,
					)
				if err != nil {
					return fmt.Errorf(
						"revoke account ban: %w",
						err,
					)
				}

				if tag.RowsAffected() != 1 {
					return ErrAdminBanAlreadyRevoked
				}

				if currentBan.Scope ==
					AccountBanScopeFullAccount {

					if err :=
						setAccountBanTargetStatusTx(
							ctx,
							tx,
							targetType,
							targetID,
							restoreStatus,
						); err != nil {

						return err
					}
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						accountBanAuditEventType(
							targetType,
							true,
						),
						map[string]any{
							"ban_id": banID,

							"target_type": targetType,

							"target_id": targetID,

							"scope": currentBan.Scope,

							"ban_type": currentBan.BanType,

							"original_reason": currentBan.Reason,

							"revocation_reason": reason,

							"restored_account_status": restoreStatus,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAccountBanTx(
						ctx,
						tx,
						banID,
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
		return AccountBan{}, err
	}

	return result, nil
}

func normalizeAndValidateAccountBanInput(
	targetType string,
	input *CreateAccountBanInput,
	now time.Time,
) error {
	input.Scope =
		strings.ToLower(
			strings.TrimSpace(
				input.Scope,
			),
		)

	input.BanType =
		strings.ToLower(
			strings.TrimSpace(
				input.BanType,
			),
		)

	input.Reason =
		strings.TrimSpace(
			input.Reason,
		)

	if _, exists :=
		accountBanScopes[input.Scope]; !exists {

		return ErrAdminBanInvalidInput
	}

	if targetType ==
		AccountBanTargetStaff &&
		input.Scope !=
			AccountBanScopeFullAccount {

		return ErrAdminBanInvalidInput
	}

	if input.Reason == "" ||
		utf8.RuneCountInString(
			input.Reason,
		) > 1000 {

		return ErrAdminBanInvalidInput
	}

	switch input.BanType {
	case AccountBanTypePermanent:
		if input.ExpiresAt != nil {
			return ErrAdminBanInvalidInput
		}

	case AccountBanTypeTemporary:
		if input.Scope ==
			AccountBanScopeFullAccount {

			return ErrAdminTemporaryFullAccountBanUnsupported
		}

		if input.ExpiresAt == nil ||
			!input.ExpiresAt.After(
				now,
			) {

			return ErrAdminBanInvalidInput
		}

	default:
		return ErrAdminBanInvalidInput
	}

	return nil
}

func validAccountBanTarget(
	targetType string,
	targetID string,
) bool {
	if targetType !=
		AccountBanTargetStaff &&
		targetType !=
			AccountBanTargetCustomer {

		return false
	}

	return accountBanUUIDPattern.MatchString(
		strings.TrimSpace(
			targetID,
		),
	)
}

func accountBanTargetExists(
	ctx context.Context,
	querier interface {
		QueryRow(
			context.Context,
			string,
			...any,
		) pgx.Row
	},
	targetType string,
	targetID string,
) (bool, error) {
	var exists bool

	var query string

	switch targetType {
	case AccountBanTargetStaff:
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM staff_accounts
				WHERE id = $1::uuid
			)
		`

	case AccountBanTargetCustomer:
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM customers
				WHERE id = $1::uuid
			)
		`

	default:
		return false,
			ErrAdminBanInvalidInput
	}

	if err :=
		querier.QueryRow(
			ctx,
			query,
			targetID,
		).Scan(
			&exists,
		); err != nil {

		return false,
			fmt.Errorf(
				"check account ban target: %w",
				err,
			)
	}

	return exists, nil
}

func lockAccountBanTargetTx(
	ctx context.Context,
	tx pgx.Tx,
	targetType string,
	targetID string,
) (string, error) {
	var status string

	var query string

	switch targetType {
	case AccountBanTargetStaff:
		query = `
			SELECT status
			FROM staff_accounts
			WHERE id = $1::uuid
			FOR UPDATE
		`

	case AccountBanTargetCustomer:
		query = `
			SELECT status
			FROM customers
			WHERE id = $1::uuid
			FOR UPDATE
		`

	default:
		return "",
			ErrAdminBanInvalidInput
	}

	err :=
		tx.QueryRow(
			ctx,
			query,
			targetID,
		).Scan(
			&status,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return "",
			ErrAdminBanTargetNotFound
	}

	if err != nil {
		return "",
			fmt.Errorf(
				"lock account ban target: %w",
				err,
			)
	}

	return status, nil
}

func activeAccountBanExistsTx(
	ctx context.Context,
	tx pgx.Tx,
	targetType string,
	targetID string,
	scope string,
) (bool, error) {
	var exists bool

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM account_bans
					WHERE
						target_type = $1
						AND scope = $3
						AND revoked_at IS NULL
						AND starts_at <= now()
						AND (
							expires_at IS NULL
							OR expires_at > now()
						)
						AND (
							(
								$1 = 'staff'
								AND staff_account_id = $2::uuid
							)
							OR
							(
								$1 = 'customer'
								AND customer_id = $2::uuid
							)
						)
				)
			`,
			targetType,
			targetID,
			scope,
		).Scan(
			&exists,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"check active account ban: %w",
				err,
			)
	}

	return exists, nil
}

func validateFullAccountBanStatus(
	targetType string,
	status string,
) error {
	switch targetType {
	case AccountBanTargetStaff:
		switch status {
		case "active",
			"suspended",
			"disabled":

			return nil

		case "deleted":
			return ErrAdminStaffDeleted

		default:
			return ErrAdminBanStateConflict
		}

	case AccountBanTargetCustomer:
		switch status {
		case "active",
			"disabled":

			return nil

		default:
			return ErrAdminBanStateConflict
		}

	default:
		return ErrAdminBanInvalidInput
	}
}

func validateAccountBanRestoreStatus(
	targetType string,
	status string,
) error {
	switch targetType {
	case AccountBanTargetStaff:
		switch status {
		case "active",
			"suspended",
			"disabled":

			return nil
		}

	case AccountBanTargetCustomer:
		switch status {
		case "active",
			"disabled":

			return nil
		}
	}

	return ErrAdminBanStateConflict
}

func insertAccountBanTx(
	ctx context.Context,
	tx pgx.Tx,
	targetType string,
	targetID string,
	input CreateAccountBanInput,
	previousStatus string,
	issuedByStaffID string,
) (string, error) {
	var banID string

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO account_bans (
					target_type,
					staff_account_id,
					customer_id,
					scope,
					ban_type,
					reason,
					previous_account_status,
					starts_at,
					expires_at,
					issued_by_staff_id,
					created_at,
					updated_at
				)
				VALUES (
					$1::varchar(20),
					CASE
						WHEN $1::varchar(20) = 'staff'
							THEN $2::uuid
						ELSE NULL
					END,
					CASE
						WHEN $1::varchar(20) = 'customer'
							THEN $2::uuid
						ELSE NULL
					END,
					$3,
					$4,
					$5,
					NULLIF($6, ''),
					now(),
					$7,
					$8::uuid,
					now(),
					now()
				)
				RETURNING id::text
			`,
			targetType,
			targetID,
			input.Scope,
			input.BanType,
			input.Reason,
			previousStatus,
			input.ExpiresAt,
			issuedByStaffID,
		).Scan(
			&banID,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"insert account ban: %w",
				err,
			)
	}

	return banID, nil
}

func setAccountBanTargetStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	targetType string,
	targetID string,
	status string,
) error {
	var query string

	switch targetType {
	case AccountBanTargetStaff:
		query = `
			UPDATE staff_accounts
			SET
				status = $2,
				updated_at = now()
			WHERE id = $1::uuid
		`

	case AccountBanTargetCustomer:
		query = `
			UPDATE customers
			SET
				status = $2,
				updated_at = now()
			WHERE id = $1::uuid
		`

	default:
		return ErrAdminBanInvalidInput
	}

	tag, err :=
		tx.Exec(
			ctx,
			query,
			targetID,
			status,
		)
	if err != nil {
		return fmt.Errorf(
			"update account ban target status: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrAdminBanTargetNotFound
	}

	return nil
}

func revokeCustomerSessionsForBanTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE auth_sessions
				SET
					revoked_at = now(),
					updated_at = now()
				WHERE
					customer_id = $1::uuid
					AND revoked_at IS NULL
			`,
			customerID,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke customer sessions after account ban: %w",
			err,
		)
	}

	return nil
}

func lockAccountBanTx(
	ctx context.Context,
	tx pgx.Tx,
	targetType string,
	targetID string,
	banID string,
) (AccountBan, error) {
	item, err :=
		scanAccountBan(
			tx.QueryRow(
				ctx,
				`
					SELECT
				`+accountBanSelectColumns+`
					FROM account_bans b
					WHERE
						b.id = $1::uuid
						AND b.target_type = $2
						AND (
							(
								$2 = 'staff'
								AND b.staff_account_id = $3::uuid
							)
							OR
							(
								$2 = 'customer'
								AND b.customer_id = $3::uuid
							)
						)
					FOR UPDATE
				`,
				banID,
				targetType,
				targetID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AccountBan{},
			ErrAdminBanNotFound
	}

	if err != nil {
		return AccountBan{},
			fmt.Errorf(
				"lock account ban: %w",
				err,
			)
	}

	return item, nil
}

func getAccountBanTx(
	ctx context.Context,
	tx pgx.Tx,
	banID string,
) (AccountBan, error) {
	item, err :=
		scanAccountBan(
			tx.QueryRow(
				ctx,
				`
					SELECT
				`+accountBanSelectColumns+`
					FROM account_bans b
					WHERE b.id = $1::uuid
				`,
				banID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AccountBan{},
			ErrAdminBanNotFound
	}

	if err != nil {
		return AccountBan{},
			fmt.Errorf(
				"get account ban: %w",
				err,
			)
	}

	return item, nil
}

func scanAccountBan(
	row accountBanScanner,
) (AccountBan, error) {
	var result AccountBan

	var expiresAt pgtype.Timestamptz
	var revokedAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.TargetType,
			&result.StaffAccountID,
			&result.CustomerID,
			&result.Scope,
			&result.BanType,
			&result.Reason,
			&result.PreviousAccountStatus,
			&result.StartsAt,
			&expiresAt,
			&result.IssuedByStaffID,
			&revokedAt,
			&result.RevokedByStaffID,
			&result.RevocationReason,
			&result.CreatedAt,
			&result.UpdatedAt,
			&result.Active,
		)
	if err != nil {
		return AccountBan{}, err
	}

	if expiresAt.Valid {
		value :=
			expiresAt.Time

		result.ExpiresAt =
			&value
	}

	if revokedAt.Valid {
		value :=
			revokedAt.Time

		result.RevokedAt =
			&value
	}

	return result, nil
}

func accountBanContainsRole(
	roles []string,
	roleCode string,
) bool {
	for _, current := range roles {

		if current ==
			roleCode {

			return true
		}
	}

	return false
}

func accountBanAuditEventType(
	targetType string,
	revoke bool,
) string {
	switch targetType {
	case AccountBanTargetStaff:
		if revoke {
			return adminEventStaffBanRevoked
		}

		return adminEventStaffBanCreated

	case AccountBanTargetCustomer:
		if revoke {
			return adminEventCustomerBanRevoked
		}

		return adminEventCustomerBanCreated
	}

	return "account_ban_changed"
}

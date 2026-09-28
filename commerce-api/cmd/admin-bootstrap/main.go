package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/adminauth"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	
)

type bootstrapInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

type bootstrapResult struct {
	StaffID   string
	StaffCode string
	Role      string
}

func main() {
	log.SetFlags(
		0,
	)

	input, err :=
		loadInput()
	if err != nil {
		log.Fatal(
			err,
		)
	}

	cfg, err :=
		config.Load()
	if err != nil {
		log.Fatalf(
			"load configuration: %v",
			err,
		)
	}

	if strings.EqualFold(
		cfg.AppEnv,
		"production",
	) {
		log.Fatal(
			"admin-bootstrap is disabled in production",
		)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
	defer cancel()

	db, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		log.Fatalf(
			"connect postgres: %v",
			err,
		)
	}
	defer db.Close()

	result, err :=
		bootstrap(
			ctx,
			db,
			input,
		)
	if err != nil {
		log.Fatalf(
			"bootstrap admin: %v",
			err,
		)
	}

	fmt.Println(
		"Admin bootstrap complete.",
	)

	fmt.Printf(
		"Staff ID: %s\n",
		result.StaffID,
	)

	fmt.Printf(
		"Staff code: %s\n",
		result.StaffCode,
	)

	fmt.Printf(
		"Role: %s\n",
		result.Role,
	)

	fmt.Printf(
		"Permissions: %d\n",
		len(
			admin.Permissions,
		),
	)

	fmt.Println(
		"Password was not printed.",
	)
}

func loadInput() (
	bootstrapInput,
	error,
) {
	name :=
		strings.TrimSpace(
			os.Getenv(
				"ADMIN_BOOTSTRAP_NAME",
			),
		)

	email :=
		strings.ToLower(
			strings.TrimSpace(
				os.Getenv(
					"ADMIN_BOOTSTRAP_EMAIL",
				),
			),
		)

	phone :=
		strings.TrimSpace(
			os.Getenv(
				"ADMIN_BOOTSTRAP_PHONE",
			),
		)

	password :=
		os.Getenv(
			"ADMIN_BOOTSTRAP_PASSWORD",
		)

	if name == "" {
		return bootstrapInput{},
			errors.New(
				"ADMIN_BOOTSTRAP_NAME is required",
			)
	}

	if email == "" {
		return bootstrapInput{},
			errors.New(
				"ADMIN_BOOTSTRAP_EMAIL is required",
			)
	}

	parsedEmail, err :=
		mail.ParseAddress(
			email,
		)
	if err != nil ||
		!strings.EqualFold(
			parsedEmail.Address,
			email,
		) {

		return bootstrapInput{},
			errors.New(
				"ADMIN_BOOTSTRAP_EMAIL must be a valid email address",
			)
	}

	if err :=
		adminauth.ValidateNewPassword(
			password,
		); err != nil {

		return bootstrapInput{},
			fmt.Errorf(
				"ADMIN_BOOTSTRAP_PASSWORD: %w",
				err,
			)
	}

	return bootstrapInput{
			Name: name,

			Email: email,

			Phone: phone,

			Password: password,
		},
		nil
}

func bootstrap(
	ctx context.Context,
	db *pgxpool.Pool,
	input bootstrapInput,
) (
	bootstrapResult,
	error,
) {
	passwordHash, err :=
		adminauth.HashNewPassword(
			input.Password,
		)
	if err != nil {
		return bootstrapResult{},
			fmt.Errorf(
				"hash admin password: %w",
				err,
			)
	}

	var result bootstrapResult

	err =
		database.WithinTx(
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
									'commerce-admin-bootstrap-v1'
								)
							)
						`,
					); err != nil {

					return fmt.Errorf(
						"lock admin bootstrap: %w",
						err,
					)
				}

				if err :=
					seedPermissions(
						ctx,
						tx,
					); err != nil {

					return err
				}

				if err :=
					seedSuperAdminRole(
						ctx,
						tx,
					); err != nil {

					return err
				}

				staffID,
					staffCode,
					err :=
					ensureStaffAccount(
						ctx,
						tx,
						input,
						passwordHash,
					)
				if err != nil {
					return err
				}

				if err :=
					assignRole(
						ctx,
						tx,
						staffID,
						admin.RoleSuperAdmin,
					); err != nil {

					return err
				}

				if err :=
					revokeExistingSessions(
						ctx,
						tx,
						staffID,
					); err != nil {

					return err
				}

				result =
					bootstrapResult{
						StaffID: staffID,

						StaffCode: staffCode,

						Role: admin.RoleSuperAdmin,
					}

				return nil
			},
		)
	if err != nil {
		return bootstrapResult{},
			err
	}

	return result,
		nil
}

func seedPermissions(
	ctx context.Context,
	tx pgx.Tx,
) error {
	for _, permission := range admin.Permissions {

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
				"seed Admin permission %s: %w",
				permission.Code,
				err,
			)
		}
	}

	return nil
}

func seedSuperAdminRole(
	ctx context.Context,
	tx pgx.Tx,
) error {
	role :=
		admin.SuperAdminRole

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
			"seed Admin superuser role: %w",
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
				"assign Admin permission %s: %w",
				permissionCode,
				err,
			)
		}

		if tag.RowsAffected() == 0 {
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
					"verify Admin permission %s: %w",
					permissionCode,
					err,
				)
			}

			if !exists {
				return fmt.Errorf(
					"Admin permission %s does not exist",
					permissionCode,
				)
			}
		}
	}

	return nil
}

func ensureStaffAccount(
	ctx context.Context,
	tx pgx.Tx,
	input bootstrapInput,
	passwordHash string,
) (
	string,
	string,
	error,
) {
	var staffID string
	var staffCode string
	var status string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					staff_code,
					status
				FROM staff_accounts
				WHERE lower(email) = lower($1)
				LIMIT 1
				FOR UPDATE
			`,
			input.Email,
		).Scan(
			&staffID,
			&staffCode,
			&status,
		)

	if err == nil {
		if status != "active" {
			return "",
				"",
				fmt.Errorf(
					"existing staff account %s is %s",
					staffCode,
					status,
				)
		}

		if _, err :=
			tx.Exec(
				ctx,
				`
					UPDATE staff_accounts
					SET
						full_name = $2,
						phone = $3,
						password_hash = $4,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				staffID,
				input.Name,
				nullableString(
					input.Phone,
				),
				passwordHash,
			); err != nil {

			return "",
				"",
				fmt.Errorf(
					"update Admin bootstrap staff account: %w",
					err,
				)
		}

		return staffID,
			staffCode,
			nil
	}

	if !errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return "",
			"",
			fmt.Errorf(
				"find Admin bootstrap staff account: %w",
				err,
			)
	}

	nextNumber, err :=
		nextStaffNumber(
			ctx,
			tx,
		)
	if err != nil {
		return "",
			"",
			err
	}

	staffCode =
		fmt.Sprintf(
			"EMP-%06d",
			nextNumber,
		)

	if err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO staff_accounts (
					staff_code,
					full_name,
					email,
					phone,
					password_hash,
					status,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					'active',
					now(),
					now()
				)
				RETURNING id::text
			`,
			staffCode,
			input.Name,
			input.Email,
			nullableString(
				input.Phone,
			),
			passwordHash,
		).Scan(
			&staffID,
		); err != nil {

		return "",
			"",
			fmt.Errorf(
				"create Admin bootstrap staff account: %w",
				err,
			)
	}

	return staffID,
		staffCode,
		nil
}

func nextStaffNumber(
	ctx context.Context,
	tx pgx.Tx,
) (
	int,
	error,
) {
	var number int

	if err :=
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
		); err != nil {

		return 0,
			fmt.Errorf(
				"generate staff code: %w",
				err,
			)
	}

	return number,
		nil
}

func assignRole(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
	roleCode string,
) error {
	tag, err :=
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
					r.id,
					NULL,
					now()
				FROM staff_roles r
				WHERE r.code = $2
				ON CONFLICT DO NOTHING
			`,
			staffID,
			roleCode,
		)
	if err != nil {
		return fmt.Errorf(
			"assign Admin role %s: %w",
			roleCode,
			err,
		)
	}

	if tag.RowsAffected() > 0 {
		return nil
	}

	var exists bool

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM staff_account_roles ar
					JOIN staff_roles r
						ON r.id = ar.role_id
					WHERE
						ar.staff_account_id = $1::uuid
						AND r.code = $2
				)
			`,
			staffID,
			roleCode,
		).Scan(
			&exists,
		); err != nil {

		return fmt.Errorf(
			"verify Admin role %s: %w",
			roleCode,
			err,
		)
	}

	if !exists {
		return fmt.Errorf(
			"Admin role %s does not exist",
			roleCode,
		)
	}

	return nil
}

func revokeExistingSessions(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	if _, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_sessions
				SET
					revoked_at = COALESCE(
						revoked_at,
						now()
					),
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffID,
		); err != nil {

		return fmt.Errorf(
			"revoke existing staff sessions after Admin bootstrap password reset: %w",
			err,
		)
	}

	return nil
}

func nullableString(
	value string,
) any {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil
	}

	return value
}

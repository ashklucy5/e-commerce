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
	"golang.org/x/crypto/bcrypt"

	admincore "project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/support"
)

const (
	aiActorCode = "SUP-AI-000001"

	systemActorCode = "SUP-SYS-000001"
)

type permissionSeed struct {
	Code        string
	Description string
}

type roleSeed struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

type queueSeed struct {
	Code        string
	Name        string
	Description string
	SortOrder   int
}

type bootstrapInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

type bootstrapResult struct {
	StaffID          string
	StaffCode        string
	SupportActorID   string
	SupportActorCode string
	QueueCount       int
}

var permissions = []permissionSeed{
	{
		Code:        support.PermissionPanelAccess,
		Description: "Access the Support Panel",
	},
	{
		Code:        support.PermissionQueueRead,
		Description: "View support queues",
	},
	{
		Code:        support.PermissionQueueManage,
		Description: "Manage support queues and memberships",
	},
	{
		Code:        support.PermissionPresenceUpdate,
		Description: "Update own support presence",
	},
	{
		Code:        support.PermissionCaseRead,
		Description: "Read customer support cases",
	},
	{
		Code:        support.PermissionCaseReply,
		Description: "Reply to customer support cases",
	},
	{
		Code:        support.PermissionCaseClaim,
		Description: "Claim unassigned support cases",
	},
	{
		Code:        support.PermissionCaseAssign,
		Description: "Assign or reassign support cases",
	},
	{
		Code:        support.PermissionCaseEscalate,
		Description: "Escalate customer support cases",
	},
	{
		Code:        support.PermissionCaseResolve,
		Description: "Resolve customer support cases",
	},
	{
		Code:        support.PermissionCasePriority,
		Description: "Change customer support case priority",
	},
	{
		Code:        support.PermissionAgentRead,
		Description: "View support agents",
	},
	{
		Code:        support.PermissionAgentManage,
		Description: "Manage support agents",
	},

	// Admin-panel permissions required by support staff.
	{
		Code:        admincore.PermissionPanelAccess,
		Description: "Access the Admin Panel",
	},
	{
		Code:        admincore.PermissionDashboardRead,
		Description: "View Admin dashboard summaries",
	},
	{
		Code:        admincore.PermissionCustomerRead,
		Description: "View customer accounts and customer information",
	},
	{
		Code:        admincore.PermissionOrderRead,
		Description: "View customer orders",
	},
	{
		Code:        admincore.PermissionCRMRead,
		Description: "View customer-service CRM cases",
	},
	{
		Code:        admincore.PermissionCRMManage,
		Description: "Perform authorized CRM case operations",
	},

	// Product-request / sourcing permissions.
	{
		Code:        admincore.PermissionSourcingRead,
		Description: "View customer product-sourcing requests, offers, and agreements",
	},
	{
		Code:        admincore.PermissionSourcingReview,
		Description: "Review, hold, accept, or cancel customer product-sourcing requests",
	},
	{
		Code:        admincore.PermissionSourcingOfferManage,
		Description: "Create, edit, send, and manage product-sourcing commercial offers",
	},
	{
		Code:        admincore.PermissionSourcingFinalize,
		Description: "Finalize customer-accepted product-sourcing agreements",
	},
}

var roles = []roleSeed{
	{
		Code: support.RoleAgent,

		Name: "Support Agent",

		Description: "Human customer-support agent with Admin-panel support and product-sourcing access",

		Permissions: []string{
			// Existing Support Panel permissions.
			support.PermissionPanelAccess,
			support.PermissionQueueRead,
			support.PermissionPresenceUpdate,
			support.PermissionCaseRead,
			support.PermissionCaseReply,
			support.PermissionCaseClaim,
			support.PermissionCaseEscalate,
			support.PermissionCaseResolve,

			// Admin Panel access.
			admincore.PermissionPanelAccess,
			admincore.PermissionDashboardRead,

			// Read-only customer/order context.
			admincore.PermissionCustomerRead,
			admincore.PermissionOrderRead,

			// CRM.
			admincore.PermissionCRMRead,
			admincore.PermissionCRMManage,

			// Product sourcing.
			admincore.PermissionSourcingRead,
			admincore.PermissionSourcingReview,
			admincore.PermissionSourcingOfferManage,

			// Intentionally no PermissionSourcingFinalize.
		},
	},
	{
		Code: support.RoleSupervisor,

		Name: "Support Supervisor",

		Description: "Support supervisor with queue management, assignment, product sourcing, and sourcing-finalization authority",

		Permissions: []string{
			// Existing Support Panel permissions.
			support.PermissionPanelAccess,
			support.PermissionQueueRead,
			support.PermissionQueueManage,
			support.PermissionPresenceUpdate,
			support.PermissionCaseRead,
			support.PermissionCaseReply,
			support.PermissionCaseClaim,
			support.PermissionCaseAssign,
			support.PermissionCaseEscalate,
			support.PermissionCaseResolve,
			support.PermissionCasePriority,
			support.PermissionAgentRead,
			support.PermissionAgentManage,

			// Admin Panel access.
			admincore.PermissionPanelAccess,
			admincore.PermissionDashboardRead,

			// Customer/order context.
			admincore.PermissionCustomerRead,
			admincore.PermissionOrderRead,

			// CRM.
			admincore.PermissionCRMRead,
			admincore.PermissionCRMManage,

			// Product sourcing.
			admincore.PermissionSourcingRead,
			admincore.PermissionSourcingReview,
			admincore.PermissionSourcingOfferManage,
			admincore.PermissionSourcingFinalize,
		},
	},
}

var queues = []queueSeed{
	{
		Code:        "general",
		Name:        "General Support",
		Description: "General customer questions and assistance",
		SortOrder:   10,
	},
	{
		Code:        "bulk_sales",
		Name:        "Bulk Sales",
		Description: "B2B quantity requests, stock escalation, and product-sourcing enquiries",
		SortOrder:   20,
	},
	{
		Code:        "orders",
		Name:        "Orders",
		Description: "Order-related questions and problems",
		SortOrder:   30,
	},
	{
		Code:        "payments",
		Name:        "Payments",
		Description: "Payment questions, failures, and disputes",
		SortOrder:   40,
	},
	{
		Code:        "returns_refunds",
		Name:        "Returns & Refunds",
		Description: "Returns, cancellations, and refund assistance",
		SortOrder:   50,
	},
	{
		Code:        "delivery",
		Name:        "Delivery",
		Description: "Delivery and shipment-related customer support",
		SortOrder:   60,
	},
	{
		Code:        "complaints",
		Name:        "Complaints",
		Description: "Customer complaints and inconvenience cases",
		SortOrder:   70,
	},
}

func main() {
	log.SetFlags(0)

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
			"support-bootstrap is disabled in production",
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
			"bootstrap support: %v",
			err,
		)
	}

	fmt.Println(
		"Support bootstrap complete.",
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
		"Support actor ID: %s\n",
		result.SupportActorID,
	)

	fmt.Printf(
		"Support actor code: %s\n",
		result.SupportActorCode,
	)

	fmt.Printf(
		"Role: %s\n",
		support.RoleSupervisor,
	)

	fmt.Printf(
		"Queues: %d\n",
		result.QueueCount,
	)

	fmt.Printf(
		"AI actor: %s\n",
		aiActorCode,
	)

	fmt.Printf(
		"System actor: %s\n",
		systemActorCode,
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
				"SUPPORT_BOOTSTRAP_NAME",
			),
		)

	email :=
		strings.ToLower(
			strings.TrimSpace(
				os.Getenv(
					"SUPPORT_BOOTSTRAP_EMAIL",
				),
			),
		)

	phone :=
		strings.TrimSpace(
			os.Getenv(
				"SUPPORT_BOOTSTRAP_PHONE",
			),
		)

	password :=
		os.Getenv(
			"SUPPORT_BOOTSTRAP_PASSWORD",
		)

	if name == "" {
		return bootstrapInput{},
			errors.New(
				"SUPPORT_BOOTSTRAP_NAME is required",
			)
	}

	if email == "" {
		return bootstrapInput{},
			errors.New(
				"SUPPORT_BOOTSTRAP_EMAIL is required",
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
				"SUPPORT_BOOTSTRAP_EMAIL must be a valid email address",
			)
	}

	if len(password) < 12 {
		return bootstrapInput{},
			errors.New(
				"SUPPORT_BOOTSTRAP_PASSWORD must be at least 12 characters",
			)
	}

	return bootstrapInput{
		Name:     name,
		Email:    email,
		Phone:    phone,
		Password: password,
	}, nil
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
		bcrypt.GenerateFromPassword(
			[]byte(
				input.Password,
			),
			bcrypt.DefaultCost,
		)
	if err != nil {
		return bootstrapResult{},
			fmt.Errorf(
				"hash support password: %w",
				err,
			)
	}

	tx, err :=
		db.Begin(
			ctx,
		)
	if err != nil {
		return bootstrapResult{},
			fmt.Errorf(
				"begin support bootstrap: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	if _, err :=
		tx.Exec(
			ctx,
			`
				SELECT pg_advisory_xact_lock(
					hashtext(
						'commerce-support-bootstrap-v2'
					)
				)
			`,
		); err != nil {
		return bootstrapResult{},
			fmt.Errorf(
				"lock support bootstrap: %w",
				err,
			)
	}

	if err :=
		seedPermissions(
			ctx,
			tx,
		); err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		seedRoles(
			ctx,
			tx,
		); err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		seedQueues(
			ctx,
			tx,
		); err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		ensureNonHumanActor(
			ctx,
			tx,
			aiActorCode,
			"ai",
			"AI Support Agent",
			100,
		); err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		ensureNonHumanActor(
			ctx,
			tx,
			systemActorCode,
			"system",
			"Support System",
			1,
		); err != nil {
		return bootstrapResult{},
			err
	}

	staffID,
		staffCode,
		err :=
		ensureStaffAccount(
			ctx,
			tx,
			input,
			string(
				passwordHash,
			),
		)
	if err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		assignRole(
			ctx,
			tx,
			staffID,
			support.RoleSupervisor,
		); err != nil {
		return bootstrapResult{},
			err
	}

	actorID,
		actorCode,
		err :=
		ensureHumanSupportActor(
			ctx,
			tx,
			staffID,
			input.Name,
		)
	if err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		assignQueues(
			ctx,
			tx,
			actorID,
		); err != nil {
		return bootstrapResult{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return bootstrapResult{},
			fmt.Errorf(
				"commit support bootstrap: %w",
				err,
			)
	}

	return bootstrapResult{
		StaffID:          staffID,
		StaffCode:        staffCode,
		SupportActorID:   actorID,
		SupportActorCode: actorCode,
		QueueCount: len(
			queues,
		),
	}, nil
}

func seedPermissions(
	ctx context.Context,
	tx pgx.Tx,
) error {
	for _, permission := range permissions {
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
				"seed permission %s: %w",
				permission.Code,
				err,
			)
		}
	}

	return nil
}

func seedRoles(
	ctx context.Context,
	tx pgx.Tx,
) error {
	for _, role := range roles {
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
				"seed role %s: %w",
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
				"clear permissions for role %s: %w",
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
					"assign permission %s to role %s: %w",
					permissionCode,
					role.Code,
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
						"verify permission %s for role %s: %w",
						permissionCode,
						role.Code,
						err,
					)
				}

				if !exists {
					return fmt.Errorf(
						"permission %s not found while seeding role %s",
						permissionCode,
						role.Code,
					)
				}
			}
		}
	}

	return nil
}

func seedQueues(
	ctx context.Context,
	tx pgx.Tx,
) error {
	for _, queue := range queues {
		if _, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO support_queues (
						code,
						name,
						description,
						status,
						sort_order,
						created_at,
						updated_at
					)
					VALUES (
						$1,
						$2,
						$3,
						'active',
						$4,
						now(),
						now()
					)
					ON CONFLICT (code)
					DO UPDATE SET
						name = EXCLUDED.name,
						description = EXCLUDED.description,
						status = 'active',
						sort_order = EXCLUDED.sort_order,
						updated_at = now()
				`,
				queue.Code,
				queue.Name,
				queue.Description,
				queue.SortOrder,
			); err != nil {
			return fmt.Errorf(
				"seed queue %s: %w",
				queue.Code,
				err,
			)
		}
	}

	return nil
}

func ensureNonHumanActor(
	ctx context.Context,
	tx pgx.Tx,
	actorCode string,
	actorType string,
	displayName string,
	maxActiveCases int,
) error {
	var existingType string
	var staffAccountID *string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					actor_type,
					staff_account_id::text
				FROM support_actors
				WHERE actor_code = $1
				FOR UPDATE
			`,
			actorCode,
		).Scan(
			&existingType,
			&staffAccountID,
		)

	switch {
	case errors.Is(
		err,
		pgx.ErrNoRows,
	):
		_, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO support_actors (
						actor_code,
						actor_type,
						staff_account_id,
						display_name,
						status,
						presence,
						max_active_cases,
						created_at,
						updated_at
					)
					VALUES (
						$1,
						$2,
						NULL,
						$3,
						'active',
						'offline',
						$4,
						now(),
						now()
					)
				`,
				actorCode,
				actorType,
				displayName,
				maxActiveCases,
			)
		if err != nil {
			return fmt.Errorf(
				"create support actor %s: %w",
				actorCode,
				err,
			)
		}

		return nil

	case err != nil:
		return fmt.Errorf(
			"load support actor %s: %w",
			actorCode,
			err,
		)
	}

	if existingType != actorType ||
		staffAccountID != nil {
		return fmt.Errorf(
			"support actor code %s is already used by an incompatible actor",
			actorCode,
		)
	}

	if _, err :=
		tx.Exec(
			ctx,
			`
				UPDATE support_actors
				SET
					display_name = $2,
					status = 'active',
					max_active_cases = $3,
					updated_at = now()
				WHERE actor_code = $1
			`,
			actorCode,
			displayName,
			maxActiveCases,
		); err != nil {
		return fmt.Errorf(
			"update support actor %s: %w",
			actorCode,
			err,
		)
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
					"update bootstrap staff account: %w",
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
				"find bootstrap staff account: %w",
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
				"create bootstrap staff account: %w",
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

	return number, nil
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
			"assign staff role %s: %w",
			roleCode,
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
				"verify staff role %s: %w",
				roleCode,
				err,
			)
		}

		if !exists {
			return fmt.Errorf(
				"staff role %s does not exist",
				roleCode,
			)
		}
	}

	return nil
}

func ensureHumanSupportActor(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
	displayName string,
) (
	string,
	string,
	error,
) {
	var actorID string
	var actorCode string
	var status string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					actor_code,
					status
				FROM support_actors
				WHERE staff_account_id = $1::uuid
				LIMIT 1
				FOR UPDATE
			`,
			staffID,
		).Scan(
			&actorID,
			&actorCode,
			&status,
		)

	if err == nil {
		if status != "active" {
			return "",
				"",
				fmt.Errorf(
					"existing support actor %s is %s",
					actorCode,
					status,
				)
		}

		if _, err :=
			tx.Exec(
				ctx,
				`
					UPDATE support_actors
					SET
						display_name = $2,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				actorID,
				displayName,
			); err != nil {
			return "",
				"",
				fmt.Errorf(
					"update human support actor: %w",
					err,
				)
		}

		return actorID,
			actorCode,
			nil
	}

	if !errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return "",
			"",
			fmt.Errorf(
				"find human support actor: %w",
				err,
			)
	}

	nextNumber, err :=
		nextHumanActorNumber(
			ctx,
			tx,
		)
	if err != nil {
		return "",
			"",
			err
	}

	actorCode =
		fmt.Sprintf(
			"SUP-H-%06d",
			nextNumber,
		)

	if err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO support_actors (
					actor_code,
					actor_type,
					staff_account_id,
					display_name,
					status,
					presence,
					max_active_cases,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					'human',
					$2::uuid,
					$3,
					'active',
					'offline',
					10,
					now(),
					now()
				)
				RETURNING id::text
			`,
			actorCode,
			staffID,
			displayName,
		).Scan(
			&actorID,
		); err != nil {
		return "",
			"",
			fmt.Errorf(
				"create human support actor: %w",
				err,
			)
	}

	return actorID,
		actorCode,
		nil
}

func nextHumanActorNumber(
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
								actor_code
								FROM 7
							)::integer
						),
						0
					) + 1
				FROM support_actors
				WHERE actor_code ~ '^SUP-H-[0-9]{6}$'
			`,
		).Scan(
			&number,
		); err != nil {
		return 0,
			fmt.Errorf(
				"generate human support actor code: %w",
				err,
			)
	}

	return number, nil
}

func assignQueues(
	ctx context.Context,
	tx pgx.Tx,
	actorID string,
) error {
	for _, queue := range queues {
		tag, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO support_queue_members (
						queue_id,
						support_actor_id,
						membership_role,
						created_at
					)
					SELECT
						q.id,
						$1::uuid,
						'lead',
						now()
					FROM support_queues q
					WHERE q.code = $2
					ON CONFLICT (
						queue_id,
						support_actor_id
					)
					DO UPDATE SET
						membership_role = EXCLUDED.membership_role
				`,
				actorID,
				queue.Code,
			)
		if err != nil {
			return fmt.Errorf(
				"assign queue %s: %w",
				queue.Code,
				err,
			)
		}

		if tag.RowsAffected() == 0 {
			return fmt.Errorf(
				"support queue %s does not exist",
				queue.Code,
			)
		}
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

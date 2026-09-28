package adminauth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Begin(
	ctx context.Context,
) (
	pgx.Tx,
	error,
) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin Admin auth transaction: %w",
				err,
			)
	}

	return tx,
		nil
}

func (r *Repository) GetMFAAccount(
	ctx context.Context,
	staffAccountID string,
) (
	MFAAccount,
	error,
) {
	var result MFAAccount

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					sa.id::text,
					sa.staff_code,
					sa.email,
					sa.status,
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
					)
				FROM staff_accounts sa
				WHERE sa.id = $1::uuid
			`,
			staffAccountID,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.Email,
			&result.Status,
			&result.HasPanelAccess,
		)
	if err != nil {
		return MFAAccount{},
			err
	}

	return result,
		nil
}

func (r *Repository) NewUUIDTx(
	ctx context.Context,
	tx pgx.Tx,
) (
	string,
	error,
) {
	var id string

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT gen_random_uuid()::text
			`,
		).Scan(
			&id,
		); err != nil {

		return "",
			fmt.Errorf(
				"generate Admin auth UUID: %w",
				err,
			)
	}

	return id,
		nil
}

func (r *Repository) LockTOTPCredentialTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
) (
	TOTPCredential,
	error,
) {
	var result TOTPCredential

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					staff_account_id::text,
					label,
					secret_ciphertext,
					encryption_key_id,
					algorithm,
					digits,
					period_seconds,
					status,
					last_accepted_step,
					verified_at,
					last_used_at,
					disabled_at,
					created_at,
					updated_at
				FROM admin_totp_credentials
				WHERE staff_account_id = $1::uuid
				FOR UPDATE
			`,
			staffAccountID,
		).Scan(
			&result.ID,
			&result.StaffAccountID,
			&result.Label,
			&result.SecretCiphertext,
			&result.EncryptionKeyID,
			&result.Algorithm,
			&result.Digits,
			&result.PeriodSeconds,
			&result.Status,
			&result.LastAcceptedStep,
			&result.VerifiedAt,
			&result.LastUsedAt,
			&result.DisabledAt,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return TOTPCredential{},
			err
	}

	return result,
		nil
}

func (r *Repository) SavePendingTOTPCredentialTx(
	ctx context.Context,
	tx pgx.Tx,
	credential TOTPCredential,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO admin_totp_credentials (
					id,
					staff_account_id,
					label,
					secret_ciphertext,
					encryption_key_id,
					algorithm,
					digits,
					period_seconds,
					status,
					last_accepted_step,
					verified_at,
					last_used_at,
					disabled_at,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					$4,
					$5,
					$6,
					$7,
					$8,
					'pending',
					NULL,
					NULL,
					NULL,
					NULL,
					now(),
					now()
				)
				ON CONFLICT (staff_account_id)
				DO UPDATE SET
					label = EXCLUDED.label,
					secret_ciphertext = EXCLUDED.secret_ciphertext,
					encryption_key_id = EXCLUDED.encryption_key_id,
					algorithm = EXCLUDED.algorithm,
					digits = EXCLUDED.digits,
					period_seconds = EXCLUDED.period_seconds,
					status = 'pending',
					last_accepted_step = NULL,
					verified_at = NULL,
					last_used_at = NULL,
					disabled_at = NULL,
					updated_at = now()
			`,
			credential.ID,
			credential.StaffAccountID,
			credential.Label,
			credential.SecretCiphertext,
			credential.EncryptionKeyID,
			credential.Algorithm,
			credential.Digits,
			credential.PeriodSeconds,
		)
	if err != nil {
		return fmt.Errorf(
			"save pending Admin TOTP credential: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ActivateTOTPCredentialTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	acceptedStep int64,
	verifiedAt time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_totp_credentials
				SET
					status = 'active',
					last_accepted_step = $2,
					verified_at = $3,
					last_used_at = $3,
					disabled_at = NULL,
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND status = 'pending'
			`,
			staffAccountID,
			acceptedStep,
			verifiedAt,
		)
	if err != nil {
		return fmt.Errorf(
			"activate Admin TOTP credential: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrMFAEnrollmentNotPending
	}

	return nil
}

func (r *Repository) UpdateAcceptedTOTPStepTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	acceptedStep int64,
	usedAt time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_totp_credentials
				SET
					last_accepted_step = $2,
					last_used_at = $3,
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND status = 'active'
					AND (
						last_accepted_step IS NULL
						OR last_accepted_step < $2
					)
			`,
			staffAccountID,
			acceptedStep,
			usedAt,
		)
	if err != nil {
		return fmt.Errorf(
			"update Admin TOTP replay state: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidMFACode
	}

	return nil
}

func (r *Repository) ReplaceRecoveryCodesTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	codeHashes []string,
) error {
	if _, err :=
		tx.Exec(
			ctx,
			`
				DELETE FROM admin_mfa_recovery_codes
				WHERE staff_account_id = $1::uuid
			`,
			staffAccountID,
		); err != nil {

		return fmt.Errorf(
			"delete existing Admin recovery codes: %w",
			err,
		)
	}

	for _, codeHash := range codeHashes {

		if _, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO admin_mfa_recovery_codes (
						staff_account_id,
						code_hash,
						used_at,
						created_at
					)
					VALUES (
						$1::uuid,
						$2,
						NULL,
						now()
					)
				`,
				staffAccountID,
				codeHash,
			); err != nil {

			return fmt.Errorf(
				"insert Admin recovery code: %w",
				err,
			)
		}
	}

	return nil
}

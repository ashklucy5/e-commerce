package delivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const otpChallengeSelect = `
	SELECT
		id::text,
		shipment_id::text,
		purpose,
		channel,
		recipient,
		status,
		expires_at,
		attempt_count,
		max_attempts,
		verified_at,
		consumed_at,
		created_at,
		updated_at
	FROM delivery_otp_challenges
`

func scanOTPChallenge(
	row rowScanner,
) (OTPChallenge, error) {
	var item OTPChallenge

	var verifiedAt pgtype.Timestamptz
	var consumedAt pgtype.Timestamptz

	err :=
		row.Scan(
			&item.ID,
			&item.ShipmentID,
			&item.Purpose,
			&item.Channel,
			&item.Recipient,
			&item.Status,
			&item.ExpiresAt,
			&item.AttemptCount,
			&item.MaxAttempts,
			&verifiedAt,
			&consumedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)

	if err != nil {
		return OTPChallenge{},
			err
	}

	if verifiedAt.Valid {
		value :=
			verifiedAt.Time

		item.VerifiedAt =
			&value
	}

	if consumedAt.Valid {
		value :=
			consumedAt.Time

		item.ConsumedAt =
			&value
	}

	return item, nil
}

func (r *Repository) CreateOTPChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	input IssueOTPInput,
	codeHash string,
	expiresAt time.Time,
	maxAttempts int,
) (string, error) {
	var id string

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO delivery_otp_challenges (
					shipment_id,
					purpose,
					channel,
					recipient,
					code_hash,
					status,
					expires_at,
					attempt_count,
					max_attempts,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					$4,
					$5,
					'pending',
					$6::timestamptz,
					0,
					$7,
					now(),
					now()
				)
				RETURNING id::text
			`,
			input.ShipmentID,
			input.Purpose,
			input.Channel,
			input.Recipient,
			codeHash,
			expiresAt,
			maxAttempts,
		).Scan(
			&id,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"create delivery OTP challenge: %w",
				err,
			)
	}

	return id, nil
}

func (r *Repository) CancelActiveOTPChallengesTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	purpose string,
	channel string,
	recipient string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					status = 'cancelled',
					updated_at = now()

				WHERE
					shipment_id = $1::uuid
					AND purpose = $2
					AND channel = $3
					AND recipient = $4
					AND status IN (
						'pending',
						'verified'
					)
			`,
			shipmentID,
			purpose,
			channel,
			recipient,
		)

	if err != nil {
		return fmt.Errorf(
			"cancel active delivery OTP challenges: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CancelOTPChallenge(
	ctx context.Context,
	challengeID string,
) error {
	_, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					status = 'cancelled',
					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			challengeID,
		)

	if err != nil {
		return fmt.Errorf(
			"cancel delivery OTP challenge: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) GetOTPChallenge(
	ctx context.Context,
	challengeID string,
) (OTPChallenge, error) {
	item, err :=
		scanOTPChallenge(
			r.db.QueryRow(
				ctx,
				otpChallengeSelect+
					` WHERE id = $1::uuid`,
				challengeID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return OTPChallenge{},
			ErrOTPChallengeNotFound
	}

	if err != nil {
		return OTPChallenge{},
			fmt.Errorf(
				"load delivery OTP challenge: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) LockOTPChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
) (OTPChallenge, error) {
	item, err :=
		scanOTPChallenge(
			tx.QueryRow(
				ctx,
				otpChallengeSelect+
					` WHERE id = $1::uuid FOR UPDATE`,
				challengeID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return OTPChallenge{},
			ErrOTPChallengeNotFound
	}

	if err != nil {
		return OTPChallenge{},
			fmt.Errorf(
				"lock delivery OTP challenge: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) LockOTPChallengeWithHashTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
) (OTPChallenge, string, error) {
	var item OTPChallenge

	var verifiedAt pgtype.Timestamptz
	var consumedAt pgtype.Timestamptz

	var codeHash string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					shipment_id::text,
					purpose,
					channel,
					recipient,
					status,
					expires_at,
					attempt_count,
					max_attempts,
					verified_at,
					consumed_at,
					created_at,
					updated_at,
					code_hash

				FROM delivery_otp_challenges

				WHERE id = $1::uuid

				FOR UPDATE
			`,
			challengeID,
		).Scan(
			&item.ID,
			&item.ShipmentID,
			&item.Purpose,
			&item.Channel,
			&item.Recipient,
			&item.Status,
			&item.ExpiresAt,
			&item.AttemptCount,
			&item.MaxAttempts,
			&verifiedAt,
			&consumedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&codeHash,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return OTPChallenge{},
			"",
			ErrOTPChallengeNotFound
	}

	if err != nil {
		return OTPChallenge{},
			"",
			fmt.Errorf(
				"lock delivery OTP challenge with hash: %w",
				err,
			)
	}

	if verifiedAt.Valid {
		value :=
			verifiedAt.Time

		item.VerifiedAt =
			&value
	}

	if consumedAt.Valid {
		value :=
			consumedAt.Time

		item.ConsumedAt =
			&value
	}

	return item,
		codeHash,
		nil
}

func (r *Repository) RegisterOTPFailureTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
) (bool, error) {
	var status string

	err :=
		tx.QueryRow(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					attempt_count =
						attempt_count + 1,

					status =
						CASE
							WHEN
								attempt_count + 1 >=
								max_attempts
							THEN
								'locked'
							ELSE
								status
						END,

					updated_at =
						now()

				WHERE
					id = $1::uuid
					AND status = 'pending'

				RETURNING status
			`,
			challengeID,
		).Scan(
			&status,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return false,
			ErrOTPChallengeNotFound
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"register delivery OTP failure: %w",
				err,
			)
	}

	return status ==
			OTPStatusLocked,
		nil
}

func (r *Repository) MarkOTPVerifiedTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
	now time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					status = 'verified',

					verified_at =
						COALESCE(
							verified_at,
							$2::timestamptz
						),

					updated_at =
						now()

				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			challengeID,
			now,
		)

	if err != nil {
		return fmt.Errorf(
			"mark delivery OTP verified: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOTPChallengeNotFound
	}

	return nil
}

func (r *Repository) MarkOTPExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					status = 'expired',
					updated_at = now()

				WHERE
					id = $1::uuid
					AND status IN (
						'pending',
						'verified'
					)
			`,
			challengeID,
		)

	if err != nil {
		return fmt.Errorf(
			"mark delivery OTP expired: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOTPChallengeNotFound
	}

	return nil
}

func (r *Repository) ConsumeOTPChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
	now time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					status = 'consumed',

					consumed_at =
						COALESCE(
							consumed_at,
							$2::timestamptz
						),

					updated_at =
						now()

				WHERE
					id = $1::uuid
					AND status = 'verified'
			`,
			challengeID,
			now,
		)

	if err != nil {
		return fmt.Errorf(
			"consume delivery OTP challenge: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOTPChallengeNotVerified
	}

	return nil
}

package delivery

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

func TestOTPProofPipelineIntegration(
	t *testing.T,
) {
	if os.Getenv(
		"RUN_INTEGRATION_TESTS",
	) != "1" {
		t.Skip(
			"set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests",
		)
	}

	shipmentID :=
		strings.TrimSpace(
			os.Getenv(
				"DELIVERY_OTP_TEST_SHIPMENT_ID",
			),
		)

	if shipmentID == "" {
		t.Skip(
			"set DELIVERY_OTP_TEST_SHIPMENT_ID to an existing shipment",
		)
	}

	recipient :=
		strings.TrimSpace(
			os.Getenv(
				"DELIVERY_OTP_TEST_RECIPIENT",
			),
		)

	if recipient == "" {
		recipient =
			"development-runtime-recipient"
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
	defer cancel()

	cfg, err :=
		config.Load()
	if err != nil {
		t.Fatalf(
			"load integration config: %v",
			err,
		)
	}

	pool, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"open integration postgres: %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			pool.Close()
		},
	)

	repository :=
		NewRepository(
			pool,
		)

	service :=
		NewService(
			repository,
		)

	service.SetOTPConfig(
		OTPConfig{
			TTL: 5 * time.Minute,

			MaxAttempts: 3,

			CodeDigits: 6,

			// Integration tests do not need the production bcrypt
			// cost. The stored value is still a real bcrypt hash.
			BcryptCost: bcrypt.MinCost,
		},
	)

	var captured OTPMessage

	service.SetOTPSender(
		OTPSenderFunc(
			func(
				ctx context.Context,
				message OTPMessage,
			) error {
				_ = ctx

				captured =
					message

				return nil
			},
		),
	)

	// ---------------------------------------------------------
	// Issue
	// ---------------------------------------------------------

	challenge, err :=
		service.IssueOTP(
			ctx,
			IssueOTPInput{
				ShipmentID: shipmentID,

				Purpose: PurposeDeliveryConfirmation,

				Channel: OTPChannelSMS,

				Recipient: recipient,
			},
		)
	if err != nil {
		t.Fatalf(
			"issue delivery OTP: %v",
			err,
		)
	}

	if challenge.ID == "" {
		t.Fatal(
			"issued OTP challenge has no id",
		)
	}

	challengeID :=
		challenge.ID

	t.Cleanup(
		func() {
			cleanupCtx, cleanupCancel :=
				context.WithTimeout(
					context.Background(),
					10*time.Second,
				)
			defer cleanupCancel()

			_, _ =
				pool.Exec(
					cleanupCtx,
					`
						DELETE FROM delivery_proofs
						WHERE otp_challenge_id = $1::uuid
					`,
					challengeID,
				)

			_, _ =
				pool.Exec(
					cleanupCtx,
					`
						DELETE FROM delivery_otp_challenges
						WHERE id = $1::uuid
					`,
					challengeID,
				)
		},
	)

	if challenge.Status !=
		OTPStatusPending {
		t.Fatalf(
			"expected challenge status %q, got %q",
			OTPStatusPending,
			challenge.Status,
		)
	}

	if captured.ChallengeID !=
		challenge.ID {
		t.Fatalf(
			"sender received challenge %q, expected %q",
			captured.ChallengeID,
			challenge.ID,
		)
	}

	if captured.ShipmentID !=
		shipmentID {
		t.Fatalf(
			"sender received shipment %q, expected %q",
			captured.ShipmentID,
			shipmentID,
		)
	}

	if captured.Code == "" ||
		len(
			captured.Code,
		) != 6 ||
		!allDigits(
			captured.Code,
		) {
		t.Fatalf(
			"sender received invalid OTP code %q",
			captured.Code,
		)
	}

	if captured.ExpiresAt.IsZero() {
		t.Fatal(
			"sender received no OTP expiry",
		)
	}

	// ---------------------------------------------------------
	// Prove plaintext OTP is NOT stored
	// ---------------------------------------------------------

	var codeHash string
	var databaseStatus string
	var attemptCount int
	var maxAttempts int

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT
					code_hash,
					status,
					attempt_count,
					max_attempts
				FROM delivery_otp_challenges
				WHERE id = $1::uuid
			`,
			challenge.ID,
		).Scan(
			&codeHash,
			&databaseStatus,
			&attemptCount,
			&maxAttempts,
		)
	if err != nil {
		t.Fatalf(
			"load persisted OTP challenge: %v",
			err,
		)
	}

	if codeHash ==
		captured.Code {
		t.Fatal(
			"plaintext OTP was stored in the database",
		)
	}

	if err :=
		bcrypt.CompareHashAndPassword(
			[]byte(
				codeHash,
			),
			[]byte(
				captured.Code,
			),
		); err != nil {
		t.Fatalf(
			"stored OTP hash does not match generated code: %v",
			err,
		)
	}

	if databaseStatus !=
		OTPStatusPending {
		t.Fatalf(
			"expected persisted status %q, got %q",
			OTPStatusPending,
			databaseStatus,
		)
	}

	if attemptCount != 0 {
		t.Fatalf(
			"expected zero initial attempts, got %d",
			attemptCount,
		)
	}

	if maxAttempts != 3 {
		t.Fatalf(
			"expected max attempts 3, got %d",
			maxAttempts,
		)
	}

	// ---------------------------------------------------------
	// Wrong OTP increments attempt counter
	// ---------------------------------------------------------

	wrongCode :=
		"000000"

	if wrongCode ==
		captured.Code {
		wrongCode =
			"111111"
	}

	_, err =
		service.VerifyOTP(
			ctx,
			VerifyOTPInput{
				ChallengeID: challenge.ID,

				Code: wrongCode,

				Source: "admin",

				ActorID: "integration-test",
			},
		)

	if !errors.Is(
		err,
		ErrOTPInvalidCode,
	) {
		t.Fatalf(
			"expected invalid OTP error, got %v",
			err,
		)
	}

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT
					status,
					attempt_count
				FROM delivery_otp_challenges
				WHERE id = $1::uuid
			`,
			challenge.ID,
		).Scan(
			&databaseStatus,
			&attemptCount,
		)
	if err != nil {
		t.Fatalf(
			"load OTP after failed verification: %v",
			err,
		)
	}

	if databaseStatus !=
		OTPStatusPending {
		t.Fatalf(
			"wrong OTP unexpectedly changed status to %q",
			databaseStatus,
		)
	}

	if attemptCount != 1 {
		t.Fatalf(
			"expected one failed attempt, got %d",
			attemptCount,
		)
	}

	// ---------------------------------------------------------
	// Correct OTP verifies challenge
	// ---------------------------------------------------------

	verified, err :=
		service.VerifyOTP(
			ctx,
			VerifyOTPInput{
				ChallengeID: challenge.ID,

				Code: captured.Code,

				Source: "admin",

				ActorID: "integration-test",
			},
		)
	if err != nil {
		t.Fatalf(
			"verify correct delivery OTP: %v",
			err,
		)
	}

	if verified.Status !=
		OTPStatusVerified {
		t.Fatalf(
			"expected verified status, got %q",
			verified.Status,
		)
	}

	if verified.VerifiedAt == nil {
		t.Fatal(
			"verified OTP has no verified_at timestamp",
		)
	}

	// ---------------------------------------------------------
	// OTP verification must automatically create a proof
	// ---------------------------------------------------------

	var proofID string
	var proofShipmentID string
	var proofChallengeID string
	var proofPurpose string
	var proofType string
	var proofSource string
	var proofActorID string
	var proofVerificationStatus string

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					shipment_id::text,
					otp_challenge_id::text,
					purpose,
					proof_type,
					source,
					COALESCE(actor_id, ''),
					verification_status

				FROM delivery_proofs

				WHERE otp_challenge_id = $1::uuid
			`,
			challenge.ID,
		).Scan(
			&proofID,
			&proofShipmentID,
			&proofChallengeID,
			&proofPurpose,
			&proofType,
			&proofSource,
			&proofActorID,
			&proofVerificationStatus,
		)
	if err != nil {
		t.Fatalf(
			"load OTP delivery proof: %v",
			err,
		)
	}

	if proofID == "" {
		t.Fatal(
			"OTP verification created no proof id",
		)
	}

	if proofShipmentID !=
		shipmentID {
		t.Fatalf(
			"proof shipment %q, expected %q",
			proofShipmentID,
			shipmentID,
		)
	}

	if proofChallengeID !=
		challenge.ID {
		t.Fatalf(
			"proof challenge %q, expected %q",
			proofChallengeID,
			challenge.ID,
		)
	}

	if proofPurpose !=
		PurposeDeliveryConfirmation {
		t.Fatalf(
			"unexpected proof purpose %q",
			proofPurpose,
		)
	}

	if proofType !=
		ProofTypeOTP {
		t.Fatalf(
			"expected proof type %q, got %q",
			ProofTypeOTP,
			proofType,
		)
	}

	if proofSource !=
		"admin" {
		t.Fatalf(
			"expected proof source admin, got %q",
			proofSource,
		)
	}

	if proofActorID !=
		"integration-test" {
		t.Fatalf(
			"unexpected proof actor %q",
			proofActorID,
		)
	}

	if proofVerificationStatus !=
		ProofVerificationVerified {
		t.Fatalf(
			"expected proof verification %q, got %q",
			ProofVerificationVerified,
			proofVerificationStatus,
		)
	}

	// ---------------------------------------------------------
	// Consume verified OTP
	// ---------------------------------------------------------

	consumed, err :=
		service.ConsumeOTP(
			ctx,
			ConsumeOTPInput{
				ChallengeID: challenge.ID,

				ShipmentID: shipmentID,

				Purpose: PurposeDeliveryConfirmation,
			},
		)
	if err != nil {
		t.Fatalf(
			"consume verified delivery OTP: %v",
			err,
		)
	}

	if consumed.Status !=
		OTPStatusConsumed {
		t.Fatalf(
			"expected consumed status, got %q",
			consumed.Status,
		)
	}

	if consumed.ConsumedAt == nil {
		t.Fatal(
			"consumed OTP has no consumed_at timestamp",
		)
	}

	// Consumption is intentionally idempotent.
	consumedAgain, err :=
		service.ConsumeOTP(
			ctx,
			ConsumeOTPInput{
				ChallengeID: challenge.ID,

				ShipmentID: shipmentID,

				Purpose: PurposeDeliveryConfirmation,
			},
		)
	if err != nil {
		t.Fatalf(
			"repeat delivery OTP consumption: %v",
			err,
		)
	}

	if consumedAgain.Status !=
		OTPStatusConsumed {
		t.Fatalf(
			"repeat consumption changed status to %q",
			consumedAgain.Status,
		)
	}

	// ---------------------------------------------------------
	// Final persistence proof
	// ---------------------------------------------------------

	var consumedAt time.Time

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT
					status,
					attempt_count,
					consumed_at
				FROM delivery_otp_challenges
				WHERE id = $1::uuid
			`,
			challenge.ID,
		).Scan(
			&databaseStatus,
			&attemptCount,
			&consumedAt,
		)
	if err != nil {
		t.Fatalf(
			"load final OTP persistence state: %v",
			err,
		)
	}

	if databaseStatus !=
		OTPStatusConsumed {
		t.Fatalf(
			"expected final status consumed, got %q",
			databaseStatus,
		)
	}

	if attemptCount != 1 {
		t.Fatalf(
			"expected failed-attempt count to remain 1, got %d",
			attemptCount,
		)
	}

	if consumedAt.IsZero() {
		t.Fatal(
			"final persisted consumed_at is empty",
		)
	}
}

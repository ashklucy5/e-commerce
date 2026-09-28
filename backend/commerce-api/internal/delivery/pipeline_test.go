package delivery

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFixedETAEstimator(
	t *testing.T,
) {
	base :=
		time.Date(
			2026,
			8,
			20,
			10,
			0,
			0,
			0,
			time.UTC,
		)

	estimator :=
		NewFixedETAEstimator(
			24*time.Hour,
			48*time.Hour,
			"manual",
		)

	result, err :=
		estimator.EstimateETA(
			context.Background(),
			ETAEstimateInput{
				DeliveryMode: DeliveryModeCourier,

				DispatchedAt: base,
			},
		)
	if err != nil {
		t.Fatalf(
			"estimate ETA: %v",
			err,
		)
	}

	if !result.EarliestAt.Equal(
		base.Add(
			24 * time.Hour,
		),
	) {
		t.Fatalf(
			"unexpected earliest ETA: %v",
			result.EarliestAt,
		)
	}

	if !result.LatestAt.Equal(
		base.Add(
			48 * time.Hour,
		),
	) {
		t.Fatalf(
			"unexpected latest ETA: %v",
			result.LatestAt,
		)
	}
}

func TestFixedDeliveryPricer(
	t *testing.T,
) {
	pricer :=
		NewFixedDeliveryPricer(
			80,
			65,
			"BDT",
			"manual",
		)

	pricer.TTL =
		15 * time.Minute

	result, err :=
		pricer.QuoteDelivery(
			context.Background(),
			DeliveryQuoteInput{
				DeliveryMode: DeliveryModeCourier,

				Currency: "BDT",
			},
		)
	if err != nil {
		t.Fatalf(
			"quote delivery: %v",
			err,
		)
	}

	if result.CustomerChargeAmount !=
		80 ||
		result.ProviderCostAmount !=
			65 {
		t.Fatalf(
			"unexpected quote: %+v",
			result,
		)
	}

	if result.ExpiresAt == nil {
		t.Fatal(
			"expected expiring fixed delivery quote",
		)
	}
}

func TestGenerateOTPCode(
	t *testing.T,
) {
	code, err :=
		generateOTPCode(
			6,
		)
	if err != nil {
		t.Fatalf(
			"generate OTP: %v",
			err,
		)
	}

	if len(
		code,
	) != 6 {
		t.Fatalf(
			"expected six digits, got %q",
			code,
		)
	}

	if !allDigits(
		code,
	) {
		t.Fatalf(
			"expected numeric OTP, got %q",
			code,
		)
	}
}

func TestNormalizeOTPConfig(
	t *testing.T,
) {
	config :=
		normalizeOTPConfig(
			OTPConfig{},
		)

	if config.TTL !=
		5*time.Minute {
		t.Fatalf(
			"unexpected default OTP TTL: %v",
			config.TTL,
		)
	}

	if config.MaxAttempts !=
		5 {
		t.Fatalf(
			"unexpected default OTP attempts: %d",
			config.MaxAttempts,
		)
	}

	if config.CodeDigits !=
		6 {
		t.Fatalf(
			"unexpected default OTP digits: %d",
			config.CodeDigits,
		)
	}
}

func TestOTPSenderFunc(
	t *testing.T,
) {
	var captured OTPMessage

	sender :=
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
		)

	err :=
		sender.SendOTP(
			context.Background(),
			OTPMessage{
				ChallengeID: "challenge",

				Code: "123456",
			},
		)
	if err != nil {
		t.Fatalf(
			"send OTP: %v",
			err,
		)
	}

	if captured.Code !=
		"123456" {
		t.Fatalf(
			"unexpected captured OTP: %+v",
			captured,
		)
	}
}

func TestValidateProofInput(
	t *testing.T,
) {
	shipmentID :=
		"11111111-1111-1111-1111-111111111111"

	challengeID :=
		"22222222-2222-2222-2222-222222222222"

	valid :=
		CreateProofInput{
			ShipmentID: shipmentID,

			OTPChallengeID: challengeID,

			Purpose: PurposeDeliveryConfirmation,

			ProofType: ProofTypeOTP,

			Source: "customer",

			VerificationStatus: ProofVerificationVerified,
		}

	if err :=
		validateProofInput(
			valid,
		); err != nil {
		t.Fatalf(
			"expected valid OTP proof: %v",
			err,
		)
	}

	invalid :=
		valid

	invalid.OTPChallengeID =
		""

	if err :=
		validateProofInput(
			invalid,
		); err == nil {
		t.Fatal(
			"expected OTP proof without challenge to fail",
		)
	}

	photo :=
		CreateProofInput{
			ShipmentID: shipmentID,

			Purpose: PurposeDeliveryConfirmation,

			ProofType: ProofTypePhoto,

			Source: "rider",

			StorageKey: strings.Repeat(
				"a",
				10,
			),

			VerificationStatus: ProofVerificationRecorded,
		}

	if err :=
		validateProofInput(
			photo,
		); err != nil {
		t.Fatalf(
			"expected valid photo proof: %v",
			err,
		)
	}
}

package delivery

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	PurposeDeliveryConfirmation = "delivery_confirmation"
	PurposeSelfPickup           = "self_pickup"
	PurposeRiderPickup          = "rider_pickup"
	PurposeRiderDelivery        = "rider_delivery"
)

const (
	ProofTypeOTP                 = "otp"
	ProofTypePhoto               = "photo"
	ProofTypeSignature           = "signature"
	ProofTypeProviderReference   = "provider_reference"
	ProofTypeReceiptConfirmation = "receipt_confirmation"
	ProofTypePickupConfirmation  = "pickup_confirmation"
	ProofTypeStaffConfirmation   = "staff_confirmation"
)

const (
	ProofVerificationRecorded = "recorded"
	ProofVerificationPending  = "pending"
	ProofVerificationVerified = "verified"
	ProofVerificationRejected = "rejected"
)

type CreateProofInput struct {
	ShipmentID string `json:"shipment_id"`

	OTPChallengeID string `json:"otp_challenge_id,omitempty"`

	Purpose   string `json:"purpose"`
	ProofType string `json:"proof_type"`
	Source    string `json:"source"`

	ActorID string `json:"actor_id,omitempty"`

	StorageKey        string `json:"storage_key,omitempty"`
	ExternalReference string `json:"external_reference,omitempty"`

	VerificationStatus string `json:"verification_status,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`

	OccurredAt *time.Time `json:"occurred_at,omitempty"`
}

type Proof struct {
	ID string `json:"id"`

	ShipmentID string `json:"shipment_id"`

	OTPChallengeID string `json:"otp_challenge_id,omitempty"`

	Purpose   string `json:"purpose"`
	ProofType string `json:"proof_type"`
	Source    string `json:"source"`

	ActorID string `json:"actor_id,omitempty"`

	StorageKey        string `json:"storage_key,omitempty"`
	ExternalReference string `json:"external_reference,omitempty"`

	VerificationStatus string `json:"verification_status"`

	Metadata map[string]any `json:"metadata,omitempty"`

	OccurredAt time.Time `json:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func validProofPurpose(
	value string,
) bool {
	switch value {
	case PurposeDeliveryConfirmation,
		PurposeSelfPickup,
		PurposeRiderPickup,
		PurposeRiderDelivery:
		return true

	default:
		return false
	}
}

func validProofType(
	value string,
) bool {
	switch value {
	case ProofTypeOTP,
		ProofTypePhoto,
		ProofTypeSignature,
		ProofTypeProviderReference,
		ProofTypeReceiptConfirmation,
		ProofTypePickupConfirmation,
		ProofTypeStaffConfirmation:
		return true

	default:
		return false
	}
}

func validProofVerificationStatus(
	value string,
) bool {
	switch value {
	case ProofVerificationRecorded,
		ProofVerificationPending,
		ProofVerificationVerified,
		ProofVerificationRejected:
		return true

	default:
		return false
	}
}

func normalizeProofInput(
	input *CreateProofInput,
) {
	input.ShipmentID =
		strings.TrimSpace(
			input.ShipmentID,
		)

	input.OTPChallengeID =
		strings.TrimSpace(
			input.OTPChallengeID,
		)

	input.Purpose =
		strings.ToLower(
			strings.TrimSpace(
				input.Purpose,
			),
		)

	input.ProofType =
		strings.ToLower(
			strings.TrimSpace(
				input.ProofType,
			),
		)

	input.Source =
		strings.ToLower(
			strings.TrimSpace(
				input.Source,
			),
		)

	input.ActorID =
		strings.TrimSpace(
			input.ActorID,
		)

	input.StorageKey =
		strings.TrimSpace(
			input.StorageKey,
		)

	input.ExternalReference =
		strings.TrimSpace(
			input.ExternalReference,
		)

	input.VerificationStatus =
		strings.ToLower(
			strings.TrimSpace(
				input.VerificationStatus,
			),
		)

	if input.VerificationStatus == "" {
		input.VerificationStatus =
			ProofVerificationRecorded
	}
}

func validateProofInput(
	input CreateProofInput,
) error {
	if !uuidPattern.MatchString(
		input.ShipmentID,
	) ||
		!validProofPurpose(
			input.Purpose,
		) ||
		!validProofType(
			input.ProofType,
		) ||
		!validTrackingSource(
			input.Source,
		) ||
		!validProofVerificationStatus(
			input.VerificationStatus,
		) ||
		utf8.RuneCountInString(
			input.ActorID,
		) > 160 ||
		utf8.RuneCountInString(
			input.StorageKey,
		) > 1000 ||
		utf8.RuneCountInString(
			input.ExternalReference,
		) > 160 {
		return ErrInvalidInput
	}

	if input.OTPChallengeID != "" &&
		!uuidPattern.MatchString(
			input.OTPChallengeID,
		) {
		return ErrInvalidInput
	}

	switch input.ProofType {
	case ProofTypeOTP:
		if input.OTPChallengeID == "" ||
			input.VerificationStatus !=
				ProofVerificationVerified {
			return ErrInvalidInput
		}

	case ProofTypePhoto,
		ProofTypeSignature:

		if input.StorageKey == "" {
			return ErrInvalidInput
		}

	case ProofTypeProviderReference:
		if input.ExternalReference == "" {
			return ErrInvalidInput
		}

	case ProofTypeStaffConfirmation:
		if input.ActorID == "" ||
			(input.Source != "admin" &&
				input.Source != "support") {
			return ErrInvalidInput
		}
	}

	return nil
}

func (s *Service) RecordProof(
	ctx context.Context,
	input CreateProofInput,
) (Proof, error) {
	normalizeProofInput(
		&input,
	)

	if err :=
		validateProofInput(
			input,
		); err != nil {
		return Proof{}, err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Proof{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			input.ShipmentID,
		); err != nil {
		return Proof{}, err
	}

	if input.ProofType ==
		ProofTypeOTP {

		challenge, err :=
			s.repository.LockOTPChallengeTx(
				ctx,
				tx,
				input.OTPChallengeID,
			)
		if err != nil {
			return Proof{}, err
		}

		if challenge.ShipmentID !=
			input.ShipmentID ||
			challenge.Purpose !=
				input.Purpose ||
			(challenge.Status !=
				OTPStatusVerified &&
				challenge.Status !=
					OTPStatusConsumed) {
			return Proof{},
				ErrInvalidInput
		}
	}

	id, err :=
		s.repository.InsertProofTx(
			ctx,
			tx,
			input,
		)
	if err != nil {
		return Proof{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Proof{},
			fmt.Errorf(
				"commit delivery proof: %w",
				err,
			)
	}

	return s.repository.GetProof(
		ctx,
		id,
	)
}

func (s *Service) ListProofs(
	ctx context.Context,
	shipmentID string,
) ([]Proof, error) {
	shipmentID =
		strings.TrimSpace(
			shipmentID,
		)

	if !uuidPattern.MatchString(
		shipmentID,
	) {
		return nil,
			ErrInvalidInput
	}

	if _, err :=
		s.repository.GetShipment(
			ctx,
			shipmentID,
		); err != nil {
		return nil, err
	}

	return s.repository.ListProofs(
		ctx,
		shipmentID,
	)
}

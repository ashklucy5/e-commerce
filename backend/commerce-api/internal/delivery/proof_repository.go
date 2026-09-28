package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) InsertProofTx(
	ctx context.Context,
	tx pgx.Tx,
	input CreateProofInput,
) (string, error) {
	metadataJSON := ""

	if len(
		input.Metadata,
	) > 0 {
		encoded, err :=
			json.Marshal(
				input.Metadata,
			)
		if err != nil {
			return "",
				fmt.Errorf(
					"encode delivery proof metadata: %w",
					err,
				)
		}

		metadataJSON =
			string(
				encoded,
			)
	}

	occurredAt :=
		time.Now().UTC()

	if input.OccurredAt != nil &&
		!input.OccurredAt.IsZero() {
		occurredAt =
			input.OccurredAt.UTC()
	}

	var id string

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO delivery_proofs (
					shipment_id,
					otp_challenge_id,
					purpose,
					proof_type,
					source,
					actor_id,
					storage_key,
					external_reference,
					verification_status,
					metadata,
					occurred_at,
					created_at
				)
				VALUES (
					$1::uuid,
					NULLIF($2::text, '')::uuid,
					$3,
					$4,
					$5,
					NULLIF($6, ''),
					NULLIF($7, ''),
					NULLIF($8, ''),
					$9,
					NULLIF($10, '')::jsonb,
					$11::timestamptz,
					now()
				)
				RETURNING id::text
			`,
			input.ShipmentID,
			input.OTPChallengeID,
			input.Purpose,
			input.ProofType,
			input.Source,
			input.ActorID,
			input.StorageKey,
			input.ExternalReference,
			input.VerificationStatus,
			metadataJSON,
			occurredAt,
		).Scan(
			&id,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"insert delivery proof: %w",
				err,
			)
	}

	return id, nil
}

func (r *Repository) GetProof(
	ctx context.Context,
	id string,
) (Proof, error) {
	var item Proof
	var metadataJSON []byte

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					shipment_id::text,
					COALESCE(
						otp_challenge_id::text,
						''
					),
					purpose,
					proof_type,
					source,
					COALESCE(actor_id, ''),
					COALESCE(storage_key, ''),
					COALESCE(external_reference, ''),
					verification_status,
					metadata,
					occurred_at,
					created_at

				FROM delivery_proofs

				WHERE id = $1::uuid
			`,
			id,
		).Scan(
			&item.ID,
			&item.ShipmentID,
			&item.OTPChallengeID,
			&item.Purpose,
			&item.ProofType,
			&item.Source,
			&item.ActorID,
			&item.StorageKey,
			&item.ExternalReference,
			&item.VerificationStatus,
			&metadataJSON,
			&item.OccurredAt,
			&item.CreatedAt,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Proof{},
			ErrProofNotFound
	}

	if err != nil {
		return Proof{},
			fmt.Errorf(
				"load delivery proof: %w",
				err,
			)
	}

	if len(
		metadataJSON,
	) > 0 {
		if err :=
			json.Unmarshal(
				metadataJSON,
				&item.Metadata,
			); err != nil {
			return Proof{},
				fmt.Errorf(
					"decode delivery proof metadata: %w",
					err,
				)
		}
	}

	return item, nil
}

func (r *Repository) ListProofs(
	ctx context.Context,
	shipmentID string,
) ([]Proof, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					shipment_id::text,
					COALESCE(
						otp_challenge_id::text,
						''
					),
					purpose,
					proof_type,
					source,
					COALESCE(actor_id, ''),
					COALESCE(storage_key, ''),
					COALESCE(external_reference, ''),
					verification_status,
					metadata,
					occurred_at,
					created_at

				FROM delivery_proofs

				WHERE shipment_id = $1::uuid

				ORDER BY
					occurred_at ASC,
					created_at ASC,
					id ASC
			`,
			shipmentID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list delivery proofs: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]Proof,
			0,
		)

	for rows.Next() {
		var item Proof
		var metadataJSON []byte

		if err :=
			rows.Scan(
				&item.ID,
				&item.ShipmentID,
				&item.OTPChallengeID,
				&item.Purpose,
				&item.ProofType,
				&item.Source,
				&item.ActorID,
				&item.StorageKey,
				&item.ExternalReference,
				&item.VerificationStatus,
				&metadataJSON,
				&item.OccurredAt,
				&item.CreatedAt,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan delivery proof: %w",
					err,
				)
		}

		if len(
			metadataJSON,
		) > 0 {
			if err :=
				json.Unmarshal(
					metadataJSON,
					&item.Metadata,
				); err != nil {
				return nil,
					fmt.Errorf(
						"decode delivery proof metadata: %w",
						err,
					)
			}
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate delivery proofs: %w",
				err,
			)
	}

	return items, nil
}

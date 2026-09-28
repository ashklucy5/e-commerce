package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Service) Adjust(
	ctx context.Context,
	variantID string,
	request AdjustRequest,
) (StockItem, error) {
	variantID =
		strings.TrimSpace(
			variantID,
		)

	normalizeAdjustRequest(
		&request,
	)

	if err :=
		validateAdjustment(
			variantID,
			request.QuantityDelta,
			request.Reason,
			request.ActorType,
			request.ActorID,
		); err != nil {
		return StockItem{}, err
	}

	if err :=
		s.repository.AdjustTx(
			ctx,
			nil,
			AdjustReferenceInput{
				VariantID: variantID,

				QuantityDelta: request.QuantityDelta,

				Reason: request.Reason,

				Note: request.Note,

				ActorType: request.ActorType,

				ActorID: request.ActorID,
			},
		); err != nil {
		return StockItem{}, err
	}

	return s.repository.Get(
		ctx,
		variantID,
	)
}

// AdjustReferenceTx changes on-hand inventory inside a transaction owned
// by the caller.
//
// Returns use this so return inspection, restock bookkeeping, and inventory
// movement creation commit atomically.
func (s *Service) AdjustReferenceTx(
	ctx context.Context,
	tx pgx.Tx,
	input AdjustReferenceInput,
) error {
	if tx == nil {
		return ErrInvalidInput
	}

	normalizeAdjustReferenceInput(
		&input,
	)

	if err :=
		validateAdjustment(
			input.VariantID,
			input.QuantityDelta,
			input.Reason,
			input.ActorType,
			input.ActorID,
		); err != nil {
		return err
	}

	if input.ReferenceType == "" ||
		input.ReferenceID == "" {
		return ErrInvalidInput
	}

	return s.repository.AdjustTx(
		ctx,
		tx,
		input,
	)
}

func normalizeAdjustRequest(
	request *AdjustRequest,
) {
	request.Reason =
		strings.TrimSpace(
			request.Reason,
		)

	request.Note =
		strings.TrimSpace(
			request.Note,
		)

	request.ActorType =
		strings.TrimSpace(
			request.ActorType,
		)

	request.ActorID =
		strings.TrimSpace(
			request.ActorID,
		)
}

func normalizeAdjustReferenceInput(
	input *AdjustReferenceInput,
) {
	input.VariantID =
		strings.TrimSpace(
			input.VariantID,
		)

	input.ReferenceType =
		strings.TrimSpace(
			input.ReferenceType,
		)

	input.ReferenceID =
		strings.TrimSpace(
			input.ReferenceID,
		)

	input.Reason =
		strings.TrimSpace(
			input.Reason,
		)

	input.Note =
		strings.TrimSpace(
			input.Note,
		)

	input.ActorType =
		strings.TrimSpace(
			input.ActorType,
		)

	input.ActorID =
		strings.TrimSpace(
			input.ActorID,
		)
}

func validateAdjustment(
	variantID string,
	delta int,
	reason string,
	actorType string,
	actorID string,
) error {
	if variantID == "" ||
		delta == 0 ||
		reason == "" {
		return ErrInvalidInput
	}

	if !validActorPair(
		actorType,
		actorID,
	) {
		return ErrInvalidInput
	}

	return nil
}

func (r *Repository) AdjustTx(
	ctx context.Context,
	existingTx pgx.Tx,
	input AdjustReferenceInput,
) error {
	tx := existingTx

	ownsTransaction :=
		tx == nil

	if ownsTransaction {
		var err error

		tx, err =
			r.db.Begin(
				ctx,
			)
		if err != nil {
			return fmt.Errorf(
				"begin inventory adjustment: %w",
				err,
			)
		}

		defer func() {
			_ = tx.Rollback(ctx)
		}()
	}

	if err :=
		ensureInventoryRow(
			ctx,
			tx,
			input.VariantID,
		); err != nil {
		return err
	}

	balance, err :=
		lockInventory(
			ctx,
			tx,
			input.VariantID,
		)
	if err != nil {
		return err
	}

	if input.QuantityDelta > 0 &&
		input.QuantityDelta >
			maxPostgresInteger-balance.QuantityOnHand {
		return ErrInventoryInvariant
	}

	newOnHand :=
		balance.QuantityOnHand +
			input.QuantityDelta

	if newOnHand < 0 ||
		newOnHand <
			balance.QuantityReserved {
		return ErrInsufficientStock
	}

	const updateQuery = `
		UPDATE inventory
		SET
			quantity_on_hand = $2,
			updated_at = now()
		WHERE variant_id = $1::uuid
	`

	if _, err :=
		tx.Exec(
			ctx,
			updateQuery,
			input.VariantID,
			newOnHand,
		); err != nil {
		return fmt.Errorf(
			"update inventory quantity: %w",
			err,
		)
	}

	if err :=
		insertMovement(
			ctx,
			tx,
			movementInsert{
				VariantID: input.VariantID,

				MovementType: "adjustment",

				QuantityOnHandDelta: input.QuantityDelta,

				QuantityReservedDelta: 0,

				QuantityOnHandAfter: newOnHand,

				QuantityReservedAfter: balance.QuantityReserved,

				ReferenceType: input.ReferenceType,

				ReferenceID: input.ReferenceID,

				Reason: input.Reason,

				Note: input.Note,

				ActorType: input.ActorType,

				ActorID: input.ActorID,
			},
		); err != nil {
		return err
	}

	if ownsTransaction {
		if err :=
			tx.Commit(
				ctx,
			); err != nil {
			return fmt.Errorf(
				"commit inventory adjustment: %w",
				err,
			)
		}
	}

	return nil
}

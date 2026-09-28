package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) MarkExhausted(
	ctx context.Context,
	now time.Time,
	staleBefore time.Time,
) (
	int64,
	error,
) {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE notification_outbox

				SET
					status = 'dead',
					locked_at = NULL,
					processed_at = $1,
					last_error = COALESCE(
						NULLIF(last_error, ''),
						'notification delivery attempts exhausted'
					),
					skip_reason = NULL,
					updated_at = $1

				WHERE
					attempt_count >= max_attempts

					AND (
						(
							status = 'pending'
							AND available_at <= $1
						)
						OR
						(
							status = 'processing'
							AND locked_at IS NOT NULL
							AND locked_at <= $2
						)
					)
			`,
			now,
			staleBefore,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"mark exhausted notification outbox items: %w",
				err,
			)
	}

	return tag.RowsAffected(),
		nil
}

func (r *Repository) ClaimDue(
	ctx context.Context,
	now time.Time,
	staleBefore time.Time,
	limit int,
) (
	[]OutboxItem,
	error,
) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				WITH candidates AS (
					SELECT id

					FROM notification_outbox

					WHERE
						attempt_count < max_attempts

						AND (
							(
								status = 'pending'
								AND available_at <= $1
							)
							OR
							(
								status = 'processing'
								AND locked_at IS NOT NULL
								AND locked_at <= $2
							)
						)

					ORDER BY
						available_at,
						created_at,
						id

					FOR UPDATE SKIP LOCKED

					LIMIT $3
				)

				UPDATE notification_outbox

				SET
					status = 'processing',
					attempt_count = attempt_count + 1,
					locked_at = $1,
					processed_at = NULL,
					last_error = NULL,
					skip_reason = NULL,
					updated_at = $1

				WHERE
					id IN (
						SELECT id
						FROM candidates
					)

				RETURNING
			`+
				outboxColumns,
			now,
			staleBefore,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"claim notification outbox items: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]OutboxItem,
			0,
			limit,
		)

	for rows.Next() {
		item, err :=
			scanOutboxItem(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan claimed notification outbox item: %w",
					err,
				)
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
				"iterate claimed notification outbox items: %w",
				err,
			)
	}

	return items,
		nil
}

func (r *Repository) PreferenceEnabled(
	ctx context.Context,
	item OutboxItem,
) (
	bool,
	error,
) {
	if item.CustomerID == "" {
		return defaultNotificationPreference(
				item.Category,
				item.Channel,
			),
			nil
	}

	var orderSMS bool
	var orderEmail bool

	var deliverySMS bool
	var deliveryEmail bool

	var supportSMS bool
	var supportEmail bool

	var promotionsSMS bool
	var promotionsEmail bool

	var recommendationsEmail bool

	var pushEnabled bool

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					order_updates_sms,
					order_updates_email,

					delivery_updates_sms,
					delivery_updates_email,

					support_updates_sms,
					support_updates_email,

					promotions_sms,
					promotions_email,

					recommendations_email,

					push_enabled

				FROM customer_notification_preferences

				WHERE
					customer_id = $1::uuid
			`,
			item.CustomerID,
		).Scan(
			&orderSMS,
			&orderEmail,
			&deliverySMS,
			&deliveryEmail,
			&supportSMS,
			&supportEmail,
			&promotionsSMS,
			&promotionsEmail,
			&recommendationsEmail,
			&pushEnabled,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return defaultNotificationPreference(
				item.Category,
				item.Channel,
			),
			nil
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"load customer notification preferences: %w",
				err,
			)
	}

	switch item.Category {
	case CategoryOrder,
		CategoryPayment:

		switch item.Channel {
		case ChannelSMS:
			return orderSMS, nil

		case ChannelEmail:
			return orderEmail, nil

		case ChannelPush:
			return pushEnabled, nil

		default:
			return false, nil
		}

	case CategoryDelivery:
		switch item.Channel {
		case ChannelSMS:
			return deliverySMS, nil

		case ChannelEmail:
			return deliveryEmail, nil

		case ChannelPush:
			return pushEnabled, nil

		default:
			return false, nil
		}

	case CategorySupport,
		CategorySourcing:
		switch item.Channel {
		case ChannelSMS:
			return supportSMS, nil

		case ChannelEmail:
			return supportEmail, nil

		case ChannelPush:
			return pushEnabled, nil

		default:
			return false, nil
		}

	case CategoryPromotion:
		switch item.Channel {
		case ChannelSMS:
			return promotionsSMS, nil

		case ChannelEmail:
			return promotionsEmail, nil

		default:
			return false, nil
		}

	case CategoryRecommendation:
		if item.Channel ==
			ChannelEmail {

			return recommendationsEmail,
				nil
		}

		return false,
			nil

	case CategorySecurity,
		CategorySystem:

		switch item.Channel {
		case ChannelSMS,
			ChannelEmail:

			return true,
				nil

		case ChannelPush:
			return pushEnabled,
				nil

		default:
			return false,
				nil
		}

	default:
		return false,
			nil
	}
}

func (r *Repository) MarkSent(
	ctx context.Context,
	id string,
	providerMessageID string,
	now time.Time,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE notification_outbox

				SET
					status = 'sent',
					locked_at = NULL,
					processed_at = $2,
					provider_message_id =
						NULLIF($3, ''),
					last_error = NULL,
					skip_reason = NULL,
					updated_at = $2

				WHERE
					id = $1::uuid
					AND status = 'processing'
			`,
			id,
			now,
			providerMessageID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark notification sent: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOutboxNotFound
	}

	return nil
}

func (r *Repository) MarkSkipped(
	ctx context.Context,
	id string,
	reason string,
	now time.Time,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE notification_outbox

				SET
					status = 'skipped',
					locked_at = NULL,
					processed_at = $2,
					provider_message_id = NULL,
					last_error = NULL,
					skip_reason = $3,
					updated_at = $2

				WHERE
					id = $1::uuid
					AND status = 'processing'
			`,
			id,
			now,
			reason,
		)
	if err != nil {
		return fmt.Errorf(
			"mark notification skipped: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOutboxNotFound
	}

	return nil
}

func (r *Repository) MarkRetry(
	ctx context.Context,
	id string,
	lastError string,
	availableAt time.Time,
	now time.Time,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE notification_outbox

				SET
					status = 'pending',
					locked_at = NULL,
					processed_at = NULL,
					provider_message_id = NULL,
					last_error = $3,
					skip_reason = NULL,
					available_at = $4,
					updated_at = $2

				WHERE
					id = $1::uuid
					AND status = 'processing'
			`,
			id,
			now,
			lastError,
			availableAt,
		)
	if err != nil {
		return fmt.Errorf(
			"schedule notification retry: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOutboxNotFound
	}

	return nil
}

func (r *Repository) MarkDead(
	ctx context.Context,
	id string,
	lastError string,
	now time.Time,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE notification_outbox

				SET
					status = 'dead',
					locked_at = NULL,
					processed_at = $2,
					provider_message_id = NULL,
					last_error = $3,
					skip_reason = NULL,
					updated_at = $2

				WHERE
					id = $1::uuid
					AND status = 'processing'
			`,
			id,
			now,
			lastError,
		)
	if err != nil {
		return fmt.Errorf(
			"mark notification dead: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOutboxNotFound
	}

	return nil
}

func defaultNotificationPreference(
	category string,
	channel string,
) bool {
	switch category {
	case CategoryOrder,
		CategoryPayment,
		CategoryDelivery,
		CategorySourcing,
		CategorySupport,
		CategorySecurity,
		CategorySystem:

		return channel ==
			ChannelSMS ||
			channel ==
				ChannelEmail

	case CategoryPromotion,
		CategoryRecommendation:

		return false

	default:
		return false
	}
}

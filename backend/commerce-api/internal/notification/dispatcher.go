package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultDispatchBatchSize = 100

	maxDispatchBatchSize = 1000

	defaultDispatchLockTimeout = 5 * time.Minute

	maxStoredNotificationErrorLength = 1000
)

type DispatchBatchResult struct {
	Scanned int `json:"scanned"`

	Sent int `json:"sent"`

	Skipped int `json:"skipped"`

	Retried int `json:"retried"`

	Dead int `json:"dead"`

	RecoveredExhausted int `json:"recovered_exhausted"`
}

type dispatchStore interface {
	MarkExhausted(
		ctx context.Context,
		now time.Time,
		staleBefore time.Time,
	) (
		int64,
		error,
	)

	ClaimDue(
		ctx context.Context,
		now time.Time,
		staleBefore time.Time,
		limit int,
	) (
		[]OutboxItem,
		error,
	)

	PreferenceEnabled(
		ctx context.Context,
		item OutboxItem,
	) (
		bool,
		error,
	)

	MarkSent(
		ctx context.Context,
		id string,
		providerMessageID string,
		now time.Time,
	) error

	MarkSkipped(
		ctx context.Context,
		id string,
		reason string,
		now time.Time,
	) error

	MarkRetry(
		ctx context.Context,
		id string,
		lastError string,
		availableAt time.Time,
		now time.Time,
	) error

	MarkDead(
		ctx context.Context,
		id string,
		lastError string,
		now time.Time,
	) error
}

type Dispatcher struct {
	repository dispatchStore

	channels *ChannelRegistry

	templates *TemplateRegistry

	lockTimeout time.Duration
}

func NewDispatcher(
	repository *Repository,
	channels *ChannelRegistry,
	templates *TemplateRegistry,
) *Dispatcher {
	return &Dispatcher{
		repository: repository,

		channels: channels,

		templates: templates,

		lockTimeout: defaultDispatchLockTimeout,
	}
}

func (d *Dispatcher) DispatchDue(
	ctx context.Context,
	now time.Time,
	limit int,
) (
	DispatchBatchResult,
	error,
) {
	if d == nil ||
		d.repository == nil ||
		d.channels == nil ||
		d.templates == nil {

		return DispatchBatchResult{},
			fmt.Errorf(
				"notification dispatcher is not configured",
			)
	}

	if now.IsZero() {
		now =
			time.Now().
				UTC()
	} else {
		now =
			now.UTC()
	}

	limit =
		normalizeDispatchBatchSize(
			limit,
		)

	staleBefore :=
		now.Add(
			-d.lockTimeout,
		)

	recovered, err :=
		d.repository.MarkExhausted(
			ctx,
			now,
			staleBefore,
		)
	if err != nil {
		return DispatchBatchResult{},
			err
	}

	items, err :=
		d.repository.ClaimDue(
			ctx,
			now,
			staleBefore,
			limit,
		)
	if err != nil {
		return DispatchBatchResult{},
			err
	}

	result :=
		DispatchBatchResult{
			Scanned: len(items),

			RecoveredExhausted: int(recovered),
		}

	for _, item := range items {

		if err :=
			ctx.Err(); err != nil {

			return result,
				err
		}

		if err :=
			d.dispatchOne(
				ctx,
				now,
				item,
				&result,
			); err != nil {

			return result,
				err
		}
	}

	return result,
		nil
}

func (d *Dispatcher) dispatchOne(
	ctx context.Context,
	now time.Time,
	item OutboxItem,
	result *DispatchBatchResult,
) error {
	if strings.TrimSpace(
		item.Recipient,
	) == "" {

		if err :=
			d.repository.MarkSkipped(
				ctx,
				item.ID,
				"recipient_missing",
				now,
			); err != nil {

			return err
		}

		result.Skipped++

		return nil
	}

	enabled, err :=
		d.repository.PreferenceEnabled(
			ctx,
			item,
		)
	if err != nil {
		return err
	}

	if !enabled {
		if err :=
			d.repository.MarkSkipped(
				ctx,
				item.ID,
				"preference_disabled",
				now,
			); err != nil {

			return err
		}

		result.Skipped++

		return nil
	}

	sender, err :=
		d.channels.Resolve(
			item.Channel,
		)

	switch {
	case err == nil:

	case errors.Is(
		err,
		ErrChannelNotConfigured,
	):
		/*
			No fake provider.

			If SMS/email is not actually configured, explicitly
			record that fact instead of pretending delivery
			succeeded.
		*/
		if markErr :=
			d.repository.MarkSkipped(
				ctx,
				item.ID,
				"provider_unavailable",
				now,
			); markErr != nil {

			return markErr
		}

		result.Skipped++

		return nil

	case errors.Is(
		err,
		ErrChannelUnavailable,
	):
		return d.retryOrDead(
			ctx,
			now,
			item,
			err,
			result,
		)

	default:
		return d.retryOrDead(
			ctx,
			now,
			item,
			err,
			result,
		)
	}

	var payload map[string]any

	if err :=
		json.Unmarshal(
			item.Payload,
			&payload,
		); err != nil {

		return d.markPermanentFailure(
			ctx,
			now,
			item,
			fmt.Errorf(
				"invalid notification payload: %w",
				err,
			),
			result,
		)
	}

	rendered, err :=
		d.templates.Render(
			item.TemplateKey,
			payload,
		)
	if err != nil {
		return d.markPermanentFailure(
			ctx,
			now,
			item,
			err,
			result,
		)
	}

	sendResult, err :=
		sender.Send(
			ctx,
			SendRequest{
				Recipient: item.Recipient,

				Subject: rendered.Subject,

				Text: rendered.Text,

				DedupeKey: item.DedupeKey,
			},
		)
	if err != nil {
		return d.retryOrDead(
			ctx,
			now,
			item,
			err,
			result,
		)
	}

	providerMessageID :=
		strings.TrimSpace(
			sendResult.ProviderMessageID,
		)

	if len(
		providerMessageID,
	) > 255 {

		return d.markPermanentFailure(
			ctx,
			now,
			item,
			fmt.Errorf(
				"notification provider message ID exceeds 255 characters",
			),
			result,
		)
	}

	if err :=
		d.repository.MarkSent(
			ctx,
			item.ID,
			providerMessageID,
			now,
		); err != nil {

		return err
	}

	result.Sent++

	return nil
}

func (d *Dispatcher) retryOrDead(
	ctx context.Context,
	now time.Time,
	item OutboxItem,
	cause error,
	result *DispatchBatchResult,
) error {
	message :=
		safeNotificationError(
			cause,
		)

	if item.AttemptCount >=
		item.MaxAttempts {

		if err :=
			d.repository.MarkDead(
				ctx,
				item.ID,
				message,
				now,
			); err != nil {

			return err
		}

		result.Dead++

		return nil
	}

	availableAt :=
		now.Add(
			notificationRetryDelay(
				item.AttemptCount,
			),
		)

	if err :=
		d.repository.MarkRetry(
			ctx,
			item.ID,
			message,
			availableAt,
			now,
		); err != nil {

		return err
	}

	result.Retried++

	return nil
}

func (d *Dispatcher) markPermanentFailure(
	ctx context.Context,
	now time.Time,
	item OutboxItem,
	cause error,
	result *DispatchBatchResult,
) error {
	if err :=
		d.repository.MarkDead(
			ctx,
			item.ID,
			safeNotificationError(
				cause,
			),
			now,
		); err != nil {

		return err
	}

	result.Dead++

	return nil
}

func notificationRetryDelay(
	attempt int,
) time.Duration {
	if attempt <= 1 {
		return 30 *
			time.Second
	}

	delay :=
		30 *
			time.Second

	for step :=
		1; step < attempt; step++ {

		delay *= 2

		if delay >=
			15*time.Minute {

			return 15 *
				time.Minute
		}
	}

	return delay
}

func safeNotificationError(
	err error,
) string {
	if err == nil {
		return "notification delivery failed"
	}

	value :=
		strings.TrimSpace(
			err.Error(),
		)

	if value == "" {
		value =
			"notification delivery failed"
	}

	runes :=
		[]rune(
			value,
		)

	if len(runes) >
		maxStoredNotificationErrorLength {

		runes =
			runes[:maxStoredNotificationErrorLength]
	}

	return string(
		runes,
	)
}

func normalizeDispatchBatchSize(
	limit int,
) int {
	if limit <= 0 {
		return defaultDispatchBatchSize
	}

	if limit >
		maxDispatchBatchSize {

		return maxDispatchBatchSize
	}

	return limit
}

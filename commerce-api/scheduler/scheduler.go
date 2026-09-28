package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"
)

type PaymentExpirer interface {
	ExpirePendingPayments(
		ctx context.Context,
		now time.Time,
		limit int,
	) (
		int,
		error,
	)
}

type PaymentExpiryEnqueuer interface {
	EnqueuePaymentExpiry(
		ctx context.Context,
		batchSize int,
	) error
}

type PaymentReconciliationEnqueuer interface {
	Enqueue(
		ctx context.Context,
		batchSize int,
	) error
}

type Scheduler struct {
	paymentExpirer PaymentExpirer

	paymentEnqueuer PaymentExpiryEnqueuer

	paymentExpiryInterval time.Duration

	paymentExpiryBatchSize int

	paymentReconciliationEnqueuer PaymentReconciliationEnqueuer

	paymentReconciliationInterval time.Duration

	paymentReconciliationBatchSize int

	logger *log.Logger
}

func New(
	paymentExpirer PaymentExpirer,
	paymentExpiryInterval time.Duration,
	paymentExpiryBatchSize int,
	logger *log.Logger,
) *Scheduler {
	return newScheduler(
		paymentExpirer,
		nil,
		nil,
		paymentExpiryInterval,
		paymentExpiryBatchSize,
		0,
		0,
		logger,
	)
}

func NewQueue(
	paymentEnqueuer PaymentExpiryEnqueuer,
	paymentExpiryInterval time.Duration,
	paymentExpiryBatchSize int,
	logger *log.Logger,
) *Scheduler {
	return newScheduler(
		nil,
		paymentEnqueuer,
		nil,
		paymentExpiryInterval,
		paymentExpiryBatchSize,
		0,
		0,
		logger,
	)
}

func NewQueueWithReconciliation(
	paymentEnqueuer PaymentExpiryEnqueuer,
	paymentReconciliationEnqueuer PaymentReconciliationEnqueuer,
	paymentExpiryInterval time.Duration,
	paymentExpiryBatchSize int,
	paymentReconciliationInterval time.Duration,
	paymentReconciliationBatchSize int,
	logger *log.Logger,
) *Scheduler {
	return newScheduler(
		nil,
		paymentEnqueuer,
		paymentReconciliationEnqueuer,
		paymentExpiryInterval,
		paymentExpiryBatchSize,
		paymentReconciliationInterval,
		paymentReconciliationBatchSize,
		logger,
	)
}

func newScheduler(
	paymentExpirer PaymentExpirer,
	paymentEnqueuer PaymentExpiryEnqueuer,
	paymentReconciliationEnqueuer PaymentReconciliationEnqueuer,
	paymentExpiryInterval time.Duration,
	paymentExpiryBatchSize int,
	paymentReconciliationInterval time.Duration,
	paymentReconciliationBatchSize int,
	logger *log.Logger,
) *Scheduler {
	if paymentExpiryInterval <= 0 {
		paymentExpiryInterval =
			30 * time.Second
	}

	if paymentExpiryBatchSize <= 0 {
		paymentExpiryBatchSize =
			100
	}

	if paymentExpiryBatchSize > 1000 {
		paymentExpiryBatchSize =
			1000
	}

	if paymentReconciliationInterval <= 0 {
		paymentReconciliationInterval =
			30 * time.Second
	}

	if paymentReconciliationBatchSize <= 0 {
		paymentReconciliationBatchSize =
			100
	}

	if paymentReconciliationBatchSize > 1000 {
		paymentReconciliationBatchSize =
			1000
	}

	if logger == nil {
		logger =
			log.Default()
	}

	return &Scheduler{
		paymentExpirer: paymentExpirer,

		paymentEnqueuer: paymentEnqueuer,

		paymentExpiryInterval: paymentExpiryInterval,

		paymentExpiryBatchSize: paymentExpiryBatchSize,

		paymentReconciliationEnqueuer: paymentReconciliationEnqueuer,

		paymentReconciliationInterval: paymentReconciliationInterval,

		paymentReconciliationBatchSize: paymentReconciliationBatchSize,

		logger: logger,
	}
}

func (s *Scheduler) Run(
	ctx context.Context,
) {
	/*
		Run both sweeps immediately on startup rather than waiting
		for the first ticker.

		Queue deduplication makes this safe across scheduler
		replicas.
	*/
	s.runPaymentExpiry(
		ctx,
	)

	if s.paymentReconciliationEnqueuer != nil {
		s.runPaymentReconciliation(
			ctx,
		)
	}

	paymentExpiryTicker :=
		time.NewTicker(
			s.paymentExpiryInterval,
		)

	defer paymentExpiryTicker.Stop()

	var paymentReconciliationTicker *time.Ticker

	var paymentReconciliationC <-chan time.Time

	if s.paymentReconciliationEnqueuer != nil {
		paymentReconciliationTicker =
			time.NewTicker(
				s.paymentReconciliationInterval,
			)

		paymentReconciliationC =
			paymentReconciliationTicker.C

		defer paymentReconciliationTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Printf(
				"payment scheduler stopped",
			)

			return

		case <-paymentExpiryTicker.C:
			s.runPaymentExpiry(
				ctx,
			)

		case <-paymentReconciliationC:
			s.runPaymentReconciliation(
				ctx,
			)
		}
	}
}

func (s *Scheduler) RunOnce(
	ctx context.Context,
) (
	int,
	error,
) {
	if s.paymentEnqueuer != nil {
		if err :=
			s.paymentEnqueuer.
				EnqueuePaymentExpiry(
					ctx,
					s.paymentExpiryBatchSize,
				); err != nil {

			return 0,
				err
		}

		return 0,
			nil
	}

	if s.paymentExpirer == nil {
		return 0,
			fmt.Errorf(
				"payment expiry scheduler has no expirer or queue enqueuer",
			)
	}

	return s.paymentExpirer.
		ExpirePendingPayments(
			ctx,
			time.Now().
				UTC(),
			s.paymentExpiryBatchSize,
		)
}

func (s *Scheduler) RunReconciliationOnce(
	ctx context.Context,
) error {
	if s.paymentReconciliationEnqueuer == nil {
		return fmt.Errorf(
			"payment reconciliation scheduler has no queue enqueuer",
		)
	}

	return s.paymentReconciliationEnqueuer.
		Enqueue(
			ctx,
			s.paymentReconciliationBatchSize,
		)
}

func (s *Scheduler) runPaymentExpiry(
	ctx context.Context,
) {
	expired, err :=
		s.RunOnce(
			ctx,
		)
	if err != nil {
		s.logger.Printf(
			"payment expiry scheduler error: %v",
			err,
		)

		return
	}

	if s.paymentEnqueuer != nil {
		return
	}

	if expired > 0 {
		s.logger.Printf(
			"expired %d pending payment order(s)",
			expired,
		)
	}
}

func (s *Scheduler) runPaymentReconciliation(
	ctx context.Context,
) {
	if err :=
		s.RunReconciliationOnce(
			ctx,
		); err != nil {

		s.logger.Printf(
			"payment reconciliation scheduler error: %v",
			err,
		)
	}
}

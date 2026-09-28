package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Enqueue(
	ctx context.Context,
	request EnqueueRequest,
) (
	OutboxItem,
	bool,
	error,
) {
	if s == nil ||
		s.repository == nil {

		return OutboxItem{},
			false,
			fmt.Errorf(
				"notification service is not configured",
			)
	}

	payload, err :=
		normalizeAndValidateEnqueueRequest(
			&request,
		)
	if err != nil {
		return OutboxItem{},
			false,
			err
	}

	return s.repository.Enqueue(
		ctx,
		request,
		payload,
	)
}

// EnqueueTx is the important transactional-outbox entry point.
//
// Order/payment/delivery/support code should call this method using
// its existing PostgreSQL transaction so the domain change and
// notification-outbox row either commit together or roll back
// together.
func (s *Service) EnqueueTx(
	ctx context.Context,
	tx pgx.Tx,
	request EnqueueRequest,
) (
	OutboxItem,
	bool,
	error,
) {
	if s == nil ||
		s.repository == nil ||
		tx == nil {

		return OutboxItem{},
			false,
			fmt.Errorf(
				"notification transactional writer is not configured",
			)
	}

	payload, err :=
		normalizeAndValidateEnqueueRequest(
			&request,
		)
	if err != nil {
		return OutboxItem{},
			false,
			err
	}

	return s.repository.EnqueueTx(
		ctx,
		tx,
		request,
		payload,
	)
}

func normalizeAndValidateEnqueueRequest(
	request *EnqueueRequest,
) (
	[]byte,
	error,
) {
	if request == nil {
		return nil,
			ErrInvalidInput
	}

	request.DedupeKey =
		strings.TrimSpace(
			request.DedupeKey,
		)

	request.Category =
		strings.ToLower(
			strings.TrimSpace(
				request.Category,
			),
		)

	request.EventType =
		strings.ToLower(
			strings.TrimSpace(
				request.EventType,
			),
		)

	request.Channel =
		strings.ToLower(
			strings.TrimSpace(
				request.Channel,
			),
		)

	request.CustomerID =
		strings.TrimSpace(
			request.CustomerID,
		)

	request.OrderID =
		strings.TrimSpace(
			request.OrderID,
		)

	request.CaseID =
		strings.TrimSpace(
			request.CaseID,
		)

	request.Recipient =
		strings.TrimSpace(
			request.Recipient,
		)

	request.TemplateKey =
		strings.ToLower(
			strings.TrimSpace(
				request.TemplateKey,
			),
		)

	if request.MaxAttempts <= 0 {
		request.MaxAttempts = 5
	}

	if request.AvailableAt.IsZero() {
		request.AvailableAt =
			time.Now().
				UTC()
	} else {
		request.AvailableAt =
			request.AvailableAt.
				UTC()
	}

	if request.Payload == nil {
		request.Payload =
			map[string]any{}
	}

	if request.DedupeKey == "" ||
		len(request.DedupeKey) > 200 ||
		!validCategory(
			request.Category,
		) ||
		request.EventType == "" ||
		len(request.EventType) > 100 ||
		!validChannel(
			request.Channel,
		) ||
		!validOptionalUUID(
			request.CustomerID,
		) ||
		!validOptionalUUID(
			request.OrderID,
		) ||
		!validOptionalUUID(
			request.CaseID,
		) ||
		len(request.Recipient) > 255 ||
		request.TemplateKey == "" ||
		len(request.TemplateKey) > 120 ||
		request.MaxAttempts < 1 ||
		request.MaxAttempts > 20 {

		return nil,
			ErrInvalidInput
	}

	if containsSensitivePayloadKey(
		request.Payload,
	) {
		return nil,
			ErrInvalidInput
	}

	payload, err :=
		json.Marshal(
			request.Payload,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"encode notification payload: %w",
				err,
			)
	}

	if len(payload) >
		maxOutboxPayloadBytes {

		return nil,
			ErrInvalidInput
	}

	return payload,
		nil
}

func containsSensitivePayloadKey(
	value any,
) bool {
	switch typed :=
		value.(type) {

	case map[string]any:
		for key, nested := range typed {

			normalized :=
				strings.ToLower(
					strings.TrimSpace(
						key,
					),
				)

			switch normalized {
			case "password",
				"access_token",
				"refresh_token",
				"token",
				"secret",
				"api_key",
				"authorization",
				"cookie",
				"otp",
				"otp_code",
				"totp",
				"totp_secret":

				return true
			}

			if containsSensitivePayloadKey(
				nested,
			) {
				return true
			}
		}

	case []any:
		for _, nested := range typed {

			if containsSensitivePayloadKey(
				nested,
			) {
				return true
			}
		}
	}

	return false
}

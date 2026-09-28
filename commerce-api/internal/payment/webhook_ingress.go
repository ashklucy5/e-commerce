package payment

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	providerpayments "project.local/commerce-api/integrations/payments"
)

const maxPaymentWebhookBodyBytes int64 = 1 << 20

var ErrInvalidVerifiedWebhookEvent = errors.New(
	"invalid verified payment webhook event",
)

type verifiedPaymentConfirmer interface {
	ConfirmVerifiedPayment(
		context.Context,
		ConfirmVerifiedPaymentInput,
	) (ConfirmVerifiedPaymentResult, error)
}

type WebhookService struct {
	providers *providerpayments.Registry
	confirmer verifiedPaymentConfirmer
}

type WebhookProcessResult struct {
	Acknowledged bool `json:"acknowledged"`
	Processed    bool `json:"processed"`

	Replayed bool `json:"replayed,omitempty"`
	Ignored  bool `json:"ignored,omitempty"`

	ReconciliationRequired bool `json:"reconciliation_required,omitempty"`

	Provider string `json:"provider"`
	Status   string `json:"status"`

	PaymentID string `json:"payment_id,omitempty"`
}

func NewWebhookService(
	providers *providerpayments.Registry,
	confirmer verifiedPaymentConfirmer,
) *WebhookService {
	return &WebhookService{
		providers: providers,
		confirmer: confirmer,
	}
}

func (s *WebhookService) Process(
	ctx context.Context,
	providerCode string,
	headers http.Header,
	body []byte,
) (WebhookProcessResult, error) {
	if s == nil ||
		s.providers == nil ||
		s.confirmer == nil ||
		len(body) == 0 {

		return WebhookProcessResult{},
			ErrInvalidVerifiedWebhookEvent
	}

	/*
		Important:

		Use ResolveForProcessing(), not Resolve().

		A provider can be disabled for NEW payments while an
		already-created payment still needs its webhook processed.
	*/
	provider, err :=
		s.providers.ResolveForProcessing(
			providerCode,
		)
	if err != nil {
		return WebhookProcessResult{},
			err
	}

	/*
		The adapter receives the exact raw request body.

		No JSON parsing occurs before signature/provider
		verification.
	*/
	event, err :=
		provider.VerifyWebhook(
			ctx,
			providerpayments.WebhookRequest{
				Headers: headers.Clone(),

				Body: append(
					[]byte(nil),
					body...,
				),
			},
		)
	if err != nil {
		return WebhookProcessResult{},
			err
	}

	if err :=
		normalizeVerifiedWebhookEvent(
			&event,
		); err != nil {

		return WebhookProcessResult{},
			err
	}

	canonicalProvider :=
		strings.ToLower(
			strings.TrimSpace(
				provider.Code(),
			),
		)

	result :=
		WebhookProcessResult{
			Acknowledged: true,
			Provider:     canonicalProvider,
			Status:       event.Status,
		}

	/*
		Webhook Ingress v1 mutates order/payment state only for a
		cryptographically/provider-verified successful payment.

		Pending/failed/cancelled/expired provider messages are safe
		to acknowledge but must never mark an order as paid.

		Reconciliation will handle those states later.
	*/
	if event.Status !=
		providerpayments.StatusSucceeded {

		result.Ignored = true

		return result, nil
	}

	var paidAt time.Time

	if event.PaidAt != nil {
		paidAt =
			event.PaidAt.UTC()
	}

	confirmed, err :=
		s.confirmer.ConfirmVerifiedPayment(
			ctx,
			ConfirmVerifiedPaymentInput{
				OrderID: event.OrderID,

				Provider: canonicalProvider,

				ProviderEventID: event.ProviderEventID,
				EventType:       event.EventType,

				ProviderPaymentID: event.ProviderPaymentID,

				ProviderTransactionID: event.ProviderTransactionID,

				Amount: event.Amount,

				Currency: event.Currency,

				PaidAt: paidAt,

				PayloadSHA256: PayloadSHA256(
					body,
				),
			},
		)

	result.Processed =
		confirmed.Processed

	result.PaymentID =
		confirmed.Payment.ID

	if err == nil {
		result.Replayed =
			!confirmed.Processed

		return result, nil
	}

	/*
		These are deterministic verified-payment conflicts.

		ConfirmVerifiedPayment already records the failed/
		reconciliation event transactionally.

		Return a successful webhook acknowledgement so the payment
		provider does not generate an endless retry storm.
	*/
	if isAcknowledgedPaymentConflict(
		err,
	) {
		result.ReconciliationRequired = true

		return result, nil
	}

	/*
		Infrastructure/database failures remain errors so the
		provider can retry the webhook later.
	*/
	return WebhookProcessResult{},
		err
}

func normalizeVerifiedWebhookEvent(
	event *providerpayments.VerifiedWebhookEvent,
) error {
	if event == nil {
		return ErrInvalidVerifiedWebhookEvent
	}

	event.ProviderEventID =
		strings.TrimSpace(
			event.ProviderEventID,
		)

	event.EventType =
		strings.TrimSpace(
			event.EventType,
		)

	event.OrderID =
		strings.TrimSpace(
			event.OrderID,
		)

	event.ProviderPaymentID =
		strings.TrimSpace(
			event.ProviderPaymentID,
		)

	event.ProviderTransactionID =
		strings.TrimSpace(
			event.ProviderTransactionID,
		)

	event.Status =
		strings.ToLower(
			strings.TrimSpace(
				event.Status,
			),
		)

	event.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				event.Currency,
			),
		)

	if !paymentUUIDPattern.MatchString(
		event.OrderID,
	) ||
		event.ProviderEventID == "" ||
		len(event.ProviderEventID) > 200 ||
		event.EventType == "" ||
		len(event.EventType) > 100 ||
		event.ProviderPaymentID == "" ||
		len(event.ProviderPaymentID) > 160 ||
		len(event.ProviderTransactionID) > 160 {

		return ErrInvalidVerifiedWebhookEvent
	}

	switch event.Status {
	case providerpayments.StatusPending,
		providerpayments.StatusSucceeded,
		providerpayments.StatusFailed,
		providerpayments.StatusCancelled,
		providerpayments.StatusExpired:

	default:
		return ErrInvalidVerifiedWebhookEvent
	}

	if event.Status ==
		providerpayments.StatusSucceeded {

		if event.Amount < 0 ||
			len(event.Currency) != 3 {

			return ErrInvalidVerifiedWebhookEvent
		}
	}

	if event.PaidAt != nil {
		value :=
			event.PaidAt.UTC()

		event.PaidAt =
			&value
	}

	return nil
}

func isAcknowledgedPaymentConflict(
	err error,
) bool {
	return errors.Is(
		err,
		ErrPaymentMethodMismatch,
	) ||
		errors.Is(
			err,
			ErrPaymentAmountMismatch,
		) ||
		errors.Is(
			err,
			ErrPaymentCurrencyMismatch,
		) ||
		errors.Is(
			err,
			ErrPaymentReferenceConflict,
		) ||
		errors.Is(
			err,
			ErrReconciliationRequired,
		)
}

type WebhookHandler struct {
	service *WebhookService
}

func NewWebhookHandler(
	service *WebhookService,
) *WebhookHandler {
	return &WebhookHandler{
		service: service,
	}
}

func (h *WebhookHandler) Receive(
	c *gin.Context,
) {
	c.Header(
		"Cache-Control",
		"no-store",
	)

	if h == nil ||
		h.service == nil {

		writeWebhookHTTPError(
			c,
			errors.New(
				"payment webhook service unavailable",
			),
		)

		return
	}

	/*
		Webhook endpoints are public network surfaces.

		Bound the body before reading it so a provider endpoint
		cannot be used for an unbounded-memory request.
	*/
	c.Request.Body =
		http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			maxPaymentWebhookBodyBytes,
		)

	body, err :=
		io.ReadAll(
			c.Request.Body,
		)
	if err != nil {
		var tooLarge *http.MaxBytesError

		if errors.As(
			err,
			&tooLarge,
		) {
			c.JSON(
				http.StatusRequestEntityTooLarge,
				gin.H{
					"error": gin.H{
						"code": "PAYMENT_WEBHOOK_TOO_LARGE",

						"message": "Payment webhook payload is too large",
					},
				},
			)

			return
		}

		writeWebhookHTTPError(
			c,
			err,
		)

		return
	}

	if len(
		bytes.TrimSpace(
			body,
		),
	) == 0 {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_PAYMENT_WEBHOOK",

					"message": "Invalid payment webhook",
				},
			},
		)

		return
	}

	result, err :=
		h.service.Process(
			c.Request.Context(),
			c.Param(
				"provider",
			),
			c.Request.Header,
			body,
		)
	if err != nil {
		writeWebhookHTTPError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func writeWebhookHTTPError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		providerpayments.ErrUnsupportedProvider,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_WEBHOOK_PROVIDER_UNSUPPORTED",

					"message": "Payment webhook provider is not supported",
				},
			},
		)

	case errors.Is(
		err,
		providerpayments.ErrProviderUnavailable,
	),
		errors.Is(
			err,
			providerpayments.ErrProviderNotConfigured,
		):

		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_WEBHOOK_PROVIDER_UNAVAILABLE",

					"message": "Payment webhook provider is currently unavailable",
				},
			},
		)

	case errors.Is(
		err,
		providerpayments.ErrInvalidWebhook,
	),
		errors.Is(
			err,
			ErrInvalidVerifiedWebhookEvent,
		):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_PAYMENT_WEBHOOK",

					"message": "Invalid payment webhook",
				},
			},
		)

	case errors.Is(
		err,
		providerpayments.ErrWebhookVerificationFailed,
	):
		c.JSON(
			http.StatusUnauthorized,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_WEBHOOK_VERIFICATION_FAILED",

					"message": "Payment webhook verification failed",
				},
			},
		)

	default:
		log.Printf(
			"payment webhook error: %v",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_WEBHOOK_PROCESSING_FAILED",

					"message": "Unable to process payment webhook",
				},
			},
		)
	}
}

func RegisterWebhookRoutes(
	group *gin.RouterGroup,
	handler *WebhookHandler,
) {
	group.POST(
		"/payments/webhooks/:provider",
		handler.Receive,
	)
}

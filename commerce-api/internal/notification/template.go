package notification

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	texttemplate "text/template"
)

const (
	TemplateOrderPlaced = "order.placed"

	TemplateOrderProcessing = "order.processing"

	TemplatePaymentSucceeded = "payment.succeeded"

	TemplateDeliveryUpdated = "delivery.updated"

	TemplateDeliveryDispatched = "delivery.dispatched"

	TemplateDeliveryDelivered = "delivery.delivered"

	TemplateSupportReply = "support.reply"

	TemplateInvoiceIssuedCustomer = "invoice.issued.customer"

	TemplateInvoiceIssuedBusiness = "invoice.issued.business"

	TemplateInvoiceResentCustomer = "invoice.resent.customer"

	TemplateInvoiceResentBusiness = "invoice.resent.business"
)

type TemplateDefinition struct {
	Key string

	Subject string

	Text string
}

type RenderedMessage struct {
	Subject string

	Text string
}

type compiledTemplate struct {
	subject *texttemplate.Template

	text *texttemplate.Template
}

type TemplateRegistry struct {
	mu sync.RWMutex

	templates map[string]compiledTemplate
}

func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(
			map[string]compiledTemplate,
		),
	}
}

func NewDefaultTemplateRegistry() (
	*TemplateRegistry,
	error,
) {
	registry :=
		NewTemplateRegistry()

	definitions :=
		[]TemplateDefinition{
			{
				Key: TemplateOrderPlaced,

				Subject: "Order received",

				Text: "{{.message}}",
			},
			{
				Key: TemplateOrderProcessing,

				Subject: "Order update",

				Text: "{{.message}}",
			},
			{
				Key: TemplatePaymentSucceeded,

				Subject: "Payment received",

				Text: "{{.message}}",
			},
			{
				Key: TemplateDeliveryUpdated,

				Subject: "Delivery update",

				Text: "{{.message}}",
			},
			{
				Key: TemplateDeliveryDispatched,

				Subject: "Order out for delivery",

				Text: "{{.message}}",
			},
			{
				Key: TemplateDeliveryDelivered,

				Subject: "Order delivered",

				Text: "{{.message}}",
			},
			{
				Key: TemplateSupportReply,

				Subject: "Support update",

				Text: "{{.message}}",
			},
			{
				Key: TemplateInvoiceIssuedCustomer,

				Subject: "Invoice {{.invoice_number}}",

				Text: "Your invoice {{.invoice_number}} for order {{.order_number}} is ready.",
			},
			{
				Key: TemplateInvoiceIssuedBusiness,

				Subject: "Invoice issued: {{.invoice_number}}",

				Text: "Invoice {{.invoice_number}} was issued for order {{.order_number}}.",
			},
			{
				Key: TemplateInvoiceResentCustomer,

				Subject: "Invoice copy {{.invoice_number}}",

				Text: "A copy of invoice {{.invoice_number}} for order {{.order_number}} was requested.",
			},
			{
				Key: TemplateInvoiceResentBusiness,

				Subject: "Invoice copy resent: {{.invoice_number}}",

				Text: "A copy of invoice {{.invoice_number}} was queued again for order {{.order_number}}.",
			},
		}

	for _, definition := range definitions {

		if err :=
			registry.Register(
				definition,
			); err != nil {

			return nil,
				err
		}
	}

	return registry,
		nil
}

func (r *TemplateRegistry) Register(
	definition TemplateDefinition,
) error {
	if r == nil {
		return ErrTemplateInvalid
	}

	definition.Key =
		strings.ToLower(
			strings.TrimSpace(
				definition.Key,
			),
		)

	definition.Subject =
		strings.TrimSpace(
			definition.Subject,
		)

	definition.Text =
		strings.TrimSpace(
			definition.Text,
		)

	if definition.Key == "" ||
		len(
			definition.Key,
		) > 120 ||
		definition.Text == "" {

		return ErrTemplateInvalid
	}

	subject, err :=
		texttemplate.New(
			definition.Key +
				".subject",
		).
			Option(
				"missingkey=error",
			).
			Parse(
				definition.Subject,
			)
	if err != nil {
		return fmt.Errorf(
			"%w: subject: %v",
			ErrTemplateInvalid,
			err,
		)
	}

	text, err :=
		texttemplate.New(
			definition.Key +
				".text",
		).
			Option(
				"missingkey=error",
			).
			Parse(
				definition.Text,
			)
	if err != nil {
		return fmt.Errorf(
			"%w: text: %v",
			ErrTemplateInvalid,
			err,
		)
	}

	r.mu.Lock()

	defer r.mu.Unlock()

	if _, exists :=
		r.templates[definition.Key]; exists {

		return fmt.Errorf(
			"notification template %s already registered",
			definition.Key,
		)
	}

	r.templates[definition.Key] =
		compiledTemplate{
			subject: subject,

			text: text,
		}

	return nil
}

func (r *TemplateRegistry) Render(
	key string,
	payload map[string]any,
) (
	RenderedMessage,
	error,
) {
	if r == nil {
		return RenderedMessage{},
			ErrTemplateNotFound
	}

	key =
		strings.ToLower(
			strings.TrimSpace(
				key,
			),
		)

	r.mu.RLock()

	definition,
		exists :=
		r.templates[key]

	r.mu.RUnlock()

	if !exists {
		return RenderedMessage{},
			ErrTemplateNotFound
	}

	var subject bytes.Buffer

	if err :=
		definition.subject.Execute(
			&subject,
			payload,
		); err != nil {

		return RenderedMessage{},
			fmt.Errorf(
				"render notification subject: %w",
				err,
			)
	}

	var text bytes.Buffer

	if err :=
		definition.text.Execute(
			&text,
			payload,
		); err != nil {

		return RenderedMessage{},
			fmt.Errorf(
				"render notification text: %w",
				err,
			)
	}

	return RenderedMessage{
		Subject: strings.TrimSpace(
			subject.String(),
		),

		Text: strings.TrimSpace(
			text.String(),
		),
	}, nil
}

package order

import (
	"fmt"
	"net/mail"
	"os"
	"strings"
)

type invoiceRuntimeConfig struct {
	Merchant InvoiceMerchant

	BusinessEmail string
}

func ValidateInvoiceRuntimeConfig() error {
	_, err :=
		loadInvoiceRuntimeConfig()

	return err
}

func loadInvoiceRuntimeConfig() (
	invoiceRuntimeConfig,
	error,
) {
	result :=
		invoiceRuntimeConfig{
			Merchant: InvoiceMerchant{
				Name: invoiceEnv(
					"INVOICE_MERCHANT_NAME",
				),

				Email: invoiceEnv(
					"INVOICE_MERCHANT_EMAIL",
				),

				Phone: invoiceEnv(
					"INVOICE_MERCHANT_PHONE",
				),

				AddressLine1: invoiceEnv(
					"INVOICE_MERCHANT_ADDRESS_LINE1",
				),

				AddressLine2: invoiceEnv(
					"INVOICE_MERCHANT_ADDRESS_LINE2",
				),

				City: invoiceEnv(
					"INVOICE_MERCHANT_CITY",
				),

				Country: invoiceEnv(
					"INVOICE_MERCHANT_COUNTRY",
				),
			},

			BusinessEmail: invoiceEnv(
				"INVOICE_BUSINESS_EMAIL",
			),
		}

	// A separate business mailbox is optional. If one is not
	// configured, use the merchant/customer-facing mailbox.
	if result.BusinessEmail == "" {
		result.BusinessEmail =
			result.Merchant.Email
	}

	if result.Merchant.Name == "" {
		return invoiceRuntimeConfig{},
			fmt.Errorf(
				"INVOICE_MERCHANT_NAME is required",
			)
	}

	if result.BusinessEmail == "" {
		return invoiceRuntimeConfig{},
			fmt.Errorf(
				"INVOICE_BUSINESS_EMAIL or INVOICE_MERCHANT_EMAIL is required",
			)
	}

	if result.Merchant.Email != "" {
		if err :=
			validateInvoiceEmail(
				result.Merchant.Email,
			); err != nil {

			return invoiceRuntimeConfig{},
				fmt.Errorf(
					"invalid INVOICE_MERCHANT_EMAIL: %w",
					err,
				)
		}
	}

	if err :=
		validateInvoiceEmail(
			result.BusinessEmail,
		); err != nil {

		return invoiceRuntimeConfig{},
			fmt.Errorf(
				"invalid INVOICE_BUSINESS_EMAIL: %w",
				err,
			)
	}

	return result,
		nil
}

func validateInvoiceEmail(
	value string,
) error {
	parsed, err :=
		mail.ParseAddress(
			value,
		)
	if err != nil {
		return err
	}

	if !strings.EqualFold(
		strings.TrimSpace(
			parsed.Address,
		),
		strings.TrimSpace(
			value,
		),
	) {
		return fmt.Errorf(
			"must contain only an email address",
		)
	}

	return nil
}

func invoiceEnv(
	key string,
) string {
	return strings.TrimSpace(
		os.Getenv(
			key,
		),
	)
}

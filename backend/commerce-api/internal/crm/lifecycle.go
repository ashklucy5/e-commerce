package crm

const (
	StatusWaitingSupport  = "waiting_support"
	StatusWaitingCustomer = "waiting_customer"
	StatusResolved        = "resolved"
	StatusClosed          = "closed"
)

func validCaseType(value string) bool {
	switch value {
	case "bulk_stock_request",
		"product_question",
		"order_issue",
		"payment_issue",
		"return_issue",
		"delivery_issue",
		"complaint",
		"general_question",
		"other":
		return true
	default:
		return false
	}
}

package admin

type PermissionDefinition struct {
	Code        string
	Description string
}

const (
	PermissionPanelAccess = "admin.panel.access"

	PermissionDashboardRead = "admin.dashboard.read"

	// ---------------------------------------------------------
	// Catalog / merchandising
	// ---------------------------------------------------------

	PermissionCatalogRead = "admin.catalog.read"

	PermissionCatalogWrite = "admin.catalog.write"

	PermissionCatalogImport = "admin.catalog.import"

	PermissionPromotionRead = "admin.promotion.read"

	PermissionPromotionManage = "admin.promotion.manage"

	PermissionMerchandisingRead = "admin.merchandising.read"

	PermissionMerchandisingManage = "admin.merchandising.manage"

	PermissionSearchAnalyticsRead = "admin.search.analytics.read"

	// ---------------------------------------------------------
	// Inventory / procurement
	// ---------------------------------------------------------

	PermissionInventoryRead = "admin.inventory.read"

	PermissionInventoryAdjust = "admin.inventory.adjust"

	PermissionProcurementRead = "admin.procurement.read"

	PermissionProcurementManage = "admin.procurement.manage"

	// ---------------------------------------------------------
	// Orders
	// ---------------------------------------------------------

	PermissionOrderRead = "admin.order.read"

	PermissionOrderManage = "admin.order.manage"

	// ---------------------------------------------------------
	// Invoices
	// ---------------------------------------------------------

	PermissionInvoiceRead = "admin.invoice.read"

	PermissionInvoiceIssue = "admin.invoice.issue"

	PermissionInvoiceResend = "admin.invoice.resend"

	PermissionInvoiceCreditNote = "admin.invoice.credit_note"

	PermissionInvoiceVoid = "admin.invoice.void"

	// ---------------------------------------------------------
	// Customers
	// ---------------------------------------------------------

	PermissionCustomerRead = "admin.customer.read"

	PermissionCustomerManage = "admin.customer.manage"

	PermissionCustomerBan = "admin.customer.ban"

	PermissionCustomerUnban = "admin.customer.unban"

	// ---------------------------------------------------------
	// Payments / refunds
	// ---------------------------------------------------------

	PermissionPaymentRead = "admin.payment.read"

	/*
		Keep this permission for now because the existing refund
		routes still use it.

		We will migrate those routes to the granular permissions
		below without breaking the current backend.
	*/
	PermissionPaymentRefund = "admin.payment.refund"

	PermissionRefundRead = "admin.refund.read"

	PermissionRefundCreate = "admin.refund.create"

	PermissionRefundApprove = "admin.refund.approve"

	PermissionRefundProcess = "admin.refund.process"

	// ---------------------------------------------------------
	// Returns
	// ---------------------------------------------------------

	PermissionReturnRead = "admin.return.read"

	PermissionReturnManage = "admin.return.manage"

	// ---------------------------------------------------------
	// Warehouse
	// ---------------------------------------------------------

	PermissionWarehouseRead = "admin.warehouse.read"

	PermissionWarehouseManage = "admin.warehouse.manage"

	// ---------------------------------------------------------
	// Delivery
	// ---------------------------------------------------------

	PermissionDeliveryRead = "admin.delivery.read"

	PermissionDeliveryManage = "admin.delivery.manage"

	PermissionDeliveryConfirmReceipt = "admin.delivery.confirm_receipt"

	// ---------------------------------------------------------
	// CRM / support
	// ---------------------------------------------------------

	PermissionCRMRead = "admin.crm.read"

	PermissionCRMManage = "admin.crm.manage"

	// ---------------------------------------------------------
	// Product sourcing
	// ---------------------------------------------------------

	PermissionSourcingRead = "admin.sourcing.read"

	PermissionSourcingReview = "admin.sourcing.review"

	PermissionSourcingOfferManage = "admin.sourcing.offer.manage"

	PermissionSourcingFinalize = "admin.sourcing.finalize"

	// ---------------------------------------------------------
	// Reviews
	// ---------------------------------------------------------

	PermissionReviewRead = "admin.review.read"

	PermissionReviewModerate = "admin.review.moderate"

	// ---------------------------------------------------------
	// Notifications
	// ---------------------------------------------------------

	PermissionNotificationRead = "admin.notification.read"

	PermissionNotificationManage = "admin.notification.manage"

	// ---------------------------------------------------------
	// Analytics
	// ---------------------------------------------------------

	PermissionAnalyticsRead = "admin.analytics.read"

	// ---------------------------------------------------------
	// Finance
	// ---------------------------------------------------------

	PermissionFinanceRead = "admin.finance.read"

	PermissionFinanceManage = "admin.finance.manage"

	PermissionFinanceExport = "admin.finance.export"

	// ---------------------------------------------------------
	// Staff
	// ---------------------------------------------------------

	PermissionStaffRead = "admin.staff.read"

	/*
		Manage ordinary staff lifecycle.

		This is deliberately separate from assigning roles
		and managing staff security.
	*/
	PermissionStaffManage = "admin.staff.manage"

	/*
		Allows assignment of permitted operational roles.

		The backend service layer will still decide which
		roles the current administrator is allowed to assign.
	*/
	PermissionStaffRoleAssign = "admin.staff.role.assign"

	/*
		Password recovery, session revocation and MFA recovery
		for staff accounts.

		Protected staff such as Super Admins will still require
		Super Admin authority at the service layer.
	*/
	PermissionStaffSecurityManage = "admin.staff.security.manage"

	/*
		"Delete" means permanent loss of access while retaining
		the historical identity/audit record.
	*/
	PermissionStaffDelete = "admin.staff.delete"

	PermissionStaffBan = "admin.staff.ban"

	// ---------------------------------------------------------
	// Roles
	// ---------------------------------------------------------

	PermissionRoleRead = "admin.role.read"

	/*
		Modify role definitions themselves.

		Normal Admin staff will NOT receive this permission.
	*/
	PermissionRoleManage = "admin.role.manage"

	// ---------------------------------------------------------
	// Security
	// ---------------------------------------------------------

	PermissionSecurityRead = "admin.security.read"

	PermissionSecurityManage = "admin.security.manage"

	// ---------------------------------------------------------
	// Audit
	// ---------------------------------------------------------

	PermissionAuditRead = "admin.audit.read"

	PermissionAuditExport = "admin.audit.export"

	// ---------------------------------------------------------
	// Settings
	// ---------------------------------------------------------

	PermissionSettingsRead = "admin.settings.read"

	PermissionSettingsManage = "admin.settings.manage"

	// ---------------------------------------------------------
	// Integrations
	// ---------------------------------------------------------

	PermissionIntegrationRead = "admin.integration.read"

	PermissionIntegrationManage = "admin.integration.manage"

	// ---------------------------------------------------------
	// System operations
	// ---------------------------------------------------------

	PermissionSystemOperationsRead = "admin.system.operations.read"

	PermissionSystemOperationsManage = "admin.system.operations.manage"
)

var Permissions = []PermissionDefinition{
	{
		Code:        PermissionPanelAccess,
		Description: "Access the Admin Panel",
	},
	{
		Code:        PermissionDashboardRead,
		Description: "View Admin dashboard summaries",
	},

	// ---------------------------------------------------------
	// Catalog / merchandising
	// ---------------------------------------------------------

	{
		Code:        PermissionCatalogRead,
		Description: "View products, variants, categories, and catalog data",
	},
	{
		Code:        PermissionCatalogWrite,
		Description: "Create and modify products, variants, categories, and catalog data",
	},
	{
		Code:        PermissionCatalogImport,
		Description: "Upload, validate, preview, and apply catalog imports",
	},
	{
		Code:        PermissionPromotionRead,
		Description: "View promotions, coupons, and campaign configuration",
	},
	{
		Code:        PermissionPromotionManage,
		Description: "Create and manage promotions, coupons, and campaign rules",
	},
	{
		Code:        PermissionMerchandisingRead,
		Description: "View merchandising configuration such as featured products and collections",
	},
	{
		Code:        PermissionMerchandisingManage,
		Description: "Manage merchandising configuration such as featured products and collections",
	},
	{
		Code:        PermissionSearchAnalyticsRead,
		Description: "View search terms, zero-result searches, and search performance analytics",
	},

	// ---------------------------------------------------------
	// Inventory / procurement
	// ---------------------------------------------------------

	{
		Code:        PermissionInventoryRead,
		Description: "View inventory and stock availability",
	},
	{
		Code:        PermissionInventoryAdjust,
		Description: "Perform authorized inventory adjustments",
	},
	{
		Code:        PermissionProcurementRead,
		Description: "View procurement and incoming-stock workflows",
	},
	{
		Code:        PermissionProcurementManage,
		Description: "Manage procurement, receiving, and incoming-stock workflows",
	},

	// ---------------------------------------------------------
	// Orders
	// ---------------------------------------------------------

	{
		Code:        PermissionOrderRead,
		Description: "View customer orders",
	},
	{
		Code:        PermissionOrderManage,
		Description: "Perform authorized order operations",
	},

	// ---------------------------------------------------------
	// Invoices
	// ---------------------------------------------------------

	{
		Code:        PermissionInvoiceRead,
		Description: "View invoice records and immutable invoice snapshots",
	},
	{
		Code:        PermissionInvoiceIssue,
		Description: "Issue authorized invoices",
	},
	{
		Code:        PermissionInvoiceResend,
		Description: "Resend or redeliver issued invoices",
	},
	{
		Code:        PermissionInvoiceCreditNote,
		Description: "Create authorized invoice credit notes",
	},
	{
		Code:        PermissionInvoiceVoid,
		Description: "Void invoices where business and legal rules allow it",
	},

	// ---------------------------------------------------------
	// Customers
	// ---------------------------------------------------------

	{
		Code:        PermissionCustomerRead,
		Description: "View customer accounts and customer information",
	},
	{
		Code:        PermissionCustomerManage,
		Description: "Perform authorized customer-account operations",
	},
	{
		Code:        PermissionCustomerBan,
		Description: "Apply authorized customer account or capability bans",
	},
	{
		Code:        PermissionCustomerUnban,
		Description: "Remove authorized customer account or capability bans",
	},

	// ---------------------------------------------------------
	// Payments / refunds
	// ---------------------------------------------------------

	{
		Code:        PermissionPaymentRead,
		Description: "View payment information and payment state",
	},
	{
		Code:        PermissionPaymentRefund,
		Description: "Legacy broad refund permission retained during granular refund migration",
	},
	{
		Code:        PermissionRefundRead,
		Description: "View refund requests and refund processing state",
	},
	{
		Code:        PermissionRefundCreate,
		Description: "Create authorized refund requests",
	},
	{
		Code:        PermissionRefundApprove,
		Description: "Approve authorized refund requests",
	},
	{
		Code:        PermissionRefundProcess,
		Description: "Process approved refunds and record provider outcomes",
	},

	// ---------------------------------------------------------
	// Returns
	// ---------------------------------------------------------

	{
		Code:        PermissionReturnRead,
		Description: "View return requests and return state",
	},
	{
		Code:        PermissionReturnManage,
		Description: "Manage return workflows",
	},

	// ---------------------------------------------------------
	// Warehouse
	// ---------------------------------------------------------

	{
		Code:        PermissionWarehouseRead,
		Description: "View warehouse and fulfillment state",
	},
	{
		Code:        PermissionWarehouseManage,
		Description: "Manage warehouse fulfillment operations",
	},

	// ---------------------------------------------------------
	// Delivery
	// ---------------------------------------------------------

	{
		Code:        PermissionDeliveryRead,
		Description: "View delivery and shipment information",
	},
	{
		Code:        PermissionDeliveryManage,
		Description: "Manage shipment and delivery operations",
	},
	{
		Code:        PermissionDeliveryConfirmReceipt,
		Description: "Confirm customer receipt through authorized Admin operations",
	},

	// ---------------------------------------------------------
	// CRM / support
	// ---------------------------------------------------------

	{
		Code:        PermissionCRMRead,
		Description: "View customer-service CRM cases",
	},
	{
		Code:        PermissionCRMManage,
		Description: "Perform authorized CRM case operations",
	},

	// ---------------------------------------------------------
	// Product sourcing
	// ---------------------------------------------------------

	{
		Code:        PermissionSourcingRead,
		Description: "View customer product-sourcing requests, offers, and agreements",
	},
	{
		Code:        PermissionSourcingReview,
		Description: "Review, hold, accept, or cancel customer product-sourcing requests",
	},
	{
		Code:        PermissionSourcingOfferManage,
		Description: "Create, edit, send, and manage product-sourcing commercial offers",
	},
	{
		Code:        PermissionSourcingFinalize,
		Description: "Finalize customer-accepted product-sourcing agreements",
	},

	// ---------------------------------------------------------
	// Reviews
	// ---------------------------------------------------------

	{
		Code:        PermissionReviewRead,
		Description: "View customer reviews and review metadata",
	},
	{
		Code:        PermissionReviewModerate,
		Description: "Moderate customer reviews",
	},

	// ---------------------------------------------------------
	// Notifications
	// ---------------------------------------------------------

	{
		Code:        PermissionNotificationRead,
		Description: "View staff notification inboxes and notification delivery state",
	},
	{
		Code:        PermissionNotificationManage,
		Description: "Manage authorized notification templates, delivery rules, and operational notifications",
	},

	// ---------------------------------------------------------
	// Analytics
	// ---------------------------------------------------------

	{
		Code:        PermissionAnalyticsRead,
		Description: "View Admin analytics and operational metrics",
	},

	// ---------------------------------------------------------
	// Finance
	// ---------------------------------------------------------

	{
		Code:        PermissionFinanceRead,
		Description: "View finance expense ledger and profit/loss reporting inputs",
	},
	{
		Code:        PermissionFinanceManage,
		Description: "Create and void authorized finance expense ledger entries",
	},
	{
		Code:        PermissionFinanceExport,
		Description: "Export authorized finance and accounting reports",
	},

	// ---------------------------------------------------------
	// Staff
	// ---------------------------------------------------------

	{
		Code:        PermissionStaffRead,
		Description: "View staff accounts and authorization assignments",
	},
	{
		Code:        PermissionStaffManage,
		Description: "Create ordinary staff accounts and manage ordinary staff lifecycle state",
	},
	{
		Code:        PermissionStaffRoleAssign,
		Description: "Assign authorized operational roles to ordinary staff accounts",
	},
	{
		Code:        PermissionStaffSecurityManage,
		Description: "Perform authorized staff password, session, and MFA recovery operations",
	},
	{
		Code:        PermissionStaffDelete,
		Description: "Permanently remove staff access while retaining historical identity and audit records",
	},
	{
		Code:        PermissionStaffBan,
		Description: "Apply or remove authorized staff security bans",
	},

	// ---------------------------------------------------------
	// Roles
	// ---------------------------------------------------------

	{
		Code:        PermissionRoleRead,
		Description: "View staff roles and permissions",
	},
	{
		Code:        PermissionRoleManage,
		Description: "Create or modify non-protected role definitions and permission bundles",
	},

	// ---------------------------------------------------------
	// Security
	// ---------------------------------------------------------

	{
		Code:        PermissionSecurityRead,
		Description: "View staff security posture, sessions, MFA state, and security events",
	},
	{
		Code:        PermissionSecurityManage,
		Description: "Manage privileged staff security policy and security recovery operations",
	},

	// ---------------------------------------------------------
	// Audit
	// ---------------------------------------------------------

	{
		Code:        PermissionAuditRead,
		Description: "View Admin audit and security events",
	},
	{
		Code:        PermissionAuditExport,
		Description: "Export authorized Admin audit and security-event records",
	},

	// ---------------------------------------------------------
	// Settings
	// ---------------------------------------------------------

	{
		Code:        PermissionSettingsRead,
		Description: "View commerce and operational settings",
	},
	{
		Code:        PermissionSettingsManage,
		Description: "Modify authorized commerce and operational settings",
	},

	// ---------------------------------------------------------
	// Integrations
	// ---------------------------------------------------------

	{
		Code:        PermissionIntegrationRead,
		Description: "View configured integrations, providers, and webhook health",
	},
	{
		Code:        PermissionIntegrationManage,
		Description: "Manage authorized integrations, providers, and webhook configuration",
	},

	// ---------------------------------------------------------
	// System operations
	// ---------------------------------------------------------

	{
		Code:        PermissionSystemOperationsRead,
		Description: "View background jobs, queues, failures, retries, and system-operation health",
	},
	{
		Code:        PermissionSystemOperationsManage,
		Description: "Perform authorized retries and system-operation management actions",
	},
}

func PermissionCodes() []string {
	result :=
		make(
			[]string,
			0,
			len(Permissions),
		)

	for _, permission := range Permissions {
		result =
			append(
				result,
				permission.Code,
			)
	}

	return result
}

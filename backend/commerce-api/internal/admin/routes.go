package admin

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/analytics"
)

type PermissionMiddleware func(
	permission string,
) gin.HandlerFunc

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requirePermission PermissionMiddleware,
) {
	// ---------------------------------------------------------
	// Dashboard
	// ---------------------------------------------------------

	group.GET(
		"/dashboard",
		requirePermission(
			PermissionDashboardRead,
		),
		handler.Dashboard,
	)

	// ---------------------------------------------------------
	// Audit
	// ---------------------------------------------------------

	group.GET(
		"/audit/events",
		requirePermission(
			PermissionAuditRead,
		),
		handler.AuditEvents,
	)

	// ---------------------------------------------------------
	// Catalog
	// ---------------------------------------------------------

	group.GET(
		"/products",
		requirePermission(
			PermissionCatalogRead,
		),
		handler.CatalogProducts,
	)

	group.GET(
		"/products/:product_id",
		requirePermission(
			PermissionCatalogRead,
		),
		handler.CatalogProduct,
	)

	group.PATCH(
		"/products/:product_id",
		requirePermission(
			PermissionCatalogWrite,
		),
		handler.UpdateCatalogProduct,
	)

	// ---------------------------------------------------------
	// Promotions
	// ---------------------------------------------------------

	group.GET(
		"/promotions",
		requirePermission(
			PermissionPromotionRead,
		),
		handler.Promotions,
	)

	group.POST(
		"/promotions",
		requirePermission(
			PermissionPromotionManage,
		),
		handler.CreatePromotion,
	)

	group.GET(
		"/promotions/product-discounts",
		requirePermission(
			PermissionPromotionRead,
		),
		handler.ProductDiscounts,
	)

	group.PUT(
		"/promotions/product-discounts/:variant_id",
		requirePermission(
			PermissionPromotionManage,
		),
		handler.ApplyProductDiscount,
	)

	group.DELETE(
		"/promotions/product-discounts/:variant_id",
		requirePermission(
			PermissionPromotionManage,
		),
		handler.ClearProductDiscount,
	)

	group.GET(
		"/promotions/:promotion_id",
		requirePermission(
			PermissionPromotionRead,
		),
		handler.Promotion,
	)

	group.PUT(
		"/promotions/:promotion_id",
		requirePermission(
			PermissionPromotionManage,
		),
		handler.ReplacePromotion,
	)

	// ---------------------------------------------------------
	// Orders
	// ---------------------------------------------------------

	group.GET(
		"/orders",
		requirePermission(
			PermissionOrderRead,
		),
		handler.Orders,
	)

	group.GET(
		"/orders/:order_id",
		requirePermission(
			PermissionOrderRead,
		),
		handler.Order,
	)

	// ---------------------------------------------------------
	// Invoices
	// ---------------------------------------------------------

	group.GET(
		"/orders/:order_id/invoice",
		requirePermission(
			PermissionInvoiceRead,
		),
		handler.OrderInvoice,
	)

	group.POST(
		"/orders/:order_id/invoice/resend",
		requirePermission(
			PermissionInvoiceResend,
		),
		handler.ResendOrderInvoice,
	)

	// ---------------------------------------------------------
	// Returns
	// ---------------------------------------------------------

	group.GET(
		"/returns",
		requirePermission(
			PermissionReturnRead,
		),
		handler.Returns,
	)

	group.GET(
		"/returns/:return_id",
		requirePermission(
			PermissionReturnRead,
		),
		handler.Return,
	)

	// ---------------------------------------------------------
	// Payments
	// ---------------------------------------------------------

	group.GET(
		"/payments",
		requirePermission(
			PermissionPaymentRead,
		),
		handler.Payments,
	)

	group.GET(
		"/payments/:payment_id",
		requirePermission(
			PermissionPaymentRead,
		),
		handler.Payment,
	)

	// ---------------------------------------------------------
	// Customers
	// ---------------------------------------------------------

	group.GET(
		"/customers",
		requirePermission(
			PermissionCustomerRead,
		),
		handler.Customers,
	)

	group.GET(
		"/customers/:customer_id",
		requirePermission(
			PermissionCustomerRead,
		),
		handler.Customer,
	)

	group.PATCH(
		"/customers/:customer_id/status",
		requirePermission(
			PermissionCustomerManage,
		),
		handler.UpdateCustomerStatus,
	)

	// ---------------------------------------------------------
	// CRM / customer support
	// ---------------------------------------------------------

	group.GET(
		"/crm/queues",
		requirePermission(
			PermissionCRMRead,
		),
		handler.CRMQueues,
	)

	group.GET(
		"/crm/cases",
		requirePermission(
			PermissionCRMRead,
		),
		handler.CRMCases,
	)

	group.GET(
		"/crm/cases/:id",
		requirePermission(
			PermissionCRMRead,
		),
		handler.CRMCase,
	)

	group.GET(
		"/crm/cases/:id/messages",
		requirePermission(
			PermissionCRMRead,
		),
		handler.CRMCaseMessages,
	)

	group.POST(
		"/crm/cases/:id/claim",
		requirePermission(
			PermissionCRMManage,
		),
		handler.ClaimCRMCase,
	)

	group.POST(
		"/crm/cases/:id/messages",
		requirePermission(
			PermissionCRMManage,
		),
		handler.ReplyCRMCase,
	)

		group.POST(
		"/crm/cases/:id/resolve",
		requirePermission(
			PermissionCRMManage,
		),
		handler.ResolveCRMCase,
	)

	group.POST(
		"/crm/cases/:id/close",
		requirePermission(
			PermissionCRMManage,
		),
		handler.CloseCRMCase,
	)

	group.POST(
		"/crm/cases/:id/escalate",
		requirePermission(
			PermissionCRMManage,
		),
		handler.EscalateCRMCase,
	)

	// ---------------------------------------------------------
	// Reviews
	// ---------------------------------------------------------

	group.GET(
		"/reviews",
		requirePermission(
			PermissionReviewRead,
		),
		handler.Reviews,
	)

	group.GET(
		"/reviews/:review_id",
		requirePermission(
			PermissionReviewRead,
		),
		handler.Review,
	)

	group.PATCH(
		"/reviews/:review_id/status",
		requirePermission(
			PermissionReviewModerate,
		),
		handler.UpdateReviewStatus,
	)

	// ---------------------------------------------------------
	// Staff
	// ---------------------------------------------------------

	group.GET(
		"/staff",
		requirePermission(
			PermissionStaffRead,
		),
		handler.StaffMembers,
	)

	group.GET(
		"/staff/:staff_id",
		requirePermission(
			PermissionStaffRead,
		),
		handler.StaffMember,
	)

	/*
		Staff invitation state is part of the Staff Directory lifecycle.
		The activation token itself is never returned by the read endpoint;
		only a successful manual reissue returns a fresh token once.
	*/
	group.GET(
		"/staff/:staff_id/invitation",
		requirePermission(
			PermissionStaffRead,
		),
		handler.StaffInvitation,
	)

	group.POST(
		"/staff/:staff_id/invitation/reissue",
		requirePermission(
			PermissionStaffManage,
		),
		handler.ReissueStaffInvitation,
	)

	group.POST(
		"/staff/:staff_id/invitation/cancel",
		requirePermission(
			PermissionStaffManage,
		),
		handler.CancelStaffInvitation,
	)

	group.GET(
		"/staff/:staff_id/bans",
		requirePermission(
			PermissionStaffRead,
		),
		handler.StaffBans,
	)

	group.POST(
		"/staff/:staff_id/bans",
		requirePermission(
			PermissionStaffBan,
		),
		handler.CreateStaffBan,
	)

	group.POST(
		"/staff/:staff_id/bans/:ban_id/revoke",
		requirePermission(
			PermissionStaffBan,
		),
		handler.RevokeStaffBan,
	)

	/*
		Creating staff requires both lifecycle-management
		authority and staff-role assignment authority.
		The service layer still enforces exactly which roles
		the current actor may assign.
	*/
	group.POST(
		"/staff",
		requirePermission(
			PermissionStaffManage,
		),
		requirePermission(
			PermissionStaffRoleAssign,
		),
		handler.CreateStaffMember,
	)

	/*
		Generic status changes are deliberately limited to:
		- active
		- suspended
		- disabled

		Permanent deletion and banning are separate security
		operations.
	*/
	group.PATCH(
		"/staff/:staff_id/status",
		requirePermission(
			PermissionStaffManage,
		),
		handler.UpdateStaffStatus,
	)

	/*
		Permanent access removal.

		This does not physically remove the staff row.
		The service changes status to "deleted", revokes
		sessions, invalidates MFA/recovery codes, disables the
		linked support actor, and preserves the historical
		identity for audit/business records.
	*/
	group.DELETE(
		"/staff/:staff_id",
		requirePermission(
			PermissionStaffDelete,
		),
		handler.DeleteStaffMember,
	)

	/*
		Assigning existing roles to a staff account is
		different from modifying role definitions themselves.
		The service layer prevents privilege escalation into
		protected roles.
	*/
	group.PUT(
		"/staff/:staff_id/roles",
		requirePermission(
			PermissionStaffManage,
		),
		requirePermission(
			PermissionStaffRoleAssign,
		),
		handler.ReplaceStaffMemberRoles,
	)

	/*
		Administrative password and MFA recovery are security
		operations rather than ordinary staff management.
		Protected target accounts are additionally enforced in
		the service layer.
	*/
	group.PUT(
		"/staff/:staff_id/password",
		requirePermission(
			PermissionStaffSecurityManage,
		),
		handler.ResetStaffMemberPassword,
	)

	group.POST(
		"/staff/:staff_id/admin-mfa/reset",
		requirePermission(
			PermissionStaffSecurityManage,
		),
		handler.ResetStaffMemberAdminMFA,
	)

	// ---------------------------------------------------------
	// Roles and permissions
	// ---------------------------------------------------------

	group.GET(
		"/roles",
		requirePermission(
			PermissionRoleRead,
		),
		handler.Roles,
	)

	group.GET(
		"/roles/:role_id",
		requirePermission(
			PermissionRoleRead,
		),
		handler.Role,
	)

	group.GET(
		"/permissions",
		requirePermission(
			PermissionRoleRead,
		),
		handler.Permissions,
	)

	/*
		Role-definition mutation remains separate from staff
		role assignment.

		Normal Administrator does not receive
		PermissionRoleManage.
	*/
	group.POST(
		"/roles",
		requirePermission(
			PermissionRoleManage,
		),
		handler.CreateRole,
	)

	group.PATCH(
		"/roles/:role_id",
		requirePermission(
			PermissionRoleManage,
		),
		handler.UpdateRole,
	)

	group.PUT(
		"/roles/:role_id/permissions",
		requirePermission(
			PermissionRoleManage,
		),
		handler.ReplaceRolePermissionAssignments,
	)

	// ---------------------------------------------------------
	// Finance expense ledger
	// ---------------------------------------------------------

	group.GET(
		"/finance/expenses",
		requirePermission(
			PermissionFinanceRead,
		),
		handler.FinanceExpenses,
	)

	group.GET(
		"/finance/expenses/:expense_id",
		requirePermission(
			PermissionFinanceRead,
		),
		handler.FinanceExpense,
	)

	group.POST(
		"/finance/expenses",
		requirePermission(
			PermissionFinanceManage,
		),
		handler.CreateFinanceExpense,
	)

	group.POST(
		"/finance/expenses/:expense_id/void",
		requirePermission(
			PermissionFinanceManage,
		),
		handler.VoidFinanceExpense,
	)

	// ---------------------------------------------------------
	// Finance product / SKU buying-cost history
	// ---------------------------------------------------------

	group.GET(
		"/finance/product-costs",
		requirePermission(
			PermissionFinanceRead,
		),
		handler.FinanceVariantCosts,
	)

	group.GET(
		"/finance/product-costs/:cost_id",
		requirePermission(
			PermissionFinanceRead,
		),
		handler.FinanceVariantCost,
	)

	group.POST(
		"/finance/product-costs",
		requirePermission(
			PermissionFinanceManage,
		),
		handler.CreateFinanceVariantCost,
	)

	// ---------------------------------------------------------
	// Finance Excel import
	// ---------------------------------------------------------

	group.GET(
		"/finance/imports/template",
		requirePermission(
			PermissionFinanceRead,
		),
		handler.FinanceImportTemplate,
	)

	group.POST(
		"/finance/imports",
		requirePermission(
			PermissionFinanceManage,
		),
		handler.CreateFinanceImport,
	)

	group.GET(
		"/finance/imports/:import_id",
		requirePermission(
			PermissionFinanceRead,
		),
		handler.FinanceImport,
	)

	group.POST(
		"/finance/imports/:import_id/apply",
		requirePermission(
			PermissionFinanceManage,
		),
		handler.ApplyFinanceImport,
	)

	// ---------------------------------------------------------
	// Analytics + finance analytics
	// ---------------------------------------------------------

	analyticsHandler :=
		analytics.NewHandler(
			analytics.NewService(
				handler.service.db,
			),
		)

	analytics.RegisterFinanceRoutes(
		group,
		analyticsHandler,
		requirePermission,
		PermissionFinanceRead,
	)

	analytics.RegisterRoutes(
		group,
		analyticsHandler,
		requirePermission,
		PermissionAnalyticsRead,
	)
}

package admin

const (
	RoleSuperAdmin = "admin_superuser"

	RoleAdministrator = "admin_administrator"

	RoleSourcing = "admin_sourcing"

	RoleCatalog = "admin_catalog"

	RoleInventory = "admin_inventory"

	RoleWarehouse = "admin_warehouse"

	RoleFulfillment = "admin_fulfillment"

	RoleFinance = "admin_finance"

	RoleReturns = "admin_returns"

	RoleAnalytics = "admin_analytics"

	RoleSecurity = "admin_security"
)

type RoleDefinition struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

var SuperAdminRole = RoleDefinition{
	Code: RoleSuperAdmin,

	Name: "Super Admin",

	Description: "Protected full administrative access to the commerce platform",

	Permissions: PermissionCodes(),
}

var AdministratorRole = RoleDefinition{
	Code: RoleAdministrator,

	Name: "Administrator",

	Description: "Broad operational administration with ordinary staff management but without Super Admin authority",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		// -------------------------------------------------
		// Catalog / merchandising
		// -------------------------------------------------

		PermissionCatalogRead,
		PermissionCatalogWrite,
		PermissionCatalogImport,

		PermissionPromotionRead,
		PermissionPromotionManage,

		PermissionMerchandisingRead,
		PermissionMerchandisingManage,

		PermissionSearchAnalyticsRead,

		// -------------------------------------------------
		// Inventory / procurement
		// -------------------------------------------------

		PermissionInventoryRead,
		PermissionInventoryAdjust,

		PermissionProcurementRead,
		PermissionProcurementManage,

		// -------------------------------------------------
		// Orders
		// -------------------------------------------------

		PermissionOrderRead,
		PermissionOrderManage,

		// -------------------------------------------------
		// Invoices
		// -------------------------------------------------

		PermissionInvoiceRead,
		PermissionInvoiceIssue,
		PermissionInvoiceResend,

		// -------------------------------------------------
		// Customers
		// -------------------------------------------------

		PermissionCustomerRead,
		PermissionCustomerManage,

		PermissionCustomerBan,
		PermissionCustomerUnban,

		// -------------------------------------------------
		// Payments / refunds
		// -------------------------------------------------

		PermissionPaymentRead,

		/*
			Keep the current legacy refund permission while
			the existing refund routes are migrated to the
			granular refund permissions.
		*/
		PermissionPaymentRefund,

		PermissionRefundRead,
		PermissionRefundCreate,

		// -------------------------------------------------
		// Returns
		// -------------------------------------------------

		PermissionReturnRead,
		PermissionReturnManage,

		// -------------------------------------------------
		// Warehouse
		// -------------------------------------------------

		PermissionWarehouseRead,
		PermissionWarehouseManage,

		// -------------------------------------------------
		// Delivery / fulfillment
		// -------------------------------------------------

		PermissionDeliveryRead,
		PermissionDeliveryManage,
		PermissionDeliveryConfirmReceipt,

		// -------------------------------------------------
		// CRM / support
		// -------------------------------------------------

		PermissionCRMRead,
		PermissionCRMManage,

		// -------------------------------------------------
		// Sourcing
		// -------------------------------------------------

		PermissionSourcingRead,
		PermissionSourcingReview,
		PermissionSourcingOfferManage,
		PermissionSourcingFinalize,

		// -------------------------------------------------
		// Reviews
		// -------------------------------------------------

		PermissionReviewRead,
		PermissionReviewModerate,

		// -------------------------------------------------
		// Notifications
		// -------------------------------------------------

		PermissionNotificationRead,

		// -------------------------------------------------
		// Analytics
		// -------------------------------------------------

		PermissionAnalyticsRead,

		// -------------------------------------------------
		// Finance
		// -------------------------------------------------

		PermissionFinanceRead,
		PermissionFinanceManage,

		// -------------------------------------------------
		// Ordinary staff management
		// -------------------------------------------------

		PermissionStaffRead,
		PermissionStaffManage,
		PermissionStaffRoleAssign,
		PermissionStaffSecurityManage,
		PermissionStaffDelete,
		PermissionStaffBan,

		/*
			Administrator may inspect available roles,
			but cannot edit role definitions themselves.

			PermissionRoleManage remains Super Admin only.
		*/
		PermissionRoleRead,

		// -------------------------------------------------
		// Audit
		// -------------------------------------------------

		PermissionAuditRead,

		// -------------------------------------------------
		// Read-only platform configuration visibility
		// -------------------------------------------------

		PermissionSettingsRead,
		PermissionIntegrationRead,
		PermissionSystemOperationsRead,
	},
}

var SourcingRole = RoleDefinition{
	Code: RoleSourcing,

	Name: "Sourcing",

	Description: "Manage customer product requests, commercial offers, and finalized sourcing agreements",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionCatalogRead,

		PermissionInventoryRead,

		PermissionProcurementRead,

		PermissionCustomerRead,

		PermissionOrderRead,

		PermissionSourcingRead,
		PermissionSourcingReview,
		PermissionSourcingOfferManage,
		PermissionSourcingFinalize,

		PermissionNotificationRead,
	},
}

var CatalogRole = RoleDefinition{
	Code: RoleCatalog,

	Name: "Catalog",

	Description: "Manage products, variants, categories, catalog imports, promotions, and merchandising",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionCatalogRead,
		PermissionCatalogWrite,
		PermissionCatalogImport,

		PermissionPromotionRead,
		PermissionPromotionManage,

		PermissionMerchandisingRead,
		PermissionMerchandisingManage,

		PermissionSearchAnalyticsRead,

		PermissionReviewRead,

		PermissionAnalyticsRead,

		PermissionNotificationRead,
	},
}

var InventoryRole = RoleDefinition{
	Code: RoleInventory,

	Name: "Inventory",

	Description: "Manage stock availability, adjustments, procurement visibility, and inventory operations",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionCatalogRead,

		PermissionInventoryRead,
		PermissionInventoryAdjust,

		PermissionProcurementRead,
		PermissionProcurementManage,

		PermissionWarehouseRead,

		PermissionOrderRead,

		PermissionNotificationRead,
	},
}

var WarehouseRole = RoleDefinition{
	Code: RoleWarehouse,

	Name: "Warehouse",

	Description: "Manage receiving, warehouse stock handling, picking, packing, and warehouse operations",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionCatalogRead,

		PermissionInventoryRead,

		PermissionProcurementRead,

		PermissionOrderRead,

		PermissionWarehouseRead,
		PermissionWarehouseManage,

		PermissionDeliveryRead,

		PermissionNotificationRead,
	},
}

var FulfillmentRole = RoleDefinition{
	Code: RoleFulfillment,

	Name: "Fulfillment",

	Description: "Manage order fulfillment, shipment handoff, tracking, and delivery operations",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionCatalogRead,

		PermissionInventoryRead,

		PermissionOrderRead,

		PermissionWarehouseRead,
		PermissionWarehouseManage,

		PermissionDeliveryRead,
		PermissionDeliveryManage,
		PermissionDeliveryConfirmReceipt,

		PermissionNotificationRead,
	},
}

var FinanceRole = RoleDefinition{
	Code: RoleFinance,

	Name: "Finance / Accountant",

	Description: "Manage accounting operations and view payments, invoices, profitability, and financial reporting without inventory or support access",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionOrderRead,

		PermissionCustomerRead,

		PermissionPaymentRead,

		PermissionInvoiceRead,
		PermissionInvoiceIssue,
		PermissionInvoiceResend,
		PermissionInvoiceCreditNote,
		PermissionInvoiceVoid,

		PermissionRefundRead,

		PermissionFinanceRead,
		PermissionFinanceManage,
		PermissionFinanceExport,

		PermissionAnalyticsRead,

		PermissionNotificationRead,
	},
}

var ReturnsRole = RoleDefinition{
	Code: RoleReturns,

	Name: "Returns",

	Description: "Manage returns and create authorized refund requests without independently approving or processing high-risk refunds",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionOrderRead,

		PermissionCustomerRead,

		PermissionPaymentRead,

		PermissionReturnRead,
		PermissionReturnManage,

		PermissionRefundRead,
		PermissionRefundCreate,

		PermissionNotificationRead,
	},
}

var AnalyticsRole = RoleDefinition{
	Code: RoleAnalytics,

	Name: "Analytics",

	Description: "Read-only access to business analytics and operational reporting",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionAnalyticsRead,

		PermissionSearchAnalyticsRead,

		PermissionNotificationRead,
	},
}

var SecurityRole = RoleDefinition{
	Code: RoleSecurity,

	Name: "Security",

	Description: "Manage staff account security recovery, sessions, MFA state, and security auditing without Super Admin role authority",

	Permissions: []string{
		PermissionPanelAccess,
		PermissionDashboardRead,

		PermissionStaffRead,
		PermissionStaffSecurityManage,

		PermissionRoleRead,

		PermissionSecurityRead,
		PermissionSecurityManage,

		PermissionAuditRead,
		PermissionAuditExport,

		PermissionNotificationRead,
	},
}

/*
SystemRoles are maintained automatically by
SyncSystemAuthorization.

Do not place the existing Support Agent and Support
Supervisor roles here. They already belong to the
support domain and are bridged into Admin permissions by
syncSupportAdminBridgeTx().
*/
func SystemRoles() []RoleDefinition {
	return []RoleDefinition{
		SuperAdminRole,
		AdministratorRole,
		SourcingRole,
		CatalogRole,
		InventoryRole,
		WarehouseRole,
		FulfillmentRole,
		FinanceRole,
		ReturnsRole,
		AnalyticsRole,
		SecurityRole,
	}
}

/*
Protected roles require stronger authorization rules.

A normal Administrator must never be able to assign,
remove, create, modify, or otherwise escalate into one
of these roles.

Super Admin remains the root authority.

Security is protected as well because it contains staff
MFA/session recovery capabilities.
*/
var protectedRoleCodes = map[string]struct{}{
	RoleSuperAdmin:    {},
	RoleAdministrator: {},
	RoleSecurity:      {},
}

func IsProtectedRoleCode(
	roleCode string,
) bool {
	_, exists :=
		protectedRoleCodes[roleCode]

	return exists
}

func IsSuperAdminRoleCode(
	roleCode string,
) bool {
	return roleCode ==
		RoleSuperAdmin
}

func IsAdministratorRoleCode(
	roleCode string,
) bool {
	return roleCode ==
		RoleAdministrator
}

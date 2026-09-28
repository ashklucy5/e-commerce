package admin

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type createStaffRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`

	DeliveryMode string `json:"delivery_mode,omitempty"`

	RoleCodes []string `json:"role_codes"`
}

type staffStatusRequest struct {
	Status string `json:"status"`
}

type staffRolesRequest struct {
	RoleCodes []string `json:"role_codes"`
}

type staffPasswordResetRequest struct {
	Password string `json:"password"`
}

type createRoleRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`

	Description string `json:"description"`

	PermissionCodes []string `json:"permission_codes"`
}

type updateRoleRequest struct {
	Name *string `json:"name"`

	Description *string `json:"description"`
}

type rolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes"`
}

func (h *Handler) StaffMembers(
	c *gin.Context,
) {
	params, ok :=
		adminOperationalPagination(c)
	if !ok {
		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				c.Query("status"),
			),
		)

	/*
		List filtering may include every persisted staff state.

		This is intentionally broader than UpdateStaffStatus,
		which may only perform reversible ordinary lifecycle
		changes.

		"deleted" and "banned" are managed through dedicated
		security operations.
	*/
	switch status {
	case "",
		"pending_activation",
		"active",
		"suspended",
		"disabled",
		"deleted",
		"banned":

	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_STATUS",
				"Invalid staff status",
			),
		)

		return
	}

	roleCode :=
		strings.ToLower(
			strings.TrimSpace(
				c.Query("role"),
			),
		)

	if utf8.RuneCountInString(
		roleCode,
	) > 80 {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_ROLE",
				"role cannot exceed 80 characters",
			),
		)

		return
	}

	query, ok :=
		adminOperationalQuery(c)
	if !ok {
		return
	}

	queryID := ""

	if platformvalidation.IsUUID(
		query,
	) {
		queryID =
			query
	}

	result, err :=
		h.service.ListStaff(
			c.Request.Context(),
			params,
			StaffReadFilter{
				Status: status,

				RoleCode: roleCode,

				Query: query,

				QueryID: queryID,
			},
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": result.Meta,
		},
	)
}

func (h *Handler) StaffMember(
	c *gin.Context,
) {
	staffID, ok :=
		adminStaffIDParam(c)
	if !ok {
		return
	}

	result, err :=
		h.service.GetStaff(
			c.Request.Context(),
			staffID,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) CreateStaffMember(
	c *gin.Context,
) {
	var request createStaffRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_REQUEST",
				"Invalid staff request",
			),
		)

		return
	}

	request.FullName =
		strings.TrimSpace(
			request.FullName,
		)

	if request.FullName == "" ||
		utf8.RuneCountInString(
			request.FullName,
		) > 160 {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_NAME",
				"full_name is required and cannot exceed 160 characters",
			),
		)

		return
	}

	email, ok :=
		validAdminStaffEmail(
			request.Email,
		)
	if !ok {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_EMAIL",
				"email must be a valid email address",
			),
		)

		return
	}

	request.Phone =
		strings.TrimSpace(
			request.Phone,
		)

	if utf8.RuneCountInString(
		request.Phone,
	) > 40 {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PHONE",
				"phone cannot exceed 40 characters",
			),
		)

		return
	}

	request.DeliveryMode =
		strings.ToLower(
			strings.TrimSpace(
				request.DeliveryMode,
			),
		)

	if request.DeliveryMode == "" {
		request.DeliveryMode =
			StaffInvitationDeliveryManual
	}

	switch request.DeliveryMode {
	case StaffInvitationDeliveryManual,
		StaffInvitationDeliveryEmail:

	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_INVITATION_DELIVERY",
				"delivery_mode must be manual or email",
			),
		)

		return
	}

	if len(request.RoleCodes) == 0 {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"STAFF_ROLE_REQUIRED",
				"role_codes must contain at least one role",
			),
		)

		return
	}

	if len(request.RoleCodes) > 32 {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"TOO_MANY_STAFF_ROLES",
				"role_codes cannot contain more than 32 roles",
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.CreateStaff(
			c.Request.Context(),
			CreateStaffInput{
				FullName: request.FullName,

				Email: email,

				Phone: request.Phone,

				DeliveryMode: request.DeliveryMode,

				RoleCodes: request.RoleCodes,
			},
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}
func (h *Handler) UpdateStaffStatus(
	c *gin.Context,
) {
	staffID, ok :=
		adminStaffIDParam(c)
	if !ok {
		return
	}

	var request staffStatusRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_STATUS_REQUEST",
				"Invalid staff status request",
			),
		)

		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
		)

	/*
		Only ordinary reversible status transitions belong
		here.

		"deleted" is a dedicated permanent access-removal
		operation.

		"banned" will be owned by the dedicated ban system
		because it requires reason/history/issuer/expiry.
	*/
	switch status {
	case "active",
		"suspended",
		"disabled":

	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_STATUS",
				"status must be active, suspended, or disabled",
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.SetStaffStatus(
			c.Request.Context(),
			staffID,
			status,
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) DeleteStaffMember(
	c *gin.Context,
) {
	staffID, ok :=
		adminStaffIDParam(c)
	if !ok {
		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.DeleteStaff(
			c.Request.Context(),
			staffID,
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) ReplaceStaffMemberRoles(
	c *gin.Context,
) {
	staffID, ok :=
		adminStaffIDParam(c)
	if !ok {
		return
	}

	var request staffRolesRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_ROLES_REQUEST",
				"Invalid staff roles request",
			),
		)

		return
	}

	if len(request.RoleCodes) > 32 {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"TOO_MANY_STAFF_ROLES",
				"role_codes cannot contain more than 32 roles",
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.ReplaceStaffRoles(
			c.Request.Context(),
			staffID,
			request.RoleCodes,
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) ResetStaffMemberPassword(
	c *gin.Context,
) {
	staffID, ok :=
		adminStaffIDParam(c)
	if !ok {
		return
	}

	var request staffPasswordResetRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PASSWORD_REQUEST",
				"Invalid password reset request",
			),
		)

		return
	}

	if err :=
		adminauth.ValidateNewPassword(
			request.Password,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PASSWORD",
				err.Error(),
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.ResetStaffPassword(
			c.Request.Context(),
			staffID,
			request.Password,
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) ResetStaffMemberAdminMFA(
	c *gin.Context,
) {
	staffID, ok :=
		adminStaffIDParam(c)
	if !ok {
		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.ResetStaffAdminMFA(
			c.Request.Context(),
			staffID,
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Roles(
	c *gin.Context,
) {
	result, err :=
		h.service.ListRoles(
			c.Request.Context(),
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
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

func (h *Handler) Permissions(
	c *gin.Context,
) {
	result, err :=
		h.service.ListPermissions(
			c.Request.Context(),
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
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

func (h *Handler) Role(
	c *gin.Context,
) {
	roleID, ok :=
		adminRoleIDParam(c)
	if !ok {
		return
	}

	result, err :=
		h.service.GetRole(
			c.Request.Context(),
			roleID,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) CreateRole(
	c *gin.Context,
) {
	var request createRoleRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_REQUEST",
				"Invalid role request",
			),
		)

		return
	}

	request.Code =
		strings.ToLower(
			strings.TrimSpace(
				request.Code,
			),
		)

	request.Name =
		strings.TrimSpace(
			request.Name,
		)

	request.Description =
		strings.TrimSpace(
			request.Description,
		)

	if !validAdminRoleCode(
		request.Code,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_CODE",
				"role code must start with a lowercase letter and contain only lowercase letters, numbers, dots, hyphens, or underscores",
			),
		)

		return
	}

	if request.Name == "" ||
		utf8.RuneCountInString(
			request.Name,
		) > 120 {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_NAME",
				"name is required and cannot exceed 120 characters",
			),
		)

		return
	}

	if utf8.RuneCountInString(
		request.Description,
	) > 500 {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_DESCRIPTION",
				"description cannot exceed 500 characters",
			),
		)

		return
	}

	if len(request.PermissionCodes) > 128 {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"TOO_MANY_ROLE_PERMISSIONS",
				"permission_codes cannot contain more than 128 permissions",
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.CreateRole(
			c.Request.Context(),
			CreateRoleInput{
				Code: request.Code,

				Name: request.Name,

				Description: request.Description,

				PermissionCodes: request.PermissionCodes,
			},
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) UpdateRole(
	c *gin.Context,
) {
	roleID, ok :=
		adminRoleIDParam(c)
	if !ok {
		return
	}

	var request updateRoleRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_REQUEST",
				"Invalid role request",
			),
		)

		return
	}

	if request.Name == nil &&
		request.Description == nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"EMPTY_ROLE_UPDATE",
				"At least one role field must be supplied",
			),
		)

		return
	}

	if request.Name != nil {
		name :=
			strings.TrimSpace(
				*request.Name,
			)

		if name == "" ||
			utf8.RuneCountInString(
				name,
			) > 120 {

			platformerrors.Write(
				c,
				platformerrors.BadRequest(
					"INVALID_ROLE_NAME",
					"name is required and cannot exceed 120 characters",
				),
			)

			return
		}

		request.Name =
			&name
	}

	if request.Description != nil {
		description :=
			strings.TrimSpace(
				*request.Description,
			)

		if utf8.RuneCountInString(
			description,
		) > 500 {

			platformerrors.Write(
				c,
				platformerrors.BadRequest(
					"INVALID_ROLE_DESCRIPTION",
					"description cannot exceed 500 characters",
				),
			)

			return
		}

		request.Description =
			&description
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.UpdateRole(
			c.Request.Context(),
			roleID,
			RoleMetadataUpdate{
				Name: request.Name,

				Description: request.Description,
			},
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) ReplaceRolePermissionAssignments(
	c *gin.Context,
) {
	roleID, ok :=
		adminRoleIDParam(c)
	if !ok {
		return
	}

	var request rolePermissionsRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_PERMISSIONS_REQUEST",
				"Invalid role permissions request",
			),
		)

		return
	}

	if len(request.PermissionCodes) > 128 {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"TOO_MANY_ROLE_PERMISSIONS",
				"permission_codes cannot contain more than 128 permissions",
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err :=
		h.service.ReplaceRolePermissions(
			c.Request.Context(),
			roleID,
			request.PermissionCodes,
			metadata,
		)

	if writeAdminStaffRoleError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func adminStaffIDParam(
	c *gin.Context,
) (
	string,
	bool,
) {
	staffID :=
		strings.TrimSpace(
			c.Param("staff_id"),
		)

	if !platformvalidation.IsUUID(
		staffID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_ID",
				"staff_id must be a valid UUID",
			),
		)

		return "",
			false
	}

	return staffID,
		true
}

func adminRoleIDParam(
	c *gin.Context,
) (
	string,
	bool,
) {
	roleID :=
		strings.TrimSpace(
			c.Param("role_id"),
		)

	if !platformvalidation.IsUUID(
		roleID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_ID",
				"role_id must be a valid UUID",
			),
		)

		return "",
			false
	}

	return roleID,
		true
}

func adminMutationMetadata(
	c *gin.Context,
) (
	AdminActionMetadata,
	bool,
) {
	metadata, ok :=
		adminActionMetadataFromContext(c)
	if ok {
		return metadata,
			true
	}

	platformerrors.Write(
		c,
		platformerrors.Unauthorized(
			"ADMIN_AUTH_REQUIRED",
			"Admin authentication required",
		),
	)

	return AdminActionMetadata{},
		false
}

func validAdminStaffEmail(
	value string,
) (
	string,
	bool,
) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" ||
		utf8.RuneCountInString(
			value,
		) > 255 {

		return "",
			false
	}

	parsed, err :=
		mail.ParseAddress(
			value,
		)

	if err != nil ||
		!strings.EqualFold(
			parsed.Address,
			value,
		) {

		return "",
			false
	}

	return value,
		true
}

func validAdminRoleCode(
	value string,
) bool {
	if len(value) == 0 ||
		len(value) > 80 {

		return false
	}

	if value[0] < 'a' ||
		value[0] > 'z' {

		return false
	}

	for index :=
		1; index < len(value); index++ {

		character :=
			value[index]

		if character >= 'a' &&
			character <= 'z' {

			continue
		}

		if character >= '0' &&
			character <= '9' {

			continue
		}

		switch character {
		case '.',
			'-',
			'_':

			continue
		}

		return false
	}

	return true
}

func writeAdminStaffRoleError(
	c *gin.Context,
	err error,
) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(
		err,
		ErrAdminInvalidStaffOnboarding,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_ONBOARDING",
				"Invalid staff onboarding request",
			),
		)

	case errors.Is(
		err,
		ErrAdminInvalidInvitationDelivery,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_INVITATION_DELIVERY",
				"Invalid staff invitation delivery mode",
			),
		)

	case errors.Is(
		err,
		ErrAdminEmailInvitationDisabled,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_EMAIL_INVITATION_NOT_CONFIGURED",
				"Email invitation delivery is not configured; use manual delivery",
			),
		)

	case errors.Is(
		err,
		ErrAdminStaffActivationManagedSeparately,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACTIVATION_IN_PROGRESS",
				"This staff account must complete its invitation activation flow",
			),
		)
	case errors.Is(
		err,
		ErrAdminStaffNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"STAFF_NOT_FOUND",
				"Staff account not found",
			),
		)

	case errors.Is(
		err,
		ErrAdminRoleNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"ROLE_NOT_FOUND",
				"Role not found",
			),
		)

	case errors.Is(
		err,
		ErrAdminStaffConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ALREADY_EXISTS",
				"A staff account with the supplied identity already exists",
			),
		)

	case errors.Is(
		err,
		ErrAdminRoleConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ROLE_ALREADY_EXISTS",
				"A role with that code already exists",
			),
		)

	case errors.Is(
		err,
		ErrAdminStaffRolesInvalid,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_ROLES",
				"One or more role codes do not exist",
			),
		)

	case errors.Is(
		err,
		ErrAdminPermissionCodesInvalid,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ROLE_PERMISSIONS",
				"One or more permission codes do not exist",
			),
		)

	case errors.Is(
		err,
		ErrInvalidAdminStaffStatus,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_STATUS",
				"Invalid staff status",
			),
		)

	case errors.Is(
		err,
		ErrAdminSystemRoleImmutable,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SYSTEM_ROLE_IMMUTABLE",
				"System roles cannot be modified",
			),
		)

	case errors.Is(
		err,
		ErrAdminSelfRoleMutation,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SELF_ROLE_MUTATION_NOT_ALLOWED",
				"Use another authorized administrator to change your role assignments",
			),
		)

	case errors.Is(
		err,
		ErrAdminSelfDisable,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SELF_DISABLE_NOT_ALLOWED",
				"An administrator cannot suspend or disable their own account",
			),
		)

	case errors.Is(
		err,
		ErrAdminSelfPasswordReset,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SELF_PASSWORD_RESET_NOT_ALLOWED",
				"Use the authenticated password-change flow to change your own password",
			),
		)

	case errors.Is(
		err,
		ErrAdminSelfMFAReset,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SELF_MFA_RESET_NOT_ALLOWED",
				"Use the authenticated MFA replacement or recovery flow for your own account",
			),
		)

	case errors.Is(
		err,
		ErrAdminSelfDelete,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SELF_DELETE_NOT_ALLOWED",
				"An administrator cannot permanently delete their own staff account",
			),
		)

	case errors.Is(
		err,
		ErrAdminStaffDeleted,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACCOUNT_DELETED",
				"This staff account has been permanently deleted and cannot be reactivated or modified",
			),
		)

	case errors.Is(
		err,
		ErrAdminStaffBanManagedSeparately,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACCOUNT_BANNED",
				"This staff account is banned and must be managed through the staff ban system",
			),
		)

	case errors.Is(
		err,
		ErrAdminProtectedStaffMutation,
	):
		platformerrors.Write(
			c,
			platformerrors.Forbidden(
				"PROTECTED_STAFF_REQUIRES_SUPER_ADMIN",
				"This staff account can only be managed by a Super Admin",
			),
		)

	case errors.Is(
		err,
		ErrAdminProtectedRoleAssignment,
	):
		platformerrors.Write(
			c,
			platformerrors.Forbidden(
				"PROTECTED_ROLE_REQUIRES_SUPER_ADMIN",
				"One or more requested roles can only be assigned by a Super Admin",
			),
		)

	case errors.Is(
		err,
		ErrAdminRoleAssignmentNotAllowed,
	):
		platformerrors.Write(
			c,
			platformerrors.Forbidden(
				"ROLE_ASSIGNMENT_NOT_ALLOWED",
				"One or more requested roles cannot be assigned by the current administrator",
			),
		)

	case errors.Is(
		err,
		ErrAdminLastSuperAdmin,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"LAST_SUPER_ADMIN_REQUIRED",
				"At least one active Super Admin must remain",
			),
		)

	case errors.Is(
		err,
		adminauth.ErrPasswordRequired,
	),
		errors.Is(
			err,
			adminauth.ErrPasswordTooShort,
		),
		errors.Is(
			err,
			adminauth.ErrPasswordTooLong,
		),
		errors.Is(
			err,
			adminauth.ErrPasswordCommon,
		):

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PASSWORD",
				err.Error(),
			),
		)

	default:
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)
	}

	return true
}

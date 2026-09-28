package customer

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requireAuth gin.HandlerFunc,
) {
	customers :=
		group.Group(
			"/customers",
		)

	me :=
		customers.Group(
			"/me",
		)

	me.Use(
		requireAuth,
	)

	me.GET(
		"",
		handler.GetMe,
	)

	me.PATCH(
		"",
		handler.UpdateMe,
	)

	me.GET(
		"/overview",
		handler.GetAccountOverview,
	)

	me.POST(
		"/avatar/upload-target",
		handler.CreateAvatarUploadTarget,
	)

	me.POST(
		"/avatar/complete",
		handler.CompleteAvatarUpload,
	)

	me.DELETE(
		"/avatar",
		handler.DeleteAvatar,
	)

	me.GET(
		"/addresses",
		handler.ListAddresses,
	)

	me.POST(
		"/addresses",
		handler.CreateAddress,
	)

	me.PATCH(
		"/addresses/:address_id",
		handler.UpdateAddress,
	)

	me.DELETE(
		"/addresses/:address_id",
		handler.DeleteAddress,
	)

	me.GET(
		"/preferences",
		handler.GetPreferences,
	)

	me.PATCH(
		"/preferences",
		handler.UpdatePreferences,
	)

	me.GET(
		"/notification-preferences",
		handler.GetNotificationPreferences,
	)

	me.PATCH(
		"/notification-preferences",
		handler.UpdateNotificationPreferences,
	)

	me.GET(
		"/privacy-settings",
		handler.GetPrivacySettings,
	)

	me.PATCH(
		"/privacy-settings",
		handler.UpdatePrivacySettings,
	)

	me.GET(
		"/style-profile",
		handler.GetStyleProfile,
	)

	me.PUT(
		"/style-profile",
		handler.PutStyleProfile,
	)

	me.GET(
		"/sizes",
		handler.ListSavedSizes,
	)

	me.POST(
		"/sizes",
		handler.CreateSavedSize,
	)

	me.PUT(
		"/sizes/:size_id",
		handler.ReplaceSavedSize,
	)

	me.DELETE(
		"/sizes/:size_id",
		handler.DeleteSavedSize,
	)
}

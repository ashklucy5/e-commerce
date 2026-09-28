package customer

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidPreferences = errors.New(
		"invalid customer preferences",
	)

	currencyCodePattern = regexp.MustCompile(
		`^[A-Z]{3}$`,
	)
)

type Preferences struct {
	CustomerID        string `json:"customer_id"`
	Locale            string `json:"locale"`
	AssistantLanguage string `json:"assistant_language"`
	Currency          string `json:"currency"`
}

type UpdatePreferencesRequest struct {
	Locale            *string `json:"locale"`
	AssistantLanguage *string `json:"assistant_language"`
	Currency          *string `json:"currency"`
}

type NotificationPreferences struct {
	CustomerID           string `json:"customer_id"`
	OrderUpdatesSMS      bool   `json:"order_updates_sms"`
	OrderUpdatesEmail    bool   `json:"order_updates_email"`
	DeliveryUpdatesSMS   bool   `json:"delivery_updates_sms"`
	DeliveryUpdatesEmail bool   `json:"delivery_updates_email"`
	SupportUpdatesSMS    bool   `json:"support_updates_sms"`
	SupportUpdatesEmail  bool   `json:"support_updates_email"`
	PromotionsSMS        bool   `json:"promotions_sms"`
	PromotionsEmail      bool   `json:"promotions_email"`
	RecommendationsEmail bool   `json:"recommendations_email"`
	PushEnabled          bool   `json:"push_enabled"`
}

type UpdateNotificationPreferencesRequest struct {
	OrderUpdatesSMS      *bool `json:"order_updates_sms"`
	OrderUpdatesEmail    *bool `json:"order_updates_email"`
	DeliveryUpdatesSMS   *bool `json:"delivery_updates_sms"`
	DeliveryUpdatesEmail *bool `json:"delivery_updates_email"`
	SupportUpdatesSMS    *bool `json:"support_updates_sms"`
	SupportUpdatesEmail  *bool `json:"support_updates_email"`
	PromotionsSMS        *bool `json:"promotions_sms"`
	PromotionsEmail      *bool `json:"promotions_email"`
	RecommendationsEmail *bool `json:"recommendations_email"`
	PushEnabled          *bool `json:"push_enabled"`
}

func (s *Service) GetPreferences(
	ctx context.Context,
	customerID string,
) (Preferences, error) {
	return s.repository.GetPreferences(
		ctx,
		customerID,
	)
}

func (s *Service) UpdatePreferences(
	ctx context.Context,
	customerID string,
	request UpdatePreferencesRequest,
) (Preferences, error) {
	if request.Locale == nil &&
		request.AssistantLanguage == nil &&
		request.Currency == nil {
		return Preferences{},
			ErrNoChanges
	}

	if request.Locale != nil {
		value :=
			strings.ToLower(
				strings.TrimSpace(
					*request.Locale,
				),
			)

		switch value {
		case "en", "bn":
			request.Locale =
				&value

		default:
			return Preferences{},
				ErrInvalidPreferences
		}
	}

	if request.AssistantLanguage != nil {
		value :=
			strings.ToLower(
				strings.TrimSpace(
					*request.AssistantLanguage,
				),
			)

		switch value {
		case "auto", "en", "bn", "banglish":
			request.AssistantLanguage =
				&value

		default:
			return Preferences{},
				ErrInvalidPreferences
		}
	}

	if request.Currency != nil {
		value :=
			strings.ToUpper(
				strings.TrimSpace(
					*request.Currency,
				),
			)

		if !currencyCodePattern.MatchString(
			value,
		) {
			return Preferences{},
				ErrInvalidPreferences
		}

		// Current transactional commerce is Bangladesh-first.
		// Do not imply unsupported currency conversion.
		if value != "BDT" {
			return Preferences{},
				ErrInvalidPreferences
		}

		request.Currency =
			&value
	}

	return s.repository.UpdatePreferences(
		ctx,
		customerID,
		request,
	)
}

func (s *Service) GetNotificationPreferences(
	ctx context.Context,
	customerID string,
) (NotificationPreferences, error) {
	return s.repository.GetNotificationPreferences(
		ctx,
		customerID,
	)
}

func (s *Service) UpdateNotificationPreferences(
	ctx context.Context,
	customerID string,
	request UpdateNotificationPreferencesRequest,
) (NotificationPreferences, error) {
	if request.OrderUpdatesSMS == nil &&
		request.OrderUpdatesEmail == nil &&
		request.DeliveryUpdatesSMS == nil &&
		request.DeliveryUpdatesEmail == nil &&
		request.SupportUpdatesSMS == nil &&
		request.SupportUpdatesEmail == nil &&
		request.PromotionsSMS == nil &&
		request.PromotionsEmail == nil &&
		request.RecommendationsEmail == nil &&
		request.PushEnabled == nil {
		return NotificationPreferences{},
			ErrNoChanges
	}

	return s.repository.UpdateNotificationPreferences(
		ctx,
		customerID,
		request,
	)
}

func (r *Repository) GetPreferences(
	ctx context.Context,
	customerID string,
) (Preferences, error) {
	_, err :=
		r.db.Exec(
			ctx,
			`
				INSERT INTO customer_preferences (
					customer_id
				)
				VALUES (
					$1::uuid
				)
				ON CONFLICT (customer_id)
				DO NOTHING
			`,
			customerID,
		)
	if err != nil {
		return Preferences{},
			fmt.Errorf(
				"ensure customer preferences: %w",
				err,
			)
	}

	return r.selectPreferences(
		ctx,
		customerID,
	)
}

func (r *Repository) UpdatePreferences(
	ctx context.Context,
	customerID string,
	request UpdatePreferencesRequest,
) (Preferences, error) {
	var result Preferences

	err :=
		r.db.QueryRow(
			ctx,
			`
				INSERT INTO customer_preferences (
					customer_id,
					locale,
					assistant_language,
					currency
				)
				VALUES (
					$1::uuid,
					COALESCE($2::varchar, 'en'),
					COALESCE($3::varchar, 'auto'),
					COALESCE($4::varchar, 'BDT')
				)
				ON CONFLICT (customer_id)
				DO UPDATE SET
					locale =
						COALESCE(
							$2::varchar,
							customer_preferences.locale
						),
					assistant_language =
						COALESCE(
							$3::varchar,
							customer_preferences.assistant_language
						),
					currency =
						COALESCE(
							$4::varchar,
							customer_preferences.currency
						),
					updated_at = now()
				RETURNING
					customer_id::text,
					locale,
					assistant_language,
					currency
			`,
			customerID,
			stringValue(
				request.Locale,
			),
			stringValue(
				request.AssistantLanguage,
			),
			stringValue(
				request.Currency,
			),
		).Scan(
			&result.CustomerID,
			&result.Locale,
			&result.AssistantLanguage,
			&result.Currency,
		)
	if err != nil {
		return Preferences{},
			fmt.Errorf(
				"update customer preferences: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) selectPreferences(
	ctx context.Context,
	customerID string,
) (Preferences, error) {
	var result Preferences

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					customer_id::text,
					locale,
					assistant_language,
					currency
				FROM customer_preferences
				WHERE customer_id = $1::uuid
			`,
			customerID,
		).Scan(
			&result.CustomerID,
			&result.Locale,
			&result.AssistantLanguage,
			&result.Currency,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Preferences{},
				ErrCustomerNotFound
		}

		return Preferences{},
			fmt.Errorf(
				"get customer preferences: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) GetNotificationPreferences(
	ctx context.Context,
	customerID string,
) (NotificationPreferences, error) {
	_, err :=
		r.db.Exec(
			ctx,
			`
				INSERT INTO customer_notification_preferences (
					customer_id
				)
				VALUES (
					$1::uuid
				)
				ON CONFLICT (customer_id)
				DO NOTHING
			`,
			customerID,
		)
	if err != nil {
		return NotificationPreferences{},
			fmt.Errorf(
				"ensure customer notification preferences: %w",
				err,
			)
	}

	return r.selectNotificationPreferences(
		ctx,
		customerID,
	)
}

func (r *Repository) UpdateNotificationPreferences(
	ctx context.Context,
	customerID string,
	request UpdateNotificationPreferencesRequest,
) (NotificationPreferences, error) {
	row :=
		r.db.QueryRow(
			ctx,
			`
				INSERT INTO customer_notification_preferences (
					customer_id,
					order_updates_sms,
					order_updates_email,
					delivery_updates_sms,
					delivery_updates_email,
					support_updates_sms,
					support_updates_email,
					promotions_sms,
					promotions_email,
					recommendations_email,
					push_enabled
				)
				VALUES (
					$1::uuid,
					COALESCE($2::boolean, true),
					COALESCE($3::boolean, true),
					COALESCE($4::boolean, true),
					COALESCE($5::boolean, true),
					COALESCE($6::boolean, true),
					COALESCE($7::boolean, true),
					COALESCE($8::boolean, false),
					COALESCE($9::boolean, false),
					COALESCE($10::boolean, false),
					COALESCE($11::boolean, false)
				)
				ON CONFLICT (customer_id)
				DO UPDATE SET
					order_updates_sms =
						COALESCE(
							$2::boolean,
							customer_notification_preferences.order_updates_sms
						),
					order_updates_email =
						COALESCE(
							$3::boolean,
							customer_notification_preferences.order_updates_email
						),
					delivery_updates_sms =
						COALESCE(
							$4::boolean,
							customer_notification_preferences.delivery_updates_sms
						),
					delivery_updates_email =
						COALESCE(
							$5::boolean,
							customer_notification_preferences.delivery_updates_email
						),
					support_updates_sms =
						COALESCE(
							$6::boolean,
							customer_notification_preferences.support_updates_sms
						),
					support_updates_email =
						COALESCE(
							$7::boolean,
							customer_notification_preferences.support_updates_email
						),
					promotions_sms =
						COALESCE(
							$8::boolean,
							customer_notification_preferences.promotions_sms
						),
					promotions_email =
						COALESCE(
							$9::boolean,
							customer_notification_preferences.promotions_email
						),
					recommendations_email =
						COALESCE(
							$10::boolean,
							customer_notification_preferences.recommendations_email
						),
					push_enabled =
						COALESCE(
							$11::boolean,
							customer_notification_preferences.push_enabled
						),
					updated_at = now()
				RETURNING
					customer_id::text,
					order_updates_sms,
					order_updates_email,
					delivery_updates_sms,
					delivery_updates_email,
					support_updates_sms,
					support_updates_email,
					promotions_sms,
					promotions_email,
					recommendations_email,
					push_enabled
			`,
			customerID,
			boolValue(
				request.OrderUpdatesSMS,
			),
			boolValue(
				request.OrderUpdatesEmail,
			),
			boolValue(
				request.DeliveryUpdatesSMS,
			),
			boolValue(
				request.DeliveryUpdatesEmail,
			),
			boolValue(
				request.SupportUpdatesSMS,
			),
			boolValue(
				request.SupportUpdatesEmail,
			),
			boolValue(
				request.PromotionsSMS,
			),
			boolValue(
				request.PromotionsEmail,
			),
			boolValue(
				request.RecommendationsEmail,
			),
			boolValue(
				request.PushEnabled,
			),
		)

	result, err :=
		scanNotificationPreferences(
			row,
		)
	if err != nil {
		return NotificationPreferences{},
			fmt.Errorf(
				"update customer notification preferences: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) selectNotificationPreferences(
	ctx context.Context,
	customerID string,
) (NotificationPreferences, error) {
	result, err :=
		scanNotificationPreferences(
			r.db.QueryRow(
				ctx,
				`
					SELECT
						customer_id::text,
						order_updates_sms,
						order_updates_email,
						delivery_updates_sms,
						delivery_updates_email,
						support_updates_sms,
						support_updates_email,
						promotions_sms,
						promotions_email,
						recommendations_email,
						push_enabled
					FROM customer_notification_preferences
					WHERE customer_id = $1::uuid
				`,
				customerID,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return NotificationPreferences{},
				ErrCustomerNotFound
		}

		return NotificationPreferences{},
			fmt.Errorf(
				"get customer notification preferences: %w",
				err,
			)
	}

	return result, nil
}

type notificationPreferencesScanner interface {
	Scan(dest ...any) error
}

func scanNotificationPreferences(
	row notificationPreferencesScanner,
) (NotificationPreferences, error) {
	var result NotificationPreferences

	err :=
		row.Scan(
			&result.CustomerID,
			&result.OrderUpdatesSMS,
			&result.OrderUpdatesEmail,
			&result.DeliveryUpdatesSMS,
			&result.DeliveryUpdatesEmail,
			&result.SupportUpdatesSMS,
			&result.SupportUpdatesEmail,
			&result.PromotionsSMS,
			&result.PromotionsEmail,
			&result.RecommendationsEmail,
			&result.PushEnabled,
		)

	return result, err
}

func stringValue(
	value *string,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func boolValue(
	value *bool,
) any {
	if value == nil {
		return nil
	}

	return *value
}

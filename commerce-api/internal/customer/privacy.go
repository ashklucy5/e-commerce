package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type PrivacySettings struct {
	CustomerID                 string `json:"customer_id"`
	PersonalizationEnabled     bool   `json:"personalization_enabled"`
	SearchHistoryEnabled       bool   `json:"search_history_enabled"`
	RecentlyViewedEnabled      bool   `json:"recently_viewed_enabled"`
	SaveTryOnMedia             bool   `json:"save_tryon_media"`
	UseTryOnForPersonalization bool   `json:"use_tryon_for_personalization"`
}

type UpdatePrivacySettingsRequest struct {
	PersonalizationEnabled     *bool `json:"personalization_enabled"`
	SearchHistoryEnabled       *bool `json:"search_history_enabled"`
	RecentlyViewedEnabled      *bool `json:"recently_viewed_enabled"`
	SaveTryOnMedia             *bool `json:"save_tryon_media"`
	UseTryOnForPersonalization *bool `json:"use_tryon_for_personalization"`
}

func (s *Service) GetPrivacySettings(
	ctx context.Context,
	customerID string,
) (PrivacySettings, error) {
	return s.repository.GetPrivacySettings(
		ctx,
		customerID,
	)
}

func (s *Service) UpdatePrivacySettings(
	ctx context.Context,
	customerID string,
	request UpdatePrivacySettingsRequest,
) (PrivacySettings, error) {
	if request.PersonalizationEnabled == nil &&
		request.SearchHistoryEnabled == nil &&
		request.RecentlyViewedEnabled == nil &&
		request.SaveTryOnMedia == nil &&
		request.UseTryOnForPersonalization == nil {
		return PrivacySettings{},
			ErrNoChanges
	}

	return s.repository.UpdatePrivacySettings(
		ctx,
		customerID,
		request,
	)
}

func (r *Repository) GetPrivacySettings(
	ctx context.Context,
	customerID string,
) (PrivacySettings, error) {
	_, err :=
		r.db.Exec(
			ctx,
			`
				INSERT INTO customer_privacy_settings (
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
		return PrivacySettings{},
			fmt.Errorf(
				"ensure customer privacy settings: %w",
				err,
			)
	}

	return r.selectPrivacySettings(
		ctx,
		customerID,
	)
}

func (r *Repository) UpdatePrivacySettings(
	ctx context.Context,
	customerID string,
	request UpdatePrivacySettingsRequest,
) (PrivacySettings, error) {
	var result PrivacySettings

	err :=
		r.db.QueryRow(
			ctx,
			`
				INSERT INTO customer_privacy_settings (
					customer_id,
					personalization_enabled,
					search_history_enabled,
					recently_viewed_enabled,
					save_tryon_media,
					use_tryon_for_personalization
				)
				VALUES (
					$1::uuid,
					COALESCE($2::boolean, true),
					COALESCE($3::boolean, true),
					COALESCE($4::boolean, true),
					COALESCE($5::boolean, false),
					COALESCE($6::boolean, false)
				)
				ON CONFLICT (customer_id)
				DO UPDATE SET
					personalization_enabled =
						COALESCE(
							$2::boolean,
							customer_privacy_settings.personalization_enabled
						),
					search_history_enabled =
						COALESCE(
							$3::boolean,
							customer_privacy_settings.search_history_enabled
						),
					recently_viewed_enabled =
						COALESCE(
							$4::boolean,
							customer_privacy_settings.recently_viewed_enabled
						),
					save_tryon_media =
						COALESCE(
							$5::boolean,
							customer_privacy_settings.save_tryon_media
						),
					use_tryon_for_personalization =
						COALESCE(
							$6::boolean,
							customer_privacy_settings.use_tryon_for_personalization
						),
					updated_at = now()
				RETURNING
					customer_id::text,
					personalization_enabled,
					search_history_enabled,
					recently_viewed_enabled,
					save_tryon_media,
					use_tryon_for_personalization
			`,
			customerID,
			boolValue(
				request.PersonalizationEnabled,
			),
			boolValue(
				request.SearchHistoryEnabled,
			),
			boolValue(
				request.RecentlyViewedEnabled,
			),
			boolValue(
				request.SaveTryOnMedia,
			),
			boolValue(
				request.UseTryOnForPersonalization,
			),
		).Scan(
			&result.CustomerID,
			&result.PersonalizationEnabled,
			&result.SearchHistoryEnabled,
			&result.RecentlyViewedEnabled,
			&result.SaveTryOnMedia,
			&result.UseTryOnForPersonalization,
		)
	if err != nil {
		return PrivacySettings{},
			fmt.Errorf(
				"update customer privacy settings: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) selectPrivacySettings(
	ctx context.Context,
	customerID string,
) (PrivacySettings, error) {
	var result PrivacySettings

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					customer_id::text,
					personalization_enabled,
					search_history_enabled,
					recently_viewed_enabled,
					save_tryon_media,
					use_tryon_for_personalization
				FROM customer_privacy_settings
				WHERE customer_id = $1::uuid
			`,
			customerID,
		).Scan(
			&result.CustomerID,
			&result.PersonalizationEnabled,
			&result.SearchHistoryEnabled,
			&result.RecentlyViewedEnabled,
			&result.SaveTryOnMedia,
			&result.UseTryOnForPersonalization,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return PrivacySettings{},
				ErrCustomerNotFound
		}

		return PrivacySettings{},
			fmt.Errorf(
				"get customer privacy settings: %w",
				err,
			)
	}

	return result, nil
}

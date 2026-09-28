package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Repository) GetCustomerSecuritySnapshot(
	ctx context.Context,
	customerID string,
	currentAccessTokenHash string,
) (string, int64, error) {
	var currentSessionID string

	var activeSessionCount int64

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					s.id::text,
					(
						SELECT count(*)
						FROM auth_sessions active_session
						WHERE
							active_session.customer_id = s.customer_id
							AND active_session.revoked_at IS NULL
							AND active_session.refresh_expires_at > now()
					)::bigint
				FROM auth_sessions s
				WHERE
					s.customer_id = $1::uuid
					AND s.access_token_hash = $2
					AND s.revoked_at IS NULL
					AND s.access_expires_at > now()
			`,
			customerID,
			currentAccessTokenHash,
		).Scan(
			&currentSessionID,
			&activeSessionCount,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return "", 0,
				ErrInvalidAccessToken
		}

		return "", 0,
			fmt.Errorf(
				"get customer security snapshot: %w",
				err,
			)
	}

	return currentSessionID,
		activeSessionCount,
		nil
}

func (r *Repository) ListCustomerSessions(
	ctx context.Context,
	customerID string,
	currentAccessTokenHash string,
) ([]CustomerSession, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					access_token_hash = $2 AS current,
					created_at,
					last_used_at,
					access_expires_at,
					refresh_expires_at
				FROM auth_sessions
				WHERE
					customer_id = $1::uuid
					AND revoked_at IS NULL
					AND refresh_expires_at > now()
				ORDER BY
					current DESC,
					COALESCE(last_used_at, created_at) DESC,
					created_at DESC,
					id DESC
			`,
			customerID,
			currentAccessTokenHash,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer sessions: %w",
				err,
			)
	}

	defer rows.Close()

	sessions :=
		make(
			[]CustomerSession,
			0,
		)

	for rows.Next() {
		var session CustomerSession

		var lastUsedAt pgtype.Timestamptz

		if err :=
			rows.Scan(
				&session.ID,
				&session.Current,
				&session.CreatedAt,
				&lastUsedAt,
				&session.AccessExpiresAt,
				&session.RefreshExpiresAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan customer session: %w",
					err,
				)
		}

		if lastUsedAt.Valid {
			value :=
				lastUsedAt.Time

			session.LastUsedAt =
				&value
		}

		sessions =
			append(
				sessions,
				session,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate customer sessions: %w",
				err,
			)
	}

	return sessions,
		nil
}

func (r *Repository) RevokeCustomerSession(
	ctx context.Context,
	customerID string,
	sessionID string,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE auth_sessions
				SET
					revoked_at = now(),
					updated_at = now()
				WHERE
					customer_id = $1::uuid
					AND id = $2::uuid
					AND revoked_at IS NULL
					AND refresh_expires_at > now()
			`,
			customerID,
			sessionID,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke customer session: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrCustomerSessionNotFound
	}

	return nil
}

func (r *Repository) RevokeOtherActiveCustomerSessions(
	ctx context.Context,
	customerID string,
	currentAccessTokenHash string,
) (int64, error) {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE auth_sessions
				SET
					revoked_at = now(),
					updated_at = now()
				WHERE
					customer_id = $1::uuid
					AND access_token_hash <> $2
					AND revoked_at IS NULL
					AND refresh_expires_at > now()
			`,
			customerID,
			currentAccessTokenHash,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"revoke other customer sessions: %w",
				err,
			)
	}

	return tag.RowsAffected(),
		nil
}

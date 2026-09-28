package adminauth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	DefaultAdminLoginMaxFailures = 5

	DefaultAdminLoginWindow = 15 * time.Minute

	DefaultAdminLoginBlockTTL = 15 * time.Minute

	DefaultAdminMFAMaxFailures = 10

	DefaultAdminMFAWindow = 15 * time.Minute

	DefaultAdminMFABlockTTL = 15 * time.Minute
)

type LockoutConfig struct {
	MaxFailures int

	Window time.Duration

	BlockTTL time.Duration

	MFAMaxFailures int

	MFAWindow time.Duration

	MFABlockTTL time.Duration
}

type LockoutService struct {
	redis *redis.Client

	config LockoutConfig
}

var recordFailureScript = redis.NewScript(
	`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
if count >= tonumber(ARGV[2]) then
  redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
end
return count
`,
)

func DefaultLockoutConfig() LockoutConfig {
	return LockoutConfig{
		MaxFailures: DefaultAdminLoginMaxFailures,

		Window: DefaultAdminLoginWindow,

		BlockTTL: DefaultAdminLoginBlockTTL,

		MFAMaxFailures: DefaultAdminMFAMaxFailures,

		MFAWindow: DefaultAdminMFAWindow,

		MFABlockTTL: DefaultAdminMFABlockTTL,
	}
}

func NewLockoutService(
	client *redis.Client,
	cfg LockoutConfig,
) (
	*LockoutService,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"%w: Redis client is required",
				ErrAuthUnavailable,
			)
	}

	if cfg.MaxFailures <= 0 ||
		cfg.Window <= 0 ||
		cfg.BlockTTL <= 0 ||
		cfg.MFAMaxFailures <= 0 ||
		cfg.MFAWindow <= 0 ||
		cfg.MFABlockTTL <= 0 {

		return nil,
			fmt.Errorf(
				"%w: invalid lockout configuration",
				ErrAuthUnavailable,
			)
	}

	return &LockoutService{
			redis: client,

			config: cfg,
		},
		nil
}

func (s *LockoutService) IsBlocked(
	ctx context.Context,
	identifier string,
	ipAddress string,
) (
	bool,
	error,
) {
	return s.anyBlocked(
		ctx,
		s.identifierBlockKey(
			identifier,
		),
		s.ipBlockKey(
			ipAddress,
		),
	)
}

func (s *LockoutService) RecordFailure(
	ctx context.Context,
	identifier string,
	ipAddress string,
) error {
	if err :=
		s.recordDimensionFailure(
			ctx,
			s.identifierCounterKey(
				identifier,
			),
			s.identifierBlockKey(
				identifier,
			),
			s.config.Window,
			s.config.MaxFailures,
			s.config.BlockTTL,
		); err != nil {

		return err
	}

	return s.recordDimensionFailure(
		ctx,
		s.ipCounterKey(
			ipAddress,
		),
		s.ipBlockKey(
			ipAddress,
		),
		s.config.Window,
		s.config.MaxFailures,
		s.config.BlockTTL,
	)
}

func (s *LockoutService) RecordSuccess(
	ctx context.Context,
	identifier string,
) error {
	return s.clearDimension(
		ctx,
		s.identifierCounterKey(
			identifier,
		),
		s.identifierBlockKey(
			identifier,
		),
		"reset Admin identifier lockout",
	)
}

func (s *LockoutService) IsMFABlocked(
	ctx context.Context,
	staffAccountID string,
	ipAddress string,
) (
	bool,
	error,
) {
	return s.anyBlocked(
		ctx,
		s.mfaStaffBlockKey(
			staffAccountID,
		),
		s.mfaIPBlockKey(
			ipAddress,
		),
	)
}

func (s *LockoutService) RecordMFAFailure(
	ctx context.Context,
	staffAccountID string,
	ipAddress string,
) error {
	if err :=
		s.recordDimensionFailure(
			ctx,
			s.mfaStaffCounterKey(
				staffAccountID,
			),
			s.mfaStaffBlockKey(
				staffAccountID,
			),
			s.config.MFAWindow,
			s.config.MFAMaxFailures,
			s.config.MFABlockTTL,
		); err != nil {

		return err
	}

	return s.recordDimensionFailure(
		ctx,
		s.mfaIPCounterKey(
			ipAddress,
		),
		s.mfaIPBlockKey(
			ipAddress,
		),
		s.config.MFAWindow,
		s.config.MFAMaxFailures,
		s.config.MFABlockTTL,
	)
}

func (s *LockoutService) RecordMFASuccess(
	ctx context.Context,
	staffAccountID string,
) error {
	return s.clearDimension(
		ctx,
		s.mfaStaffCounterKey(
			staffAccountID,
		),
		s.mfaStaffBlockKey(
			staffAccountID,
		),
		"reset Admin MFA lockout",
	)
}

func (s *LockoutService) anyBlocked(
	ctx context.Context,
	keys ...string,
) (
	bool,
	error,
) {
	filtered :=
		make(
			[]string,
			0,
			len(
				keys,
			),
		)

	for _, key := range keys {

		if key != "" {
			filtered =
				append(
					filtered,
					key,
				)
		}
	}

	if len(filtered) == 0 {
		return false,
			nil
	}

	count, err :=
		s.redis.Exists(
			ctx,
			filtered...,
		).Result()
	if err != nil {
		return false,
			fmt.Errorf(
				"check Admin authentication lockout: %w",
				err,
			)
	}

	return count > 0,
		nil
}

func (s *LockoutService) clearDimension(
	ctx context.Context,
	counterKey string,
	blockKey string,
	operation string,
) error {
	if counterKey == "" ||
		blockKey == "" {

		return nil
	}

	if err :=
		s.redis.Del(
			ctx,
			counterKey,
			blockKey,
		).Err(); err != nil {

		return fmt.Errorf(
			"%s: %w",
			operation,
			err,
		)
	}

	return nil
}

func (s *LockoutService) recordDimensionFailure(
	ctx context.Context,
	counterKey string,
	blockKey string,
	window time.Duration,
	maxFailures int,
	blockTTL time.Duration,
) error {
	if counterKey == "" ||
		blockKey == "" {

		return nil
	}

	windowSeconds :=
		int64(
			window /
				time.Second,
		)

	blockSeconds :=
		int64(
			blockTTL /
				time.Second,
		)

	if windowSeconds < 1 {
		windowSeconds = 1
	}

	if blockSeconds < 1 {
		blockSeconds = 1
	}

	if _, err :=
		recordFailureScript.Run(
			ctx,
			s.redis,
			[]string{
				counterKey,
				blockKey,
			},
			strconv.FormatInt(
				windowSeconds,
				10,
			),
			strconv.Itoa(
				maxFailures,
			),
			strconv.FormatInt(
				blockSeconds,
				10,
			),
		).Result(); err != nil {

		return fmt.Errorf(
			"record Admin authentication failure: %w",
			err,
		)
	}

	return nil
}

func (s *LockoutService) identifierCounterKey(
	identifier string,
) string {
	value :=
		normalizeLockoutIdentifier(
			identifier,
		)

	if value == "" {
		return ""
	}

	return "admin:login:identifier:fail:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) identifierBlockKey(
	identifier string,
) string {
	value :=
		normalizeLockoutIdentifier(
			identifier,
		)

	if value == "" {
		return ""
	}

	return "admin:login:identifier:block:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) ipCounterKey(
	ipAddress string,
) string {
	value :=
		strings.TrimSpace(
			ipAddress,
		)

	if value == "" {
		return ""
	}

	return "admin:login:ip:fail:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) ipBlockKey(
	ipAddress string,
) string {
	value :=
		strings.TrimSpace(
			ipAddress,
		)

	if value == "" {
		return ""
	}

	return "admin:login:ip:block:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) mfaStaffCounterKey(
	staffAccountID string,
) string {
	value :=
		strings.TrimSpace(
			staffAccountID,
		)

	if value == "" {
		return ""
	}

	return "admin:mfa:staff:fail:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) mfaStaffBlockKey(
	staffAccountID string,
) string {
	value :=
		strings.TrimSpace(
			staffAccountID,
		)

	if value == "" {
		return ""
	}

	return "admin:mfa:staff:block:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) mfaIPCounterKey(
	ipAddress string,
) string {
	value :=
		strings.TrimSpace(
			ipAddress,
		)

	if value == "" {
		return ""
	}

	return "admin:mfa:ip:fail:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LockoutService) mfaIPBlockKey(
	ipAddress string,
) string {
	value :=
		strings.TrimSpace(
			ipAddress,
		)

	if value == "" {
		return ""
	}

	return "admin:mfa:ip:block:" +
		platformsecurity.SHA256String(
			value,
		)
}

func normalizeLockoutIdentifier(
	identifier string,
) string {
	return strings.ToLower(
		strings.TrimSpace(
			identifier,
		),
	)
}

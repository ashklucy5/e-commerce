package auth

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
	DefaultLoginMaxFailures = 8

	DefaultLoginFailureWindow = 15 * time.Minute

	DefaultLoginBlockTTL = 15 * time.Minute
)

type LoginLockoutConfig struct {
	MaxFailures int

	Window time.Duration

	BlockTTL time.Duration
}

type LoginLockoutService struct {
	redis *redis.Client

	config LoginLockoutConfig
}

var loginFailureScript = redis.NewScript(
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

func DefaultLoginLockoutConfig() LoginLockoutConfig {
	return LoginLockoutConfig{
		MaxFailures: DefaultLoginMaxFailures,

		Window: DefaultLoginFailureWindow,

		BlockTTL: DefaultLoginBlockTTL,
	}
}

func NewLoginLockoutService(
	client *redis.Client,
	cfg LoginLockoutConfig,
) (
	*LoginLockoutService,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"customer login lockout requires Redis",
			)
	}

	if cfg.MaxFailures <= 0 ||
		cfg.Window <= 0 ||
		cfg.BlockTTL <= 0 {

		return nil,
			fmt.Errorf(
				"invalid customer login lockout configuration",
			)
	}

	return &LoginLockoutService{
			redis: client,

			config: cfg,
		},
		nil
}

func (s *LoginLockoutService) IsBlocked(
	ctx context.Context,
	identifier string,
	ipAddress string,
) (
	bool,
	error,
) {
	keys :=
		make(
			[]string,
			0,
			2,
		)

	if key :=
		s.identifierBlockKey(
			identifier,
		); key != "" {

		keys =
			append(
				keys,
				key,
			)
	}

	if key :=
		s.ipBlockKey(
			ipAddress,
		); key != "" {

		keys =
			append(
				keys,
				key,
			)
	}

	if len(keys) == 0 {
		return false, nil
	}

	count, err :=
		s.redis.Exists(
			ctx,
			keys...,
		).Result()
	if err != nil {
		return false,
			fmt.Errorf(
				"check customer login lockout: %w",
				err,
			)
	}

	return count > 0,
		nil
}

func (s *LoginLockoutService) RecordFailure(
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
	)
}

func (s *LoginLockoutService) RecordSuccess(
	ctx context.Context,
	identifier string,
) error {
	counterKey :=
		s.identifierCounterKey(
			identifier,
		)

	blockKey :=
		s.identifierBlockKey(
			identifier,
		)

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
			"reset customer login lockout: %w",
			err,
		)
	}

	return nil
}

func (s *LoginLockoutService) recordDimensionFailure(
	ctx context.Context,
	counterKey string,
	blockKey string,
) error {
	if counterKey == "" ||
		blockKey == "" {

		return nil
	}

	windowSeconds :=
		int64(
			s.config.Window /
				time.Second,
		)

	blockSeconds :=
		int64(
			s.config.BlockTTL /
				time.Second,
		)

	if windowSeconds < 1 {
		windowSeconds = 1
	}

	if blockSeconds < 1 {
		blockSeconds = 1
	}

	if _, err :=
		loginFailureScript.Run(
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
				s.config.MaxFailures,
			),
			strconv.FormatInt(
				blockSeconds,
				10,
			),
		).Result(); err != nil {

		return fmt.Errorf(
			"record customer login failure: %w",
			err,
		)
	}

	return nil
}

func (s *LoginLockoutService) identifierCounterKey(
	identifier string,
) string {
	value :=
		normalizeLoginLockoutIdentifier(
			identifier,
		)

	if value == "" {
		return ""
	}

	return "customer:login:identifier:fail:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LoginLockoutService) identifierBlockKey(
	identifier string,
) string {
	value :=
		normalizeLoginLockoutIdentifier(
			identifier,
		)

	if value == "" {
		return ""
	}

	return "customer:login:identifier:block:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LoginLockoutService) ipCounterKey(
	ipAddress string,
) string {
	value :=
		strings.TrimSpace(
			ipAddress,
		)

	if value == "" {
		return ""
	}

	return "customer:login:ip:fail:" +
		platformsecurity.SHA256String(
			value,
		)
}

func (s *LoginLockoutService) ipBlockKey(
	ipAddress string,
) string {
	value :=
		strings.TrimSpace(
			ipAddress,
		)

	if value == "" {
		return ""
	}

	return "customer:login:ip:block:" +
		platformsecurity.SHA256String(
			value,
		)
}

func normalizeLoginLockoutIdentifier(
	value string,
) string {
	if phone, err :=
		normalizePhone(
			value,
		); err == nil {

		return phone
	}

	return strings.ToLower(
		strings.TrimSpace(
			value,
		),
	)
}

package jobruntime

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const runtimeTickLockKey = "commerce:runtime:tick:lock"

var releaseRuntimeTickLockScript = redis.NewScript(
	`
if redis.call(
  'GET',
  KEYS[1]
) == ARGV[1] then
  return redis.call(
    'DEL',
    KEYS[1]
  )
end

return 0
`,
)

type TickLock struct {
	redis *redis.Client

	key string

	ttl time.Duration
}

type TickLease struct {
	lock *TickLock

	token string

	released bool
}

func NewTickLock(
	client *redis.Client,
	ttl time.Duration,
) (*TickLock, error) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"runtime tick lock requires Redis",
			)
	}

	if ttl <= 0 {
		return nil,
			fmt.Errorf(
				"runtime tick lock TTL must be greater than zero",
			)
	}

	return &TickLock{
		redis: client,

		key: runtimeTickLockKey,

		ttl: ttl,
	}, nil
}

// Acquire attempts to acquire the runtime tick lock.
//
// acquired=false is not an error. It means another runtime tick is
// already active and the caller should exit without doing background work.
func (l *TickLock) Acquire(
	ctx context.Context,
) (
	lease *TickLease,
	acquired bool,
	err error,
) {
	if l == nil ||
		l.redis == nil {

		return nil,
			false,
			fmt.Errorf(
				"runtime tick lock is not configured",
			)
	}

	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	token, err :=
		platformsecurity.RandomURLSafe(
			24,
		)
	if err != nil {
		return nil,
			false,
			fmt.Errorf(
				"generate runtime tick lock token: %w",
				err,
			)
	}

	acquired, err =
		l.redis.SetNX(
			ctx,
			l.key,
			token,
			l.ttl,
		).Result()
	if err != nil {
		return nil,
			false,
			fmt.Errorf(
				"acquire runtime tick lock: %w",
				err,
			)
	}

	if !acquired {
		return nil, false, nil
	}

	return &TickLease{
			lock: l,

			token: token,
		},
		true,
		nil
}

// Release removes the lock only when Redis still contains this lease's
// unique token.
//
// This prevents an old invocation from deleting a lock that expired and
// was subsequently acquired by a newer invocation.
func (l *TickLease) Release(
	ctx context.Context,
) error {
	if l == nil ||
		l.lock == nil ||
		l.lock.redis == nil {

		return fmt.Errorf(
			"runtime tick lease is not configured",
		)
	}

	if l.released {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	_, err :=
		releaseRuntimeTickLockScript.Run(
			ctx,
			l.lock.redis,
			[]string{
				l.lock.key,
			},
			l.token,
		).Int64()
	if err != nil {
		return fmt.Errorf(
			"release runtime tick lock: %w",
			err,
		)
	}

	l.released =
		true

	return nil
}

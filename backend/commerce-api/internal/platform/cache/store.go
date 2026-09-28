package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	DefaultMaxValueBytes = 1 << 20

	DefaultLoadTimeout = 5 * time.Second
)

var ErrValueTooLarge = errors.New(
	"cache value exceeds maximum size",
)

type Store struct {
	redis *redis.Client

	group singleflight.Group

	maxValueBytes int

	loadTimeout time.Duration
}

func NewStore(
	client *redis.Client,
) (
	*Store,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"cache store requires Redis",
			)
	}

	return &Store{
			redis: client,

			maxValueBytes: DefaultMaxValueBytes,

			loadTimeout: DefaultLoadTimeout,
		},
		nil
}

func (s *Store) GetJSON(
	ctx context.Context,
	key string,
	destination any,
) (
	bool,
	error,
) {
	if s == nil ||
		s.redis == nil {

		return false, nil
	}

	key =
		strings.TrimSpace(
			key,
		)

	if key == "" {
		return false,
			fmt.Errorf(
				"cache key is required",
			)
	}

	raw, err :=
		s.redis.Get(
			ctx,
			key,
		).Bytes()

	if errors.Is(
		err,
		redis.Nil,
	) {
		return false, nil
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"cache get: %w",
				err,
			)
	}

	if len(raw) >
		s.maxValueBytes {

		_ =
			s.redis.Del(
				ctx,
				key,
			).Err()

		return false,
			fmt.Errorf(
				"%w: %d bytes",
				ErrValueTooLarge,
				len(raw),
			)
	}

	if err :=
		json.Unmarshal(
			raw,
			destination,
		); err != nil {

		// Corrupt cache data must never poison the application.
		//
		// Remove it and let the caller reload from the source
		// of truth.
		_ =
			s.redis.Del(
				ctx,
				key,
			).Err()

		return false,
			fmt.Errorf(
				"decode cached value: %w",
				err,
			)
	}

	return true, nil
}

func (s *Store) SetJSON(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) error {
	if s == nil ||
		s.redis == nil {

		return nil
	}

	key =
		strings.TrimSpace(
			key,
		)

	if key == "" {
		return fmt.Errorf(
			"cache key is required",
		)
	}

	if ttl <= 0 {
		return fmt.Errorf(
			"cache TTL must be greater than zero",
		)
	}

	raw, err :=
		json.Marshal(
			value,
		)
	if err != nil {
		return fmt.Errorf(
			"encode cached value: %w",
			err,
		)
	}

	if len(raw) >
		s.maxValueBytes {

		return fmt.Errorf(
			"%w: %d bytes",
			ErrValueTooLarge,
			len(raw),
		)
	}

	if err :=
		s.redis.Set(
			ctx,
			key,
			raw,
			TTLWithJitter(
				key,
				ttl,
			),
		).Err(); err != nil {

		return fmt.Errorf(
			"cache set: %w",
			err,
		)
	}

	return nil
}

func (s *Store) Delete(
	ctx context.Context,
	keys ...string,
) error {
	if s == nil ||
		s.redis == nil ||
		len(keys) == 0 {

		return nil
	}

	clean :=
		make(
			[]string,
			0,
			len(keys),
		)

	for _, key := range keys {

		key =
			strings.TrimSpace(
				key,
			)

		if key == "" {
			continue
		}

		clean =
			append(
				clean,
				key,
			)
	}

	if len(clean) == 0 {
		return nil
	}

	if err :=
		s.redis.Del(
			ctx,
			clean...,
		).Err(); err != nil {

		return fmt.Errorf(
			"delete cached value: %w",
			err,
		)
	}

	return nil
}

// RememberJSON implements cache-aside loading with stampede protection.
//
// Redis failures deliberately fail open:
// the authoritative loader still runs, so a cache outage does not turn
// into a storefront outage.
//
// Concurrent misses for the same key are collapsed into one loader call.
func RememberJSON[T any](
	ctx context.Context,
	store *Store,
	key string,
	ttl time.Duration,
	loader func(
		context.Context,
	) (
		T,
		error,
	),
) (
	T,
	error,
) {
	var zero T

	if loader == nil {
		return zero,
			fmt.Errorf(
				"cache loader is required",
			)
	}

	if store == nil {
		return loader(
			ctx,
		)
	}

	var cached T

	hit, err :=
		store.GetJSON(
			ctx,
			key,
			&cached,
		)

	if err == nil &&
		hit {

		return cached, nil
	}

	resultChannel :=
		store.group.DoChan(
			key,
			func() (
				any,
				error,
			) {
				// Another request may have filled the cache while this
				// caller was waiting to enter the singleflight section.
				var secondCached T

				hit, secondErr :=
					store.GetJSON(
						context.WithoutCancel(
							ctx,
						),
						key,
						&secondCached,
					)

				if secondErr == nil &&
					hit {

					return secondCached,
						nil
				}

				loadCtx, cancel :=
					context.WithTimeout(
						context.WithoutCancel(
							ctx,
						),
						store.loadTimeout,
					)

				defer cancel()

				loaded, loadErr :=
					loader(
						loadCtx,
					)

				if loadErr != nil {
					return zero,
						loadErr
				}

				// Cache writes are an optimization.
				//
				// The authoritative result must still succeed when Redis
				// is temporarily unavailable.
				_ =
					store.SetJSON(
						loadCtx,
						key,
						loaded,
						ttl,
					)

				return loaded,
					nil
			},
		)

	select {
	case result :=
		<-resultChannel:

		if result.Err != nil {
			return zero,
				result.Err
		}

		value, ok :=
			result.Val.(T)

		if !ok {
			return zero,
				fmt.Errorf(
					"cached loader returned unexpected type",
				)
		}

		return value, nil

	case <-ctx.Done():
		return zero,
			ctx.Err()
	}
}

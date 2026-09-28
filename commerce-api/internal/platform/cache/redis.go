package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"project.local/commerce-api/internal/platform/config"
)

var errInvalidRedisURL =
	errors.New(
		"invalid REDIS_URL",
	)

func NewRedis(
	ctx context.Context,
	cfg config.Config,
) (*redis.Client, error) {
	options, err :=
		redisOptions(
			cfg,
		)
	if err != nil {
		return nil, err
	}

	client :=
		redis.NewClient(
			options,
		)

	pingCtx, cancel :=
		context.WithTimeout(
			ctx,
			5*time.Second,
		)

	defer cancel()

	if err :=
		client.Ping(
			pingCtx,
		).Err(); err != nil {

		_ =
			client.Close()

		return nil,
			errors.New(
				"redis ping failed",
			)
	}

	return client, nil
}

func redisOptions(
	cfg config.Config,
) (*redis.Options, error) {
	rawURL :=
		strings.TrimSpace(
			cfg.RedisURL,
		)

	var options *redis.Options

	if rawURL != "" {
		parsed, err :=
			redis.ParseURL(
				rawURL,
			)
		if err != nil {
			/*
				Do not wrap the parser error.

				REDIS_URL commonly contains the Redis password.
				A parser error must never risk echoing the URL
				into application logs.
			*/
			return nil,
				errInvalidRedisURL
		}

		options =
			parsed
	} else {
		options =
			&redis.Options{
				Addr:
					strings.TrimSpace(
						cfg.RedisAddr,
					),

				Password:
					cfg.RedisPassword,

				DB:
					cfg.RedisDB,
			}
	}

	/*
		Keep the existing connection behavior identical for now.

		Batch 14 will tune the pool and queue settings specifically
		for managed/serverless Redis after command compatibility is
		verified.
	*/
	options.DialTimeout =
		5 * time.Second

	options.ReadTimeout =
		3 * time.Second

	options.WriteTimeout =
		3 * time.Second

	options.PoolSize =
		20

	return options, nil
}
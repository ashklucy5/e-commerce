package cache

import (
	"errors"
	"strings"
	"testing"
	"time"

	"project.local/commerce-api/internal/platform/config"
)

func TestRedisOptionsLegacyConfiguration(
	t *testing.T,
) {
	t.Parallel()

	options, err :=
		redisOptions(
			config.Config{
				RedisAddr: "localhost:6379",

				RedisPassword: "local-password",

				RedisDB: 3,
			},
		)
	if err != nil {
		t.Fatalf(
			"redis options: %v",
			err,
		)
	}

	if options.Addr !=
		"localhost:6379" {

		t.Fatalf(
			"addr = %q, want localhost:6379",
			options.Addr,
		)
	}

	if options.Password !=
		"local-password" {

		t.Fatal(
			"legacy Redis password was not preserved",
		)
	}

	if options.DB !=
		3 {

		t.Fatalf(
			"db = %d, want 3",
			options.DB,
		)
	}

	if options.TLSConfig !=
		nil {

		t.Fatal(
			"legacy redis configuration unexpectedly enabled TLS",
		)
	}

	assertRedisRuntimeOptions(
		t,
		options.DialTimeout,
		options.ReadTimeout,
		options.WriteTimeout,
		options.PoolSize,
	)
}

func TestRedisOptionsRedisURL(
	t *testing.T,
) {
	t.Parallel()

	options, err :=
		redisOptions(
			config.Config{
				RedisURL: "redis://default:test-password@redis.example.test:6379/4",
			},
		)
	if err != nil {
		t.Fatalf(
			"redis URL options: %v",
			err,
		)
	}

	if options.Addr !=
		"redis.example.test:6379" {

		t.Fatalf(
			"addr = %q, want redis.example.test:6379",
			options.Addr,
		)
	}

	if options.Username !=
		"default" {

		t.Fatalf(
			"username = %q, want default",
			options.Username,
		)
	}

	if options.Password !=
		"test-password" {

		t.Fatal(
			"Redis URL password was not parsed",
		)
	}

	if options.DB !=
		4 {

		t.Fatalf(
			"db = %d, want 4",
			options.DB,
		)
	}

	if options.TLSConfig !=
		nil {

		t.Fatal(
			"redis:// unexpectedly enabled TLS",
		)
	}
}

func TestRedisOptionsRedissEnablesTLS(
	t *testing.T,
) {
	t.Parallel()

	options, err :=
		redisOptions(
			config.Config{
				RedisURL: "rediss://default:test-password@managed-redis.example.test:6379/0",
			},
		)
	if err != nil {
		t.Fatalf(
			"rediss options: %v",
			err,
		)
	}

	if options.Addr !=
		"managed-redis.example.test:6379" {

		t.Fatalf(
			"addr = %q",
			options.Addr,
		)
	}

	if options.TLSConfig ==
		nil {

		t.Fatal(
			"rediss:// must enable TLS",
		)
	}

	assertRedisRuntimeOptions(
		t,
		options.DialTimeout,
		options.ReadTimeout,
		options.WriteTimeout,
		options.PoolSize,
	)
}

func TestRedisURLTakesPrecedence(
	t *testing.T,
) {
	t.Parallel()

	options, err :=
		redisOptions(
			config.Config{
				RedisURL: "rediss://default:url-password@managed.example.test:6379/2",

				RedisAddr: "legacy.example.test:6380",

				RedisPassword: "legacy-password",

				RedisDB: 9,
			},
		)
	if err != nil {
		t.Fatalf(
			"redis options: %v",
			err,
		)
	}

	if options.Addr !=
		"managed.example.test:6379" {

		t.Fatalf(
			"URL did not take precedence: addr=%q",
			options.Addr,
		)
	}

	if options.Password !=
		"url-password" {

		t.Fatal(
			"URL password did not take precedence",
		)
	}

	if options.DB !=
		2 {

		t.Fatalf(
			"URL database did not take precedence: %d",
			options.DB,
		)
	}

	if options.TLSConfig ==
		nil {

		t.Fatal(
			"URL TLS configuration was not preserved",
		)
	}
}

func TestRedisOptionsRejectsInvalidURLWithoutLeakingValue(
	t *testing.T,
) {
	t.Parallel()

	const secret = "do-not-leak-this-secret"

	_, err :=
		redisOptions(
			config.Config{
				RedisURL: "not-a-redis-url://" +
					secret,
			},
		)

	if !errors.Is(
		err,
		errInvalidRedisURL,
	) {
		t.Fatalf(
			"error = %v, want errInvalidRedisURL",
			err,
		)
	}

	if err != nil &&
		strings.Contains(
			err.Error(),
			secret,
		) {

		t.Fatal(
			"Redis URL error leaked secret material",
		)
	}
}

func assertRedisRuntimeOptions(
	t *testing.T,
	dialTimeout time.Duration,
	readTimeout time.Duration,
	writeTimeout time.Duration,
	poolSize int,
) {
	t.Helper()

	if dialTimeout !=
		5*time.Second {

		t.Fatalf(
			"dial timeout = %s, want 5s",
			dialTimeout,
		)
	}

	if readTimeout !=
		3*time.Second {

		t.Fatalf(
			"read timeout = %s, want 3s",
			readTimeout,
		)
	}

	if writeTimeout !=
		3*time.Second {

		t.Fatalf(
			"write timeout = %s, want 3s",
			writeTimeout,
		)
	}

	if poolSize !=
		20 {

		t.Fatalf(
			"pool size = %d, want 20",
			poolSize,
		)
	}
}

// func contains(
// 	value string,
// 	substring string,
// ) bool {
// 	for index :=
// 		0; index+
// 		len(
// 			substring,
// 		) <=
// 		len(
// 			value,
// 		); index++ {

// 		if value[
// 			index:index+
// 				len(
// 					substring,
// 				)
// 		] ==
// 			substring {

// 			return true
// 		}
// 	}

// 	return false
// }

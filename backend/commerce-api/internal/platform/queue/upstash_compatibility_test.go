package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	platformcache "project.local/commerce-api/internal/platform/cache"
	platformconfig "project.local/commerce-api/internal/platform/config"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const runUpstashRedisCompatibilityTestEnv = "RUN_UPSTASH_REDIS_COMPAT_TEST"

func TestUpstashRedisCompatibility(
	t *testing.T,
) {
	if strings.TrimSpace(
		os.Getenv(
			runUpstashRedisCompatibilityTestEnv,
		),
	) != "1" {

		t.Skipf(
			"set %s=1 to run the real Upstash Redis compatibility test",
			runUpstashRedisCompatibilityTestEnv,
		)
	}

	/*
		Production Redis uses APP_REDIS_URL only.

		Local/Docker Redis continues to use:

			REDIS_ADDR
			REDIS_PASSWORD
			REDIS_DB

		This integration test intentionally targets the production
		managed Redis connection and therefore requires APP_REDIS_URL.
	*/

	redisURL :=
		strings.TrimSpace(
			os.Getenv(
				"APP_REDIS_URL",
			),
		)

	if redisURL == "" {
		t.Fatal(
			"APP_REDIS_URL is required for the real Upstash compatibility test",
		)
	}

	if !strings.HasPrefix(
		strings.ToLower(
			redisURL,
		),
		"rediss://",
	) {
		t.Fatal(
			"real Upstash compatibility test requires a rediss:// APP_REDIS_URL",
		)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	client, err :=
		platformcache.NewRedis(
			ctx,
			platformconfig.Config{
				RedisURL: redisURL,
			},
		)
	if err != nil {
		t.Fatalf(
			"connect to managed Redis: %v",
			err,
		)
	}

	defer func() {
		_ =
			client.Close()
	}()

	prefix :=
		fmt.Sprintf(
			"ene:compat:%d",
			time.Now().
				UTC().
				UnixNano(),
		)

	stream :=
		prefix +
			":stream"

	group :=
		prefix +
			":group"

	retrySet :=
		prefix +
			":retry"

	deadLetterStream :=
		prefix +
			":dead"

	cacheKey :=
		prefix +
			":cache"

	lockKey :=
		prefix +
			":lock"

	txKeyA :=
		prefix +
			":tx:a"

	txKeyB :=
		prefix +
			":tx:b"

	dedupeName :=
		"upstash-compatibility-dedupe"

	dedupeRedisKey :=
		stream +
			":dedupe:" +
			platformsecurity.SHA256String(
				dedupeName,
			)

	cleanupKeys :=
		[]string{
			stream,
			retrySet,
			deadLetterStream,
			cacheKey,
			lockKey,
			txKeyA,
			txKeyB,
			dedupeRedisKey,
		}

	defer func() {
		cleanupCtx,
			cleanupCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)

		defer cleanupCancel()

		if err :=
			client.Del(
				cleanupCtx,
				cleanupKeys...,
			).Err(); err != nil {

			t.Logf(
				"managed Redis compatibility cleanup warning: %v",
				err,
			)
		}
	}()

	t.Run(
		"ping",
		func(
			t *testing.T,
		) {
			if err :=
				client.Ping(
					ctx,
				).Err(); err != nil {

				t.Fatalf(
					"PING failed: %v",
					err,
				)
			}
		},
	)

	t.Run(
		"string-cache",
		func(
			t *testing.T,
		) {
			if err :=
				client.Set(
					ctx,
					cacheKey,
					"compatibility-value",
					time.Minute,
				).Err(); err != nil {

				t.Fatalf(
					"SET failed: %v",
					err,
				)
			}

			value, err :=
				client.Get(
					ctx,
					cacheKey,
				).Result()
			if err != nil {
				t.Fatalf(
					"GET failed: %v",
					err,
				)
			}

			if value !=
				"compatibility-value" {

				t.Fatalf(
					"GET value = %q",
					value,
				)
			}

			exists, err :=
				client.Exists(
					ctx,
					cacheKey,
				).Result()
			if err != nil {
				t.Fatalf(
					"EXISTS failed: %v",
					err,
				)
			}

			if exists !=
				1 {

				t.Fatalf(
					"EXISTS = %d, want 1",
					exists,
				)
			}

			if err :=
				client.Del(
					ctx,
					cacheKey,
				).Err(); err != nil {

				t.Fatalf(
					"DEL failed: %v",
					err,
				)
			}

			_, err =
				client.Get(
					ctx,
					cacheKey,
				).Result()

			if !errors.Is(
				err,
				redis.Nil,
			) {
				t.Fatalf(
					"GET after DEL error = %v, want redis.Nil",
					err,
				)
			}
		},
	)

	t.Run(
		"setnx-and-lua-lock-release",
		func(
			t *testing.T,
		) {
			const token = "compatibility-lock-token"

			acquired, err :=
				client.SetNX(
					ctx,
					lockKey,
					token,
					time.Minute,
				).Result()
			if err != nil {
				t.Fatalf(
					"SETNX failed: %v",
					err,
				)
			}

			if !acquired {
				t.Fatal(
					"SETNX did not acquire a fresh lock",
				)
			}

			acquiredAgain, err :=
				client.SetNX(
					ctx,
					lockKey,
					"other-token",
					time.Minute,
				).Result()
			if err != nil {
				t.Fatalf(
					"second SETNX failed: %v",
					err,
				)
			}

			if acquiredAgain {
				t.Fatal(
					"second SETNX unexpectedly acquired existing lock",
				)
			}

			releaseScript :=
				redis.NewScript(
					`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`,
				)

			released, err :=
				releaseScript.Run(
					ctx,
					client,
					[]string{
						lockKey,
					},
					token,
				).Int64()
			if err != nil {
				t.Fatalf(
					"Lua lock release failed: %v",
					err,
				)
			}

			if released !=
				1 {

				t.Fatalf(
					"Lua lock release = %d, want 1",
					released,
				)
			}
		},
	)

	t.Run(
		"transaction",
		func(
			t *testing.T,
		) {
			pipe :=
				client.TxPipeline()

			pipe.Set(
				ctx,
				txKeyA,
				"a",
				time.Minute,
			)

			pipe.Set(
				ctx,
				txKeyB,
				"b",
				time.Minute,
			)

			pipe.Del(
				ctx,
				txKeyA,
			)

			if _, err :=
				pipe.Exec(
					ctx,
				); err != nil {

				t.Fatalf(
					"MULTI/EXEC transaction failed: %v",
					err,
				)
			}

			_, err :=
				client.Get(
					ctx,
					txKeyA,
				).Result()

			if !errors.Is(
				err,
				redis.Nil,
			) {
				t.Fatalf(
					"transaction key A still exists: %v",
					err,
				)
			}

			value, err :=
				client.Get(
					ctx,
					txKeyB,
				).Result()
			if err != nil {
				t.Fatalf(
					"load transaction key B: %v",
					err,
				)
			}

			if value !=
				"b" {

				t.Fatalf(
					"transaction key B = %q, want b",
					value,
				)
			}
		},
	)

	queueConfig :=
		Config{
			Stream: stream,

			Group: group,

			RetrySet: retrySet,

			DeadLetterStream: deadLetterStream,

			StreamMaxLen: 0,

			DeadLetterMaxLen: 100,

			DefaultAttempts: 3,

			RetryBaseDelay: 10 * time.Millisecond,

			RetryMaxDelay: 10 * time.Millisecond,
		}

	producer, err :=
		NewProducer(
			client,
			queueConfig,
		)
	if err != nil {
		t.Fatalf(
			"create compatibility queue producer: %v",
			err,
		)
	}

	var original Message

	t.Run(
		"queue-deduplicated-enqueue",
		func(
			t *testing.T,
		) {
			message,
				enqueued,
				err :=
				producer.Enqueue(
					ctx,
					"compatibility.probe",
					map[string]any{
						"source": "upstash-compatibility",
					},
					EnqueueOptions{
						MaxAttempts: 3,

						DedupeKey: dedupeName,

						DedupeTTL: time.Minute,
					},
				)
			if err != nil {
				t.Fatalf(
					"deduplicated enqueue failed: %v",
					err,
				)
			}

			if !enqueued {
				t.Fatal(
					"first deduplicated enqueue was rejected",
				)
			}

			if message.StreamID ==
				"" {

				t.Fatal(
					"first enqueue returned empty stream ID",
				)
			}

			original =
				message

			_,
				enqueued,
				err =
				producer.Enqueue(
					ctx,
					"compatibility.probe",
					map[string]any{
						"source": "duplicate",
					},
					EnqueueOptions{
						MaxAttempts: 3,

						DedupeKey: dedupeName,

						DedupeTTL: time.Minute,
					},
				)
			if err != nil {
				t.Fatalf(
					"duplicate enqueue failed: %v",
					err,
				)
			}

			if enqueued {
				t.Fatal(
					"duplicate dedupe key unexpectedly created another job",
				)
			}
		},
	)

	t.Run(
		"stream-consumer-group",
		func(
			t *testing.T,
		) {
			if err :=
				client.XGroupCreateMkStream(
					ctx,
					stream,
					group,
					"0",
				).Err(); err != nil {

				t.Fatalf(
					"XGROUP CREATE MKSTREAM failed: %v",
					err,
				)
			}

			streams, err :=
				client.XReadGroup(
					ctx,
					&redis.XReadGroupArgs{
						Group: group,

						Consumer: "compat-consumer-a",

						Streams: []string{
							stream,
							">",
						},

						Count: 1,

						Block: 2 * time.Second,
					},
				).Result()
			if err != nil {
				t.Fatalf(
					"XREADGROUP BLOCK failed: %v",
					err,
				)
			}

			if len(
				streams,
			) !=
				1 ||
				len(
					streams[0].
						Messages,
				) !=
					1 {

				t.Fatalf(
					"XREADGROUP returned unexpected response: %#v",
					streams,
				)
			}

			if streams[0].
				Messages[0].
				ID !=
				original.StreamID {

				t.Fatalf(
					"stream ID = %q, want %q",
					streams[0].
						Messages[0].
						ID,
					original.StreamID,
				)
			}

			pending, err :=
				client.XPendingExt(
					ctx,
					&redis.XPendingExtArgs{
						Stream: stream,

						Group: group,

						Start: "-",

						End: "+",

						Count: 10,
					},
				).Result()
			if err != nil {
				t.Fatalf(
					"XPENDING failed: %v",
					err,
				)
			}

			if len(
				pending,
			) !=
				1 {

				t.Fatalf(
					"XPENDING count = %d, want 1",
					len(
						pending,
					),
				)
			}

			time.Sleep(
				10 * time.Millisecond,
			)

			claimed, err :=
				client.XClaim(
					ctx,
					&redis.XClaimArgs{
						Stream: stream,

						Group: group,

						Consumer: "compat-consumer-b",

						MinIdle: time.Millisecond,

						Messages: []string{
							original.StreamID,
						},
					},
				).Result()
			if err != nil {
				t.Fatalf(
					"XCLAIM failed: %v",
					err,
				)
			}

			if len(
				claimed,
			) !=
				1 {

				t.Fatalf(
					"XCLAIM count = %d, want 1",
					len(
						claimed,
					),
				)
			}

			acknowledged, err :=
				ackDeleteScript.Run(
					ctx,
					client,
					[]string{
						stream,
					},
					group,
					original.StreamID,
				).Int64()
			if err != nil {
				t.Fatalf(
					"XACK/XDEL Lua script failed: %v",
					err,
				)
			}

			if acknowledged !=
				1 {

				t.Fatalf(
					"acknowledged = %d, want 1",
					acknowledged,
				)
			}
		},
	)

	t.Run(
		"blocking-read-empty-stream",
		func(
			t *testing.T,
		) {
			readCtx,
				readCancel :=
				context.WithTimeout(
					ctx,
					3*time.Second,
				)

			defer readCancel()

			_,
				err :=
				client.XReadGroup(
					readCtx,
					&redis.XReadGroupArgs{
						Group: group,

						Consumer: "compat-consumer-a",

						Streams: []string{
							stream,
							">",
						},

						Count: 1,

						Block: 250 * time.Millisecond,
					},
				).Result()

			if err != nil &&
				!errors.Is(
					err,
					redis.Nil,
				) {

				t.Fatalf(
					"empty XREADGROUP BLOCK failed: %v",
					err,
				)
			}
		},
	)

	t.Run(
		"sorted-set-retry-promotion",
		func(
			t *testing.T,
		) {
			if err :=
				producer.ScheduleRetry(
					ctx,
					original,
				); err != nil {

				t.Fatalf(
					"ZADD retry scheduling failed: %v",
					err,
				)
			}

			time.Sleep(
				30 * time.Millisecond,
			)

			moved, err :=
				producer.PromoteDueRetries(
					ctx,
					10,
				)
			if err != nil {
				t.Fatalf(
					"retry promotion Lua script failed: %v",
					err,
				)
			}

			if moved !=
				1 {

				t.Fatalf(
					"promoted retries = %d, want 1",
					moved,
				)
			}

			remaining, err :=
				client.ZCard(
					ctx,
					retrySet,
				).Result()
			if err != nil {
				t.Fatalf(
					"ZCARD retry set failed: %v",
					err,
				)
			}

			if remaining !=
				0 {

				t.Fatalf(
					"retry sorted set contains %d items after promotion",
					remaining,
				)
			}

			streams, err :=
				client.XReadGroup(
					ctx,
					&redis.XReadGroupArgs{
						Group: group,

						Consumer: "compat-consumer-b",

						Streams: []string{
							stream,
							">",
						},

						Count: 1,

						Block: 2 * time.Second,
					},
				).Result()
			if err != nil {
				t.Fatalf(
					"read promoted retry: %v",
					err,
				)
			}

			if len(
				streams,
			) !=
				1 ||
				len(
					streams[0].
						Messages,
				) !=
					1 {

				t.Fatal(
					"promoted retry was not written to stream",
				)
			}

			retryStreamID :=
				streams[0].
					Messages[0].
					ID

			acknowledged, err :=
				ackDeleteScript.Run(
					ctx,
					client,
					[]string{
						stream,
					},
					group,
					retryStreamID,
				).Int64()
			if err != nil {
				t.Fatalf(
					"ack promoted retry: %v",
					err,
				)
			}

			if acknowledged !=
				1 {

				t.Fatalf(
					"promoted retry acknowledged = %d, want 1",
					acknowledged,
				)
			}
		},
	)

	t.Run(
		"dead-letter-stream",
		func(
			t *testing.T,
		) {
			if err :=
				producer.DeadLetter(
					ctx,
					original,
					errors.New(
						"compatibility probe",
					),
				); err != nil {

				t.Fatalf(
					"dead-letter XADD failed: %v",
					err,
				)
			}

			length, err :=
				client.XLen(
					ctx,
					deadLetterStream,
				).Result()
			if err != nil {
				t.Fatalf(
					"XLEN dead-letter stream failed: %v",
					err,
				)
			}

			if length !=
				1 {

				t.Fatalf(
					"dead-letter stream length = %d, want 1",
					length,
				)
			}
		},
	)

	t.Log(
		"Upstash compatibility passed: TLS, strings, SETNX, Lua, MULTI/EXEC, XADD, XGROUP, XREADGROUP BLOCK, XPENDING, XCLAIM, XACK/XDEL, sorted-set retry promotion, and dead-letter streams",
	)
}

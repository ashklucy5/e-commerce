package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

type EnqueueOptions struct {
	MaxAttempts int

	DedupeKey string
	DedupeTTL time.Duration
}

type Producer struct {
	redis *redis.Client

	config Config
}

var uniqueEnqueueScript = redis.NewScript(
	`
if redis.call('EXISTS', KEYS[1]) == 1 then
  return 0
end

local stream_id

if tonumber(ARGV[2]) > 0 then
  stream_id = redis.call(
    'XADD', KEYS[2],
    'MAXLEN', '~', ARGV[2],
    '*',
    'job_id', ARGV[3],
    'job_type', ARGV[4],
    'payload', ARGV[5],
    'attempt', ARGV[6],
    'max_attempts', ARGV[7],
    'created_at', ARGV[8]
  )
else
  stream_id = redis.call(
    'XADD', KEYS[2],
    '*',
    'job_id', ARGV[3],
    'job_type', ARGV[4],
    'payload', ARGV[5],
    'attempt', ARGV[6],
    'max_attempts', ARGV[7],
    'created_at', ARGV[8]
  )
end

redis.call(
  'SET',
  KEYS[1],
  '1',
  'PX',
  ARGV[1]
)

return stream_id
`,
)

var promoteRetryScript = redis.NewScript(
	`
local items = redis.call(
  'ZRANGEBYSCORE',
  KEYS[1],
  '-inf',
  ARGV[1],
  'LIMIT', 0, ARGV[2]
)

for _, raw in ipairs(items) do
  local item = cjson.decode(raw)

  if tonumber(ARGV[3]) > 0 then
    redis.call(
      'XADD', KEYS[2],
      'MAXLEN', '~', ARGV[3],
      '*',
      'job_id', item.job_id,
      'job_type', item.job_type,
      'payload', item.payload,
      'attempt', item.attempt,
      'max_attempts', item.max_attempts,
      'created_at', item.created_at
    )
  else
    redis.call(
      'XADD', KEYS[2],
      '*',
      'job_id', item.job_id,
      'job_type', item.job_type,
      'payload', item.payload,
      'attempt', item.attempt,
      'max_attempts', item.max_attempts,
      'created_at', item.created_at
    )
  end

  redis.call(
    'ZREM',
    KEYS[1],
    raw
  )
end

return #items
`,
)

type retryEnvelope struct {
	JobID string `json:"job_id"`

	JobType string `json:"job_type"`

	Payload string `json:"payload"`

	Attempt string `json:"attempt"`

	MaxAttempts string `json:"max_attempts"`

	CreatedAt string `json:"created_at"`
}

func NewProducer(
	client *redis.Client,
	cfg Config,
) (
	*Producer,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"queue producer requires Redis",
			)
	}

	cfg =
		normalizeConfig(
			cfg,
		)

	return &Producer{
			redis: client,

			config: cfg,
		},
		nil
}

func (p *Producer) Enqueue(
	ctx context.Context,
	jobType string,
	payload any,
	options EnqueueOptions,
) (
	Message,
	bool,
	error,
) {
	message, err :=
		p.newMessage(
			jobType,
			payload,
			options.MaxAttempts,
		)
	if err != nil {
		return Message{},
			false,
			err
	}

	dedupeKey :=
		strings.TrimSpace(
			options.DedupeKey,
		)

	if dedupeKey == "" {
		args :=
			&redis.XAddArgs{
				Stream: p.config.Stream,

				Values: messageValues(
					message,
				),
			}

		if p.config.StreamMaxLen >
			0 {

			args.MaxLen =
				p.config.StreamMaxLen

			args.Approx =
				true
		}

		streamID, err :=
			p.redis.XAdd(
				ctx,
				args,
			).Result()
		if err != nil {
			return Message{},
				false,
				fmt.Errorf(
					"enqueue job: %w",
					err,
				)
		}

		message.StreamID =
			streamID

		return message,
			true,
			nil
	}

	if options.DedupeTTL <= 0 {
		return Message{},
			false,
			fmt.Errorf(
				"queue dedupe TTL must be greater than zero",
			)
	}

	dedupeRedisKey :=
		p.config.Stream +
			":dedupe:" +
			platformsecurity.SHA256String(
				dedupeKey,
			)

	result, err :=
		uniqueEnqueueScript.Run(
			ctx,
			p.redis,
			[]string{
				dedupeRedisKey,
				p.config.Stream,
			},
			strconv.FormatInt(
				options.DedupeTTL.Milliseconds(),
				10,
			),
			strconv.FormatInt(
				p.config.StreamMaxLen,
				10,
			),
			message.ID,
			message.Type,
			string(
				message.Payload,
			),
			strconv.Itoa(
				message.Attempt,
			),
			strconv.Itoa(
				message.MaxAttempts,
			),
			strconv.FormatInt(
				message.CreatedAt.UnixMilli(),
				10,
			),
		).Result()
	if err != nil {
		return Message{},
			false,
			fmt.Errorf(
				"enqueue deduplicated job: %w",
				err,
			)
	}

	switch value :=
		result.(type) {

	case string:
		message.StreamID =
			value

		return message,
			true,
			nil

	case int64:
		if value == 0 {
			return Message{},
				false,
				nil
		}
	}

	return Message{},
		false,
		fmt.Errorf(
			"enqueue deduplicated job returned unexpected Redis result %T",
			result,
		)
}

func (p *Producer) ScheduleRetry(
	ctx context.Context,
	message Message,
) error {
	if message.Attempt >=
		message.MaxAttempts {

		return fmt.Errorf(
			"job %s has no retry attempts remaining",
			message.ID,
		)
	}

	nextAttempt :=
		message.Attempt +
			1

	envelope :=
		retryEnvelope{
			JobID: message.ID,

			JobType: message.Type,

			Payload: string(
				message.Payload,
			),

			Attempt: strconv.Itoa(
				nextAttempt,
			),

			MaxAttempts: strconv.Itoa(
				message.MaxAttempts,
			),

			CreatedAt: strconv.FormatInt(
				message.CreatedAt.UnixMilli(),
				10,
			),
		}

	raw, err :=
		json.Marshal(
			envelope,
		)
	if err != nil {
		return fmt.Errorf(
			"encode retry job: %w",
			err,
		)
	}

	dueAt :=
		time.Now().
			UTC().
			Add(
				p.retryDelay(
					message.ID,
					nextAttempt,
				),
			)

	if err :=
		p.redis.ZAdd(
			ctx,
			p.config.RetrySet,
			redis.Z{
				Score: float64(
					dueAt.UnixMilli(),
				),

				Member: string(
					raw,
				),
			},
		).Err(); err != nil {

		return fmt.Errorf(
			"schedule job retry: %w",
			err,
		)
	}

	return nil
}

func (p *Producer) PromoteDueRetries(
	ctx context.Context,
	limit int64,
) (
	int64,
	error,
) {
	if limit <= 0 {
		limit = 100
	}

	if limit > 1000 {
		limit = 1000
	}

	moved, err :=
		promoteRetryScript.Run(
			ctx,
			p.redis,
			[]string{
				p.config.RetrySet,
				p.config.Stream,
			},
			strconv.FormatInt(
				time.Now().
					UTC().
					UnixMilli(),
				10,
			),
			strconv.FormatInt(
				limit,
				10,
			),
			strconv.FormatInt(
				p.config.StreamMaxLen,
				10,
			),
		).Int64()
	if err != nil {
		return 0,
			fmt.Errorf(
				"promote due retries: %w",
				err,
			)
	}

	return moved, nil
}

func (p *Producer) DeadLetter(
	ctx context.Context,
	message Message,
	cause error,
) error {
	values :=
		map[string]interface{}{
			"job_id": message.ID,

			"job_type": message.Type,

			"payload": string(
				message.Payload,
			),

			"attempt": message.Attempt,

			"max_attempts": message.MaxAttempts,

			"created_at": message.CreatedAt.UnixMilli(),

			"source_stream_id": message.StreamID,

			"failed_at": time.Now().
				UTC().
				UnixMilli(),

			"error": boundedError(
				cause,
			),
		}

	if _, err :=
		p.redis.XAdd(
			ctx,
			&redis.XAddArgs{
				Stream: p.config.
					DeadLetterStream,

				MaxLen: p.config.
					DeadLetterMaxLen,

				Approx: true,

				Values: values,
			},
		).Result(); err != nil {

		return fmt.Errorf(
			"dead-letter job: %w",
			err,
		)
	}

	return nil
}

func (p *Producer) DeadLetterRaw(
	ctx context.Context,
	streamID string,
	values map[string]interface{},
	cause error,
) error {
	raw, err :=
		json.Marshal(
			values,
		)
	if err != nil {
		raw =
			[]byte(
				`{"unserializable":true}`,
			)
	}

	if len(raw) >
		MaxPayloadBytes {

		raw =
			raw[:MaxPayloadBytes]
	}

	now :=
		time.Now().
			UTC().
			UnixMilli()

	if _, err :=
		p.redis.XAdd(
			ctx,
			&redis.XAddArgs{
				Stream: p.config.
					DeadLetterStream,

				MaxLen: p.config.
					DeadLetterMaxLen,

				Approx: true,

				Values: map[string]interface{}{
					"job_id": "invalid-" +
						streamID,

					"job_type": "invalid_message",

					"payload": string(
						raw,
					),

					"attempt": 1,

					"max_attempts": 1,

					"created_at": now,

					"source_stream_id": streamID,

					"failed_at": now,

					"error": boundedError(
						cause,
					),
				},
			},
		).Result(); err != nil {

		return fmt.Errorf(
			"dead-letter invalid job: %w",
			err,
		)
	}

	return nil
}

func (p *Producer) newMessage(
	jobType string,
	payload any,
	maxAttempts int,
) (
	Message,
	error,
) {
	jobType =
		normalizeJobType(
			jobType,
		)

	if !validJobType(
		jobType,
	) {
		return Message{},
			fmt.Errorf(
				"invalid queue job type %q",
				jobType,
			)
	}

	body :=
		[]byte(
			`{}`,
		)

	if payload != nil {
		encoded, err :=
			json.Marshal(
				payload,
			)
		if err != nil {
			return Message{},
				fmt.Errorf(
					"encode queue payload: %w",
					err,
				)
		}

		body =
			encoded
	}

	if len(body) >
		MaxPayloadBytes {

		return Message{},
			fmt.Errorf(
				"queue payload exceeds %d bytes",
				MaxPayloadBytes,
			)
	}

	if maxAttempts <= 0 {
		maxAttempts =
			p.config.
				DefaultAttempts
	}

	if maxAttempts > 100 {
		return Message{},
			fmt.Errorf(
				"queue max attempts cannot exceed 100",
			)
	}

	jobID, err :=
		platformsecurity.RandomURLSafe(
			18,
		)
	if err != nil {
		return Message{},
			fmt.Errorf(
				"generate queue job ID: %w",
				err,
			)
	}

	return Message{
			ID: jobID,

			Type: jobType,

			Payload: json.RawMessage(
				body,
			),

			Attempt: 1,

			MaxAttempts: maxAttempts,

			CreatedAt: time.Now().
				UTC(),
		},
		nil
}

func messageValues(
	message Message,
) map[string]interface{} {
	return map[string]interface{}{
		"job_id": message.ID,

		"job_type": message.Type,

		"payload": string(
			message.Payload,
		),

		"attempt": message.Attempt,

		"max_attempts": message.MaxAttempts,

		"created_at": message.CreatedAt.UnixMilli(),
	}
}

func (p *Producer) retryDelay(
	jobID string,
	nextAttempt int,
) time.Duration {
	delay :=
		p.config.
			RetryBaseDelay

	for attempt :=
		2; attempt < nextAttempt; attempt++ {

		if delay >=
			p.config.
				RetryMaxDelay/2 {

			delay =
				p.config.
					RetryMaxDelay

			break
		}

		delay *=
			2
	}

	if delay >
		p.config.
			RetryMaxDelay {

		delay =
			p.config.
				RetryMaxDelay
	}

	jitterWindow :=
		delay / 5

	if jitterWindow <= 0 {
		return delay
	}

	hash :=
		fnv.New64a()

	_, _ =
		hash.Write(
			[]byte(
				jobID,
			),
		)

	_, _ =
		hash.Write(
			[]byte{
				':',
			},
		)

	_, _ =
		hash.Write(
			[]byte(
				strconv.Itoa(
					nextAttempt,
				),
			),
		)

	jitter :=
		time.Duration(
			hash.Sum64() %
				uint64(
					jitterWindow+1,
				),
		)

	if delay+jitter >
		p.config.
			RetryMaxDelay {

		return p.config.
			RetryMaxDelay
	}

	return delay +
		jitter
}

func boundedError(
	err error,
) string {
	if err == nil {
		return ""
	}

	message :=
		strings.TrimSpace(
			err.Error(),
		)

	const maxRunes = 2048

	runes :=
		[]rune(
			message,
		)

	if len(runes) >
		maxRunes {

		runes =
			runes[:maxRunes]
	}

	return string(
		runes,
	)
}

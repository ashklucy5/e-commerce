package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const MaxPayloadBytes = 256 << 10

type Message struct {
	StreamID string `json:"-"`

	ID      string          `json:"job_id"`
	Type    string          `json:"job_type"`
	Payload json.RawMessage `json:"payload"`

	Attempt     int `json:"attempt"`
	MaxAttempts int `json:"max_attempts"`

	CreatedAt time.Time `json:"created_at"`
}

type permanentError struct {
	err error
}

func (e *permanentError) Error() string {
	if e == nil ||
		e.err == nil {

		return "permanent queue error"
	}

	return e.err.Error()
}

func (e *permanentError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

func Permanent(
	err error,
) error {
	if err == nil {
		return nil
	}

	return &permanentError{
		err: err,
	}
}

func IsPermanent(
	err error,
) bool {
	var target *permanentError

	return errors.As(
		err,
		&target,
	)
}

func decodeMessage(
	input redis.XMessage,
) (
	Message,
	error,
) {
	jobID, ok :=
		messageString(
			input.Values,
			"job_id",
		)
	if !ok ||
		strings.TrimSpace(
			jobID,
		) == "" {

		return Message{},
			fmt.Errorf(
				"queue message %s is missing job_id",
				input.ID,
			)
	}

	jobType, ok :=
		messageString(
			input.Values,
			"job_type",
		)
	if !ok {
		return Message{},
			fmt.Errorf(
				"queue message %s is missing job_type",
				input.ID,
			)
	}

	jobType =
		normalizeJobType(
			jobType,
		)

	if !validJobType(
		jobType,
	) {
		return Message{},
			fmt.Errorf(
				"queue message %s has invalid job_type",
				input.ID,
			)
	}

	payload, ok :=
		messageString(
			input.Values,
			"payload",
		)
	if !ok ||
		strings.TrimSpace(
			payload,
		) == "" {

		payload =
			"{}"
	}

	if len(
		payload,
	) >
		MaxPayloadBytes {

		return Message{},
			fmt.Errorf(
				"queue message %s payload exceeds %d bytes",
				input.ID,
				MaxPayloadBytes,
			)
	}

	if !json.Valid(
		[]byte(
			payload,
		),
	) {
		return Message{},
			fmt.Errorf(
				"queue message %s contains invalid JSON payload",
				input.ID,
			)
	}

	attemptValue, ok :=
		messageString(
			input.Values,
			"attempt",
		)
	if !ok {
		return Message{},
			fmt.Errorf(
				"queue message %s is missing attempt",
				input.ID,
			)
	}

	attempt, err :=
		strconv.Atoi(
			attemptValue,
		)
	if err != nil ||
		attempt <= 0 {

		return Message{},
			fmt.Errorf(
				"queue message %s has invalid attempt",
				input.ID,
			)
	}

	maxAttemptsValue, ok :=
		messageString(
			input.Values,
			"max_attempts",
		)
	if !ok {
		return Message{},
			fmt.Errorf(
				"queue message %s is missing max_attempts",
				input.ID,
			)
	}

	maxAttempts, err :=
		strconv.Atoi(
			maxAttemptsValue,
		)
	if err != nil ||
		maxAttempts <= 0 ||
		attempt > maxAttempts {

		return Message{},
			fmt.Errorf(
				"queue message %s has invalid max_attempts",
				input.ID,
			)
	}

	createdAtValue, ok :=
		messageString(
			input.Values,
			"created_at",
		)
	if !ok {
		return Message{},
			fmt.Errorf(
				"queue message %s is missing created_at",
				input.ID,
			)
	}

	createdAtMillis, err :=
		strconv.ParseInt(
			createdAtValue,
			10,
			64,
		)
	if err != nil ||
		createdAtMillis <= 0 {

		return Message{},
			fmt.Errorf(
				"queue message %s has invalid created_at",
				input.ID,
			)
	}

	return Message{
			StreamID: input.ID,

			ID: strings.TrimSpace(
				jobID,
			),

			Type: jobType,

			Payload: json.RawMessage(
				payload,
			),

			Attempt: attempt,

			MaxAttempts: maxAttempts,

			CreatedAt: time.UnixMilli(
				createdAtMillis,
			).UTC(),
		},
		nil
}

func messageString(
	values map[string]interface{},
	key string,
) (
	string,
	bool,
) {
	value, ok :=
		values[key]

	if !ok ||
		value == nil {

		return "", false
	}

	switch typed :=
		value.(type) {

	case string:
		return typed, true

	case []byte:
		return string(
			typed,
		), true

	case int:
		return strconv.Itoa(
			typed,
		), true

	case int64:
		return strconv.FormatInt(
			typed,
			10,
		), true

	case uint64:
		return strconv.FormatUint(
			typed,
			10,
		), true

	default:
		return fmt.Sprint(
			typed,
		), true
	}
}

func normalizeJobType(
	value string,
) string {
	return strings.ToLower(
		strings.TrimSpace(
			value,
		),
	)
}

func validJobType(
	value string,
) bool {
	if value == "" ||
		len(value) > 128 {

		return false
	}

	for index :=
		0; index < len(value); index++ {

		character :=
			value[index]

		if (character >= 'a' &&
			character <= 'z') ||
			(character >= '0' &&
				character <= '9') ||
			character == '.' ||
			character == '_' ||
			character == '-' {

			continue
		}

		return false
	}

	return true
}

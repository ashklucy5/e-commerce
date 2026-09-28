package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/platform/security"
)

const (
	RequestIDHeader = "X-Request-ID"

	requestIDContextKey = "request_id"

	requestIDBytes = 18

	maxRequestIDLength = 128
)

type requestIDRequestContextKey struct{}

var requestIDFallbackCounter atomic.Uint64

func RequestID() gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		requestID :=
			strings.TrimSpace(
				c.GetHeader(
					RequestIDHeader,
				),
			)

		if !validRequestID(
			requestID,
		) {
			requestID =
				newRequestID()
		}

		c.Set(
			requestIDContextKey,
			requestID,
		)

		c.Header(
			RequestIDHeader,
			requestID,
		)

		requestContext :=
			context.WithValue(
				c.Request.Context(),
				requestIDRequestContextKey{},
				requestID,
			)

		c.Request =
			c.Request.WithContext(
				requestContext,
			)

		c.Next()
	}
}

func RequestIDFromContext(
	c *gin.Context,
) (string, bool) {
	if c == nil {
		return "", false
	}

	value, exists :=
		c.Get(
			requestIDContextKey,
		)

	if !exists {
		return "", false
	}

	requestID, ok :=
		value.(string)

	if !ok ||
		requestID == "" {
		return "", false
	}

	return requestID, true
}

func RequestIDFromRequestContext(
	ctx context.Context,
) (string, bool) {
	if ctx == nil {
		return "", false
	}

	requestID, ok :=
		ctx.Value(
			requestIDRequestContextKey{},
		).(string)

	if !ok ||
		requestID == "" {
		return "", false
	}

	return requestID, true
}

func newRequestID() string {
	requestID, err :=
		security.RandomURLSafe(
			requestIDBytes,
		)

	if err == nil {
		return requestID
	}

	// Request IDs are correlation identifiers rather than secrets.
	// If crypto/rand becomes unavailable, retain observability instead
	// of crashing the HTTP server.
	sequence :=
		requestIDFallbackCounter.
			Add(
				1,
			)

	return fmt.Sprintf(
		"fallback-%d-%d",
		time.Now().
			UTC().
			UnixNano(),
		sequence,
	)
}

func validRequestID(
	value string,
) bool {
	if value == "" ||
		len(value) >
			maxRequestIDLength {
		return false
	}

	for index := 0; index < len(value); index++ {

		character :=
			value[index]

		if (character >= 'a' &&
			character <= 'z') ||
			(character >= 'A' &&
				character <= 'Z') ||
			(character >= '0' &&
				character <= '9') ||
			character == '-' ||
			character == '_' ||
			character == '.' {

			continue
		}

		return false
	}

	return true
}

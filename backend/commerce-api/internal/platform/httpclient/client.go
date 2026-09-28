package httpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 15 * time.Second

	defaultMaxResponseBytes int64 = 16 << 20
)

var ErrResponseTooLarge = errors.New(
	"HTTP response body exceeds configured limit",
)

type Config struct {
	Timeout time.Duration

	MaxResponseBytes int64

	Retry RetryConfig

	CircuitBreaker CircuitBreakerConfig

	Transport http.RoundTripper
}

type Client struct {
	client *http.Client

	retry RetryConfig

	breaker *CircuitBreaker

	maxResponseBytes int64
}

func DefaultConfig() Config {
	return Config{
		Timeout: defaultTimeout,

		MaxResponseBytes: defaultMaxResponseBytes,

		Retry: DefaultRetryConfig(),

		CircuitBreaker: DefaultCircuitBreakerConfig(),
	}
}

func New(
	cfg Config,
) *Client {
	defaults :=
		DefaultConfig()

	if cfg.Timeout <= 0 {
		cfg.Timeout =
			defaults.Timeout
	}

	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes =
			defaults.MaxResponseBytes
	}

	cfg.Retry =
		normalizeRetryConfig(
			cfg.Retry,
		)

	cfg.CircuitBreaker =
		normalizeCircuitBreakerConfig(
			cfg.CircuitBreaker,
		)

	transport :=
		cfg.Transport

	if transport == nil {
		transport =
			defaultTransport()
	}

	return &Client{
		client: &http.Client{
			Transport: transport,

			Timeout: cfg.Timeout,
		},

		retry: cfg.Retry,

		breaker: NewCircuitBreaker(
			cfg.CircuitBreaker,
		),

		maxResponseBytes: cfg.MaxResponseBytes,
	}
}

func (c *Client) Do(
	request *http.Request,
) (
	*http.Response,
	error,
) {
	if c == nil {
		return nil,
			errors.New(
				"HTTP client is nil",
			)
	}

	if request == nil {
		return nil,
			errors.New(
				"HTTP request is nil",
			)
	}

	now :=
		time.Now().
			UTC()

	if err :=
		c.breaker.Before(
			now,
		); err != nil {

		return nil, err
	}

	maxAttempts :=
		1

	if requestMayRetry(
		request,
	) {
		maxAttempts =
			c.retry.MaxAttempts
	}

	if request.Body != nil &&
		request.Body != http.NoBody &&
		request.GetBody == nil &&
		maxAttempts > 1 {

		// The body cannot safely be replayed.
		maxAttempts = 1
	}

	var lastErr error

	for attempt :=
		1; attempt <= maxAttempts; attempt++ {

		if err :=
			request.Context().
				Err(); err != nil {

			c.breaker.Failure(
				time.Now().
					UTC(),
			)

			return nil, err
		}

		attemptRequest, err :=
			requestForAttempt(
				request,
				attempt,
			)
		if err != nil {
			c.breaker.Failure(
				time.Now().
					UTC(),
			)

			return nil, err
		}

		response, err :=
			c.client.Do(
				attemptRequest,
			)

		if err != nil {
			lastErr =
				err

			if request.Context().
				Err() != nil {

				c.breaker.Failure(
					time.Now().
						UTC(),
				)

				return nil,
					request.Context().
						Err()
			}

			if attempt <
				maxAttempts &&
				shouldRetryError(
					err,
				) {

				if waitErr :=
					waitForRetry(
						request.Context(),
						retryDelay(
							c.retry,
							attempt,
							nil,
							time.Now().
								UTC(),
						),
					); waitErr != nil {

					c.breaker.Failure(
						time.Now().
							UTC(),
					)

					return nil,
						waitErr
				}

				continue
			}

			c.breaker.Failure(
				time.Now().
					UTC(),
			)

			return nil,
				fmt.Errorf(
					"HTTP request failed after %d attempt(s): %w",
					attempt,
					err,
				)
		}

		if shouldRetryStatus(
			response.StatusCode,
		) &&
			attempt <
				maxAttempts {

			drainAndClose(
				response,
			)

			if err :=
				waitForRetry(
					request.Context(),
					retryDelay(
						c.retry,
						attempt,
						response,
						time.Now().
							UTC(),
					),
				); err != nil {

				c.breaker.Failure(
					time.Now().
						UTC(),
				)

				return nil, err
			}

			continue
		}

		if shouldRetryStatus(
			response.StatusCode,
		) {
			c.breaker.Failure(
				time.Now().
					UTC(),
			)
		} else {
			c.breaker.Success()
		}

		return response, nil
	}

	c.breaker.Failure(
		time.Now().
			UTC(),
	)

	if lastErr != nil {
		return nil,
			lastErr
	}

	return nil,
		errors.New(
			"HTTP request failed",
		)
}

// ReadAll reads and closes no resources itself; callers still own
// response.Body and should defer response.Body.Close().
//
// It rejects responses exceeding MaxResponseBytes rather than silently
// truncating them.
func (c *Client) ReadAll(
	response *http.Response,
) (
	[]byte,
	error,
) {
	if c == nil {
		return nil,
			errors.New(
				"HTTP client is nil",
			)
	}

	if response == nil ||
		response.Body == nil {

		return nil,
			errors.New(
				"HTTP response body is nil",
			)
	}

	reader :=
		io.LimitReader(
			response.Body,
			c.maxResponseBytes+1,
		)

	body, err :=
		io.ReadAll(
			reader,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"read HTTP response body: %w",
				err,
			)
	}

	if int64(
		len(body),
	) >
		c.maxResponseBytes {

		return nil,
			fmt.Errorf(
				"%w: maximum %d bytes",
				ErrResponseTooLarge,
				c.maxResponseBytes,
			)
	}

	return body, nil
}

func requestForAttempt(
	request *http.Request,
	attempt int,
) (
	*http.Request,
	error,
) {
	result :=
		request.Clone(
			request.Context(),
		)

	result.Header =
		request.Header.Clone()

	if request.Body == nil ||
		request.Body == http.NoBody {

		result.Body =
			request.Body

		return result, nil
	}

	if request.GetBody != nil {
		body, err :=
			request.GetBody()
		if err != nil {
			return nil,
				fmt.Errorf(
					"recreate HTTP request body: %w",
					err,
				)
		}

		result.Body =
			body

		return result, nil
	}

	if attempt == 1 {
		result.Body =
			request.Body

		return result, nil
	}

	return nil,
		errors.New(
			"HTTP request body cannot be replayed",
		)
}

func drainAndClose(
	response *http.Response,
) {
	if response == nil ||
		response.Body == nil {

		return
	}

	_, _ =
		io.Copy(
			io.Discard,
			io.LimitReader(
				response.Body,
				64<<10,
			),
		)

	_ =
		response.Body.Close()
}

func defaultTransport() http.RoundTripper {
	base, ok :=
		http.DefaultTransport.(*http.Transport)

	if !ok {
		return http.DefaultTransport
	}

	transport :=
		base.Clone()

	transport.MaxIdleConns =
		100

	transport.MaxIdleConnsPerHost =
		10

	transport.IdleConnTimeout =
		90 * time.Second

	transport.TLSHandshakeTimeout =
		10 * time.Second

	transport.ExpectContinueTimeout =
		time.Second

	return transport
}

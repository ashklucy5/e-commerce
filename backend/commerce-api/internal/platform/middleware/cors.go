package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

var ErrInvalidCORSConfig = errors.New(
	"invalid CORS configuration",
)

type CORSConfig struct {
	AllowedOrigins []string

	AllowedMethods []string

	AllowedHeaders []string

	ExposedHeaders []string

	AllowCredentials bool

	MaxAge time.Duration
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},

		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"X-CSRF-Token",
		},

		ExposedHeaders: []string{
			"X-Request-ID",
		},

		MaxAge: 10 * time.Minute,
	}
}

func NewCORS(
	cfg CORSConfig,
) (
	gin.HandlerFunc,
	error,
) {
	normalized, err := normalizeCORSConfig(
		cfg,
	)
	if err != nil {
		return nil, err
	}

	return func(
			c *gin.Context,
		) {
			origin := strings.TrimSpace(
				c.GetHeader(
					"Origin",
				),
			)

			if origin == "" {
				c.Next()

				return
			}

			originAllowed :=
				normalized.allowAnyOrigin

			if !originAllowed {
				_, originAllowed =
					normalized.allowedOrigins[normalizeOrigin(
						origin,
					)]
			}

			isPreflight :=
				c.Request.Method ==
					http.MethodOptions &&
					strings.TrimSpace(
						c.GetHeader(
							"Access-Control-Request-Method",
						),
					) != ""

			if isPreflight &&
				!originAllowed {
				platformerrors.Abort(
					c,
					platformerrors.Forbidden(
						"CORS_ORIGIN_DENIED",
						"Cross-origin request is not allowed",
					),
				)

				return
			}

			// For a non-preflight request, CORS is not authentication.
			// If the origin is not allowed, simply omit CORS response
			// headers and allow the normal authentication/authorization
			// layers to make the actual security decision.
			if !originAllowed {
				c.Next()

				return
			}

			addVary(
				c,
				"Origin",
			)

			if normalized.allowAnyOrigin {
				c.Header(
					"Access-Control-Allow-Origin",
					"*",
				)
			} else {
				c.Header(
					"Access-Control-Allow-Origin",
					origin,
				)
			}

			if normalized.allowCredentials {
				c.Header(
					"Access-Control-Allow-Credentials",
					"true",
				)
			}

			if len(
				normalized.exposedHeaders,
			) > 0 {
				c.Header(
					"Access-Control-Expose-Headers",
					strings.Join(
						normalized.exposedHeaders,
						", ",
					),
				)
			}

			if !isPreflight {
				c.Next()

				return
			}

			addVary(
				c,
				"Access-Control-Request-Method",
			)

			addVary(
				c,
				"Access-Control-Request-Headers",
			)

			requestedMethod :=
				strings.ToUpper(
					strings.TrimSpace(
						c.GetHeader(
							"Access-Control-Request-Method",
						),
					),
				)

			if _, ok :=
				normalized.allowedMethods[requestedMethod]; !ok {

				platformerrors.Abort(
					c,
					platformerrors.Forbidden(
						"CORS_METHOD_DENIED",
						"Cross-origin request method is not allowed",
					),
				)

				return
			}

			if !requestedHeadersAllowed(
				c.GetHeader(
					"Access-Control-Request-Headers",
				),
				normalized.allowedHeaders,
			) {
				platformerrors.Abort(
					c,
					platformerrors.Forbidden(
						"CORS_HEADERS_DENIED",
						"Cross-origin request headers are not allowed",
					),
				)

				return
			}

			c.Header(
				"Access-Control-Allow-Methods",
				strings.Join(
					normalized.allowedMethodsList,
					", ",
				),
			)

			if len(
				normalized.allowedHeadersList,
			) > 0 {
				c.Header(
					"Access-Control-Allow-Headers",
					strings.Join(
						normalized.allowedHeadersList,
						", ",
					),
				)
			}

			if normalized.maxAge > 0 {
				c.Header(
					"Access-Control-Max-Age",
					strconv.FormatInt(
						int64(
							normalized.maxAge/time.Second,
						),
						10,
					),
				)
			}

			c.AbortWithStatus(
				http.StatusNoContent,
			)
		},
		nil
}

type normalizedCORSConfig struct {
	allowedOrigins map[string]struct{}

	allowAnyOrigin bool

	allowedMethods map[string]struct{}

	allowedMethodsList []string

	allowedHeaders map[string]struct{}

	allowedHeadersList []string

	exposedHeaders []string

	allowCredentials bool

	maxAge time.Duration
}

func normalizeCORSConfig(
	cfg CORSConfig,
) (
	normalizedCORSConfig,
	error,
) {
	result := normalizedCORSConfig{
		allowedOrigins: make(
			map[string]struct{},
		),

		allowedMethods: make(
			map[string]struct{},
		),

		allowedHeaders: make(
			map[string]struct{},
		),

		allowCredentials: cfg.AllowCredentials,

		maxAge: cfg.MaxAge,
	}

	for _, rawOrigin := range cfg.AllowedOrigins {

		origin := strings.TrimSpace(
			rawOrigin,
		)

		if origin == "" {
			continue
		}

		if origin == "*" {
			result.allowAnyOrigin =
				true

			continue
		}

		normalized :=
			normalizeOrigin(
				origin,
			)

		parsed, err :=
			url.Parse(
				normalized,
			)
		if err != nil ||
			parsed.Scheme == "" ||
			parsed.Host == "" ||
			parsed.User != nil ||
			parsed.RawQuery != "" ||
			parsed.Fragment != "" ||
			(parsed.Path != "" &&
				parsed.Path != "/") {

			return normalizedCORSConfig{},
				fmt.Errorf(
					"%w: invalid origin %q",
					ErrInvalidCORSConfig,
					origin,
				)
		}

		result.allowedOrigins[normalized] = struct{}{}
	}

	if result.allowAnyOrigin &&
		result.allowCredentials {
		return normalizedCORSConfig{},
			fmt.Errorf(
				"%w: wildcard origin cannot be combined with credentials",
				ErrInvalidCORSConfig,
			)
	}

	methods :=
		cfg.AllowedMethods

	if len(methods) == 0 {
		methods =
			DefaultCORSConfig().
				AllowedMethods
	}

	for _, rawMethod := range methods {

		method := strings.ToUpper(
			strings.TrimSpace(
				rawMethod,
			),
		)

		if method == "" {
			continue
		}

		if _, exists :=
			result.allowedMethods[method]; exists {
			continue
		}

		result.allowedMethods[method] = struct{}{}

		result.allowedMethodsList =
			append(
				result.allowedMethodsList,
				method,
			)
	}

	headers :=
		cfg.AllowedHeaders

	if len(headers) == 0 {
		headers =
			DefaultCORSConfig().
				AllowedHeaders
	}

	for _, rawHeader := range headers {

		header :=
			http.CanonicalHeaderKey(
				strings.TrimSpace(
					rawHeader,
				),
			)

		if header == "" {
			continue
		}

		key :=
			strings.ToLower(
				header,
			)

		if _, exists :=
			result.allowedHeaders[key]; exists {
			continue
		}

		result.allowedHeaders[key] = struct{}{}

		result.allowedHeadersList =
			append(
				result.allowedHeadersList,
				header,
			)
	}

	for _, rawHeader := range cfg.ExposedHeaders {

		header :=
			http.CanonicalHeaderKey(
				strings.TrimSpace(
					rawHeader,
				),
			)

		if header == "" {
			continue
		}

		result.exposedHeaders =
			append(
				result.exposedHeaders,
				header,
			)
	}

	if result.maxAge < 0 {
		return normalizedCORSConfig{},
			fmt.Errorf(
				"%w: max age cannot be negative",
				ErrInvalidCORSConfig,
			)
	}

	return result, nil
}

func requestedHeadersAllowed(
	value string,
	allowed map[string]struct{},
) bool {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return true
	}

	for _, rawHeader := range strings.Split(
		value,
		",",
	) {

		header :=
			strings.ToLower(
				strings.TrimSpace(
					rawHeader,
				),
			)

		if header == "" {
			continue
		}

		if _, ok :=
			allowed[header]; !ok {

			return false
		}
	}

	return true
}

func normalizeOrigin(
	value string,
) string {
	return strings.TrimSuffix(
		strings.TrimSpace(
			value,
		),
		"/",
	)
}

func addVary(
	c *gin.Context,
	value string,
) {
	for _, existing := range c.Writer.Header().
		Values(
			"Vary",
		) {

		for _, candidate := range strings.Split(
			existing,
			",",
		) {

			if strings.EqualFold(
				strings.TrimSpace(
					candidate,
				),
				value,
			) {
				return
			}
		}
	}

	c.Writer.Header().
		Add(
			"Vary",
			value,
		)
}

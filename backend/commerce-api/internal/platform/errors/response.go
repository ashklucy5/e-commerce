package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Error ResponseError `json:"error"`
}

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`

	Details any `json:"details,omitempty"`
}

// ResponseFor converts any application error into the public HTTP
// representation.
//
// Unknown errors never expose their raw message.
func ResponseFor(
	err error,
) (
	int,
	Response,
) {
	apiErr := Normalize(
		err,
	)

	if apiErr == nil {
		apiErr = New(
			http.StatusInternalServerError,
			CodeInternal,
			"Internal server error",
		)
	}

	return apiErr.Status,
		Response{
			Error: ResponseError{
				Code:    apiErr.Code,
				Message: apiErr.Message,
				Details: apiErr.Details,
			},
		}
}

// Write writes an error response without aborting Gin's handler chain.
//
// Use this from normal endpoint handlers, where the handler returns
// immediately after calling Write.
func Write(
	c *gin.Context,
	err error,
) {
	status, response := ResponseFor(
		err,
	)

	c.JSON(
		status,
		response,
	)
}

// Abort writes an error response and aborts the Gin handler chain.
//
// This is intended primarily for authentication, authorization,
// rate-limiting, and other middleware.
func Abort(
	c *gin.Context,
	err error,
) {
	status, response := ResponseFor(
		err,
	)

	c.AbortWithStatusJSON(
		status,
		response,
	)
}

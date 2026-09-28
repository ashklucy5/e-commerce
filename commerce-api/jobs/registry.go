package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"project.local/commerce-api/internal/platform/queue"
)

type HandlerFunc func(
	ctx context.Context,
	message queue.Message,
) error

type Registry struct {
	mu sync.RWMutex

	handlers map[string]HandlerFunc
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(
			map[string]HandlerFunc,
		),
	}
}

func (r *Registry) Register(
	jobType string,
	handler HandlerFunc,
) error {
	jobType =
		strings.ToLower(
			strings.TrimSpace(
				jobType,
			),
		)

	if jobType == "" {
		return fmt.Errorf(
			"job type is required",
		)
	}

	if handler == nil {
		return fmt.Errorf(
			"job handler for %s is required",
			jobType,
		)
	}

	r.mu.Lock()

	defer r.mu.Unlock()

	if _, exists :=
		r.handlers[jobType]; exists {

		return fmt.Errorf(
			"job handler already registered for %s",
			jobType,
		)
	}

	r.handlers[jobType] =
		handler

	return nil
}

func (r *Registry) Handle(
	ctx context.Context,
	message queue.Message,
) error {
	jobType :=
		strings.ToLower(
			strings.TrimSpace(
				message.Type,
			),
		)

	r.mu.RLock()

	handler, exists :=
		r.handlers[jobType]

	r.mu.RUnlock()

	if !exists {
		return queue.Permanent(
			fmt.Errorf(
				"no job handler registered for %s",
				jobType,
			),
		)
	}

	return handler(
		ctx,
		message,
	)
}

package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

type SendRequest struct {
	Recipient string

	Subject string
	Text    string

	HTML string

	DedupeKey string
}

type SendResult struct {
	ProviderMessageID string
}

type Sender interface {
	Channel() string

	Ready() error

	Send(
		ctx context.Context,
		request SendRequest,
	) (
		SendResult,
		error,
	)
}

type ChannelRegistry struct {
	mu sync.RWMutex

	senders map[string]Sender
}

func NewChannelRegistry() *ChannelRegistry {
	return &ChannelRegistry{
		senders: make(
			map[string]Sender,
		),
	}
}

func (r *ChannelRegistry) Register(
	sender Sender,
) error {
	if r == nil ||
		sender == nil {

		return ErrInvalidInput
	}

	channel :=
		strings.ToLower(
			strings.TrimSpace(
				sender.Channel(),
			),
		)

	if !validChannel(
		channel,
	) {
		return ErrInvalidInput
	}

	r.mu.Lock()

	defer r.mu.Unlock()

	if _, exists :=
		r.senders[channel]; exists {

		return fmt.Errorf(
			"notification sender already registered for %s",
			channel,
		)
	}

	r.senders[channel] =
		sender

	return nil
}

func (r *ChannelRegistry) Resolve(
	channel string,
) (
	Sender,
	error,
) {
	if r == nil {
		return nil,
			ErrChannelNotConfigured
	}

	channel =
		strings.ToLower(
			strings.TrimSpace(
				channel,
			),
		)

	if !validChannel(
		channel,
	) {
		return nil,
			ErrChannelNotConfigured
	}

	r.mu.RLock()

	sender, exists :=
		r.senders[channel]

	r.mu.RUnlock()

	if !exists ||
		sender == nil {

		return nil,
			ErrChannelNotConfigured
	}

	if err :=
		sender.Ready(); err != nil {

		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrChannelUnavailable,
				err,
			)
	}

	return sender,
		nil
}

func IsUnavailableChannelError(
	err error,
) bool {
	return errors.Is(
		err,
		ErrChannelNotConfigured,
	) ||
		errors.Is(
			err,
			ErrChannelUnavailable,
		)
}

package global

import "errors"

var (
	// ErrEmptyTitle is returned when valid connection closed
	ErrEmptyTitle = errors.New("no topic set")

	// ErrConnectionClosed is returned when valid connection closed
	ErrConnectionClosed = errors.New("connection is closed")

	// ErrAlreadySubscribed is returned when topic is already subscribed
	ErrAlreadySubscribed = errors.New("topic is already subscribed")

	// ErrNotSubscribed is returned when the topic is not returend
	ErrNotSubscribed = errors.New("topic is not subscribed")

	// ErrPubSubDisabled is returned when the protocol is disabled
	ErrPubSubDisabled = errors.New("pubsub protocol is disabled")
)

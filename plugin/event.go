package plugin

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrBusClosed is returned when publishing to a closed EventBus.
	ErrBusClosed = errors.New("event bus is closed")

	// ErrPublishTimeout is returned when the publish buffer is full and context expires.
	ErrPublishTimeout = errors.New("event publish timeout: buffer full")
)

// Event represents a system or plugin event.
type Event struct {
	Name      string    // e.g. "user.created"
	Data      any       // payload
	Source    string    // originating plugin name
	Timestamp time.Time // when the event was created
}

// EventHandler is the typed handler for events.
type EventHandler func(ctx context.Context, event Event) error

// Subscription represents an active event subscription.
type Subscription interface {
	Unsubscribe()
}

// EventBus is the single event mechanism for plugin communication.
type EventBus interface {
	// Publish sends an event. Blocks if buffer is full until ctx expires.
	Publish(ctx context.Context, event Event) error

	// Subscribe registers a handler for a topic. Returns a Subscription for unsubscribing.
	Subscribe(topic string, handler EventHandler) Subscription

	// Close drains pending events and waits for in-flight handlers to complete.
	Close() error
}

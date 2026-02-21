package runtime

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leeforge/framework/plugin"
	"go.uber.org/zap"
)

// eventBus implements plugin.EventBus with a buffered channel and backpressure.
type eventBus struct {
	subscribers map[string][]subscriberEntry
	mu          sync.RWMutex
	ch          chan eventEnvelope
	wg          sync.WaitGroup
	closed      atomic.Bool
	logger      *zap.Logger
	nextID      atomic.Uint64
	done        chan struct{} // signals dispatcher goroutine to stop
}

type eventEnvelope struct {
	ctx   context.Context
	event plugin.Event
}

type subscriberEntry struct {
	id      uint64
	handler plugin.EventHandler
}

// subscription implements plugin.Subscription.
type subscription struct {
	bus   *eventBus
	topic string
	id    uint64
}

func (s *subscription) Unsubscribe() {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	subs := s.bus.subscribers[s.topic]
	for i, entry := range subs {
		if entry.id == s.id {
			s.bus.subscribers[s.topic] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

// NewEventBus creates a new EventBus with the given buffer size.
func NewEventBus(bufferSize int, logger *zap.Logger) *eventBus {
	bus := &eventBus{
		subscribers: make(map[string][]subscriberEntry),
		ch:          make(chan eventEnvelope, bufferSize),
		logger:      logger,
		done:        make(chan struct{}),
	}

	go bus.dispatch()
	return bus
}

func (b *eventBus) dispatch() {
	for {
		select {
		case env, ok := <-b.ch:
			if !ok {
				return
			}
			b.fanOut(env)
		case <-b.done:
			// Drain remaining events in channel
			for {
				select {
				case env, ok := <-b.ch:
					if !ok {
						return
					}
					b.fanOut(env)
				default:
					return
				}
			}
		}
	}
}

func (b *eventBus) fanOut(env eventEnvelope) {
	b.mu.RLock()
	subs := append([]subscriberEntry{}, b.subscribers[env.event.Name]...)
	b.mu.RUnlock()

	for _, entry := range subs {
		b.wg.Add(1)
		go func(h plugin.EventHandler) {
			defer b.wg.Done()
			if err := h(env.ctx, env.event); err != nil {
				b.logger.Warn("event handler error",
					zap.String("event", env.event.Name),
					zap.Error(err))
			}
		}(entry.handler)
	}
}

// Publish sends an event. Blocks until buffer has space or ctx expires.
func (b *eventBus) Publish(ctx context.Context, event plugin.Event) error {
	if b.closed.Load() {
		return plugin.ErrBusClosed
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	env := eventEnvelope{ctx: ctx, event: event}

	select {
	case b.ch <- env:
		return nil
	default:
		// Buffer full -- block with backpressure
		select {
		case b.ch <- env:
			return nil
		case <-ctx.Done():
			return plugin.ErrPublishTimeout
		}
	}
}

// Subscribe registers a handler for a topic.
func (b *eventBus) Subscribe(topic string, handler plugin.EventHandler) plugin.Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID.Add(1)
	b.subscribers[topic] = append(b.subscribers[topic], subscriberEntry{
		id:      id,
		handler: handler,
	})

	return &subscription{bus: b, topic: topic, id: id}
}

// Close stops accepting new events, drains pending, and waits for in-flight handlers.
func (b *eventBus) Close() error {
	if b.closed.Swap(true) {
		return nil // already closed
	}

	close(b.done) // signal dispatcher to drain and stop
	b.wg.Wait()   // wait for in-flight handlers
	return nil
}

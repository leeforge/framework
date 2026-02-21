package runtime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leeforge/framework/plugin"
	"go.uber.org/zap"
)

func TestEventBus_PublishAndSubscribe(t *testing.T) {
	bus := NewEventBus(1024, zap.NewNop())
	defer bus.Close()

	var called atomic.Int32
	bus.Subscribe("test.event", func(ctx context.Context, e plugin.Event) error {
		called.Add(1)
		return nil
	})

	err := bus.Publish(context.Background(), plugin.Event{Name: "test.event", Data: "hello"})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if got := called.Load(); got != 1 {
		t.Fatalf("expected handler called 1 time, got %d", got)
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus(1024, zap.NewNop())
	defer bus.Close()

	var count atomic.Int32
	bus.Subscribe("evt", func(ctx context.Context, e plugin.Event) error {
		count.Add(1)
		return nil
	})
	bus.Subscribe("evt", func(ctx context.Context, e plugin.Event) error {
		count.Add(1)
		return nil
	})

	bus.Publish(context.Background(), plugin.Event{Name: "evt"})
	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 2 {
		t.Fatalf("expected 2 handler calls, got %d", got)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus(1024, zap.NewNop())
	defer bus.Close()

	var count atomic.Int32
	sub := bus.Subscribe("evt", func(ctx context.Context, e plugin.Event) error {
		count.Add(1)
		return nil
	})

	bus.Publish(context.Background(), plugin.Event{Name: "evt"})
	time.Sleep(100 * time.Millisecond)

	sub.Unsubscribe()

	bus.Publish(context.Background(), plugin.Event{Name: "evt"})
	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 call (unsubscribed before second), got %d", got)
	}
}

func TestEventBus_PublishAfterClose(t *testing.T) {
	bus := NewEventBus(1024, zap.NewNop())
	bus.Close()

	err := bus.Publish(context.Background(), plugin.Event{Name: "evt"})
	if err != plugin.ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}

func TestEventBus_CloseWaitsForInFlight(t *testing.T) {
	bus := NewEventBus(1024, zap.NewNop())

	done := make(chan struct{})
	bus.Subscribe("slow", func(ctx context.Context, e plugin.Event) error {
		time.Sleep(200 * time.Millisecond)
		close(done)
		return nil
	})

	bus.Publish(context.Background(), plugin.Event{Name: "slow"})
	time.Sleep(50 * time.Millisecond) // let dispatcher pick it up

	bus.Close() // should block until handler finishes

	select {
	case <-done:
		// good -- handler completed before Close returned
	default:
		t.Fatal("Close returned before in-flight handler completed")
	}
}

func TestEventBus_NoMatchingSubscriber(t *testing.T) {
	bus := NewEventBus(1024, zap.NewNop())
	defer bus.Close()

	// Should not error -- just no-op
	err := bus.Publish(context.Background(), plugin.Event{Name: "no.listeners"})
	if err != nil {
		t.Fatalf("Publish with no subscribers should not error: %v", err)
	}
}

func TestEventBus_ContextCancellation(t *testing.T) {
	// Small buffer to force backpressure
	bus := NewEventBus(1, zap.NewNop())
	defer bus.Close()

	bus.Subscribe("fill", func(ctx context.Context, e plugin.Event) error {
		time.Sleep(5 * time.Second) // intentionally slow
		return nil
	})

	// Fill the buffer
	bus.Publish(context.Background(), plugin.Event{Name: "fill"})

	// Now try with a cancelled context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := bus.Publish(ctx, plugin.Event{Name: "fill"})
	// Should either succeed (if buffer drained) or timeout
	if err != nil && err != plugin.ErrPublishTimeout {
		t.Fatalf("expected nil or ErrPublishTimeout, got %v", err)
	}
}

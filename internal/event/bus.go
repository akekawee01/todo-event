package event

import (
	"context"
	"log/slog"
	"sync"
)

type Event struct {
	Type    string
	Payload any
}

type Publisher interface {
	Publish(ctx context.Context, e Event)
}

type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]func(context.Context, Event) error
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]func(context.Context, Event) error)}
}

func (b *EventBus) Subscribe(eventType string, fn func(context.Context, Event) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], fn)
}

func (b *EventBus) Publish(ctx context.Context, e Event) {
	b.mu.RLock()
	handlers := b.handlers[e.Type]
	b.mu.RUnlock()

	for _, fn := range handlers {
		if err := fn(ctx, e); err != nil {
			slog.Error("eventbus: handler error", "event_type", e.Type, "err", err)
		}
	}
}

var _ Publisher = (*EventBus)(nil)

type FanoutPublisher struct {
	publishers []Publisher
}

func NewFanoutPublisher(pubs ...Publisher) Publisher {
	return &FanoutPublisher{publishers: pubs}
}

func (f *FanoutPublisher) Publish(ctx context.Context, e Event) {
	for _, p := range f.publishers {
		p.Publish(ctx, e)
	}
}

var _ Publisher = (*FanoutPublisher)(nil)

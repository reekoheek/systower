package systower

import "github.com/reekoheek/systower/internal/event"

type EventHandler func(event.Event)

type eventBus struct {
	handlers map[event.Kind][]EventHandler
}

func newEventBus() *eventBus {
	return &eventBus{
		handlers: make(map[event.Kind][]EventHandler),
	}
}

func (b *eventBus) on(kind event.Kind, h EventHandler) {
	b.handlers[kind] = append(b.handlers[kind], h)
}

func (b *eventBus) publish(e event.Event) {
	for _, h := range b.handlers[e.Kind] {
		h(e)
	}
}

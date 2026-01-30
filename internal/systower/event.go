package systower

import "sync"

type EventKind int

const (
	BatteryUpdated EventKind = iota
	CaffeineUpdated
	ClockUpdated
	CPUUpdated
	MemUpdated
	StorageUpdated
)

type Event struct {
	Kind    EventKind
	Payload any
}

type EventHandler func(Event)

type eventBus struct {
	mu       sync.RWMutex
	handlers map[EventKind][]EventHandler
}

func newEventBus() *eventBus {
	return &eventBus{
		handlers: make(map[EventKind][]EventHandler),
	}
}

func (b *eventBus) on(kind EventKind, h EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[kind] = append(b.handlers[kind], h)
}

func (b *eventBus) publish(e Event) {
	b.mu.RLock()
	handlers := b.handlers[e.Kind]
	b.mu.RUnlock()

	for _, h := range handlers {
		h(e)
	}
}

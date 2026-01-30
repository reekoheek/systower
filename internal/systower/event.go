package systower

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
	handlers map[EventKind][]EventHandler
}

func newEventBus() *eventBus {
	return &eventBus{
		handlers: make(map[EventKind][]EventHandler),
	}
}

func (b *eventBus) on(kind EventKind, h EventHandler) {
	b.handlers[kind] = append(b.handlers[kind], h)
}

func (b *eventBus) publish(e Event) {
	for _, h := range b.handlers[e.Kind] {
		h(e)
	}
}

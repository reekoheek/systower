package poller

import (
	"context"
	"reflect"
	"time"

	"github.com/reekoheek/systower/internal/event"
)

type task struct {
	interval time.Duration
	elapsed  time.Duration
	fn       func() event.Event
	last     event.Event
}

type Poller struct {
	tasks []*task
}

func New() *Poller {
	return &Poller{}
}

func (p *Poller) Register(interval time.Duration, fn func() event.Event) {
	p.tasks = append(p.tasks, &task{
		interval: interval,
		fn:       fn,
	})
}

func (p *Poller) base() time.Duration {
	if len(p.tasks) == 0 {
		return time.Second
	}
	result := p.tasks[0].interval
	for _, t := range p.tasks[1:] {
		result = gcd(result, t.interval)
	}
	return result
}

func gcd(a, b time.Duration) time.Duration {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (p *Poller) Poll(ctx context.Context, onChange func([]event.Event)) {
	base := p.base()

	go func() {
		// Run all tasks immediately and collect initial values
		events := make([]event.Event, 0, len(p.tasks))
		for _, t := range p.tasks {
			t.last = t.fn()
			events = append(events, t.last)
		}
		if len(events) > 0 {
			onChange(events)
		}

		ticker := time.NewTicker(base)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				events = events[:0]
				for _, t := range p.tasks {
					t.elapsed += base
					if t.elapsed >= t.interval {
						e := t.fn()
						if !reflect.DeepEqual(e.Payload, t.last.Payload) {
							t.last = e
							events = append(events, e)
						}
						t.elapsed = 0
					}
				}
				if len(events) > 0 {
					onChange(events)
				}
			}
		}
	}()
}

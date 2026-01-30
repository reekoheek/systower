package poller

import (
	"context"
	"time"
)

type Task struct {
	interval time.Duration
	elapsed  time.Duration
	fn       func()
}

type Poller struct {
	tasks []*Task
}

func New() *Poller {
	return &Poller{}
}

func (p *Poller) Register(interval time.Duration, fn func()) {
	p.tasks = append(p.tasks, &Task{
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

func (p *Poller) Poll(ctx context.Context) {
	base := p.base()

	go func() {
		// Run all tasks immediately
		for _, t := range p.tasks {
			t.fn()
		}

		ticker := time.NewTicker(base)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, t := range p.tasks {
					t.elapsed += base
					if t.elapsed >= t.interval {
						t.fn()
						t.elapsed = 0
					}
				}
			}
		}
	}()
}

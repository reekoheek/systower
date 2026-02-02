package poller

import (
	"context"
	"testing"
	"time"

	"github.com/reekoheek/systower/internal/event"
)

func TestGCD(t *testing.T) {
	tests := []struct {
		a, b, want time.Duration
	}{
		{1 * time.Second, 5 * time.Second, 1 * time.Second},
		{5 * time.Second, 60 * time.Second, 5 * time.Second},
		{2 * time.Second, 4 * time.Second, 2 * time.Second},
		{3 * time.Second, 5 * time.Second, 1 * time.Second},
	}

	for _, tt := range tests {
		got := gcd(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("gcd(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPoller_Base(t *testing.T) {
	p := New()
	p.Register(1*time.Second, func() event.Event { return event.Event{Kind: 0, Payload: 1} })
	p.Register(5*time.Second, func() event.Event { return event.Event{Kind: 1, Payload: 2} })
	p.Register(60*time.Second, func() event.Event { return event.Event{Kind: 2, Payload: 3} })

	if got := p.base(); got != 1*time.Second {
		t.Errorf("base() = %v, want %v", got, 1*time.Second)
	}
}

func TestPoller_Poll(t *testing.T) {
	p := New()

	var count1, count2 int
	p.Register(100*time.Millisecond, func() event.Event { count1++; return event.Event{Kind: 0, Payload: count1} })
	p.Register(200*time.Millisecond, func() event.Event { count2++; return event.Event{Kind: 1, Payload: count2} })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events []event.Event
	p.Poll(ctx, func(batch []event.Event) {
		events = append(events, batch...)
	})

	// Wait for initial + a few ticks
	time.Sleep(450 * time.Millisecond)

	// count1: initial + 4 ticks (100, 200, 300, 400) = 5
	// count2: initial + 2 ticks (200, 400) = 3
	if count1 < 4 || count1 > 6 {
		t.Errorf("count1 = %d, want ~5", count1)
	}
	if count2 < 2 || count2 > 4 {
		t.Errorf("count2 = %d, want ~3", count2)
	}

	// Should have received events (values always change since count increments)
	if len(events) == 0 {
		t.Error("expected events to be received")
	}
}

func TestPoller_OnlyChangedValues(t *testing.T) {
	p := New()

	val := 1
	p.Register(100*time.Millisecond, func() event.Event { return event.Event{Kind: 0, Payload: val} })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var eventCount int
	p.Poll(ctx, func(batch []event.Event) {
		eventCount += len(batch)
	})

	// Wait for initial + a few ticks
	time.Sleep(350 * time.Millisecond)

	// Should only get initial event since value never changes
	if eventCount != 1 {
		t.Errorf("eventCount = %d, want 1 (only initial)", eventCount)
	}
}

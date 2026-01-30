package poller

import (
	"testing"
	"time"
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
	p.Register(1*time.Second, func() {})
	p.Register(5*time.Second, func() {})
	p.Register(60*time.Second, func() {})

	if got := p.base(); got != 1*time.Second {
		t.Errorf("base() = %v, want %v", got, 1*time.Second)
	}
}

func TestPoller_Run(t *testing.T) {
	p := New()

	var count1, count2 int
	p.Register(100*time.Millisecond, func() { count1++ })
	p.Register(200*time.Millisecond, func() { count2++ })

	ch := p.Run()

	// Wait for initial + a few ticks
	time.Sleep(450 * time.Millisecond)

	// Drain channel
	for {
		select {
		case <-ch:
		default:
			goto done
		}
	}
done:

	// count1: initial + 4 ticks (100, 200, 300, 400) = 5
	// count2: initial + 2 ticks (200, 400) = 3
	if count1 < 4 || count1 > 6 {
		t.Errorf("count1 = %d, want ~5", count1)
	}
	if count2 < 2 || count2 > 4 {
		t.Errorf("count2 = %d, want ~3", count2)
	}
}

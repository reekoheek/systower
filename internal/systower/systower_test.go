package systower

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s, err := New(Intervals{Clock: 30, CPU: 5, Mem: 5, Storage: 60})
	if err != nil {
		t.Skipf("skipping test due to dbus unavailable: %v", err)
	}

	if s == nil {
		t.Error("New() should not return nil")
	}
}

func TestGCD(t *testing.T) {
	tests := []struct {
		a, b, want uint64
	}{
		{10, 5, 5},
		{5, 10, 5},
		{12, 8, 4},
		{30, 5, 5},
		{60, 30, 30},
		{7, 3, 1},
		{0, 5, 5},
		{5, 0, 5},
	}

	for _, tt := range tests {
		got := gcd(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("gcd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestBaseInterval(t *testing.T) {
	tests := []struct {
		name      string
		intervals Intervals
		want      time.Duration
	}{
		{
			name:      "common divisor 5",
			intervals: Intervals{Clock: 30, CPU: 5, Mem: 5, Storage: 60},
			want:      5 * time.Second,
		},
		{
			name:      "common divisor 1",
			intervals: Intervals{Clock: 7, CPU: 3, Mem: 5, Storage: 11},
			want:      1 * time.Second,
		},
		{
			name:      "all same",
			intervals: Intervals{Clock: 10, CPU: 10, Mem: 10, Storage: 10},
			want:      10 * time.Second,
		},
		{
			name:      "large GCD",
			intervals: Intervals{Clock: 60, CPU: 30, Mem: 30, Storage: 120},
			want:      30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.intervals.baseInterval()
			if got != tt.want {
				t.Errorf("baseInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

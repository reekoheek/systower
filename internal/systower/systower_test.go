package systower

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s, err := New(Intervals{
		Clock:    time.Second,
		CPU:      time.Second,
		Mem:      time.Second,
		Storage:  time.Second,
		Debounce: 100 * time.Millisecond,
	})

	if err != nil {
		t.Skipf("skipping test due to dbus unavailable: %v", err)
	}

	if s == nil {
		t.Error("New() should not return nil")
	}

	s.Close()
}

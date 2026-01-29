package systower

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s, err := New(time.Second, time.Second, time.Second, time.Second)

	if err != nil {
		t.Skipf("skipping test due to dbus unavailable: %v", err)
	}

	if s == nil {
		t.Error("New() should not return nil")
	}

	s.Close()
}

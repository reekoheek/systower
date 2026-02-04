package systower

import "testing"

func TestNew(t *testing.T) {
	s, err := New(Intervals{CPU: 3, Mem: 3, Storage: 60})
	if err != nil {
		t.Skipf("skipping test due to dbus unavailable: %v", err)
	}

	if s == nil {
		t.Error("New() should not return nil")
	}
}

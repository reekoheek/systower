package systower

import "testing"

func TestNew(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Skipf("skipping test due to dbus unavailable: %v", err)
	}

	if s == nil {
		t.Error("New() should not return nil")
	}

	s.Close()
}

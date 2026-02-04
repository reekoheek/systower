package caffeine

import "testing"

func TestNewX11Adapter(t *testing.T) {
	a := NewX11Adapter()
	if a == nil {
		t.Error("NewX11Adapter() should not return nil")
	}
}

func TestNewWaylandAdapter(t *testing.T) {
	lockfile := "/tmp/test-caffeine.lock"
	a := NewWaylandAdapter(lockfile)

	if a == nil {
		t.Error("NewWaylandAdapter() should not return nil")
	}
	if a.lockfile != lockfile {
		t.Errorf("lockfile = %v, want %v", a.lockfile, lockfile)
	}
}

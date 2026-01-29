package caffeine

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestWaylandAdapter_Status(t *testing.T) {
	tmpDir := t.TempDir()
	lockfile := filepath.Join(tmpDir, "caffeine.lock")

	a := NewWaylandAdapter(lockfile)

	// Initially off (no lockfile)
	if got := a.Status(); got != "off" {
		t.Errorf("Status() = %v, want off", got)
	}

	// Create lockfile
	if err := os.WriteFile(lockfile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create lockfile: %v", err)
	}

	// Now should be on
	if got := a.Status(); got != "on" {
		t.Errorf("Status() = %v, want on", got)
	}

	// Remove lockfile
	os.Remove(lockfile)

	// Back to off
	if got := a.Status(); got != "off" {
		t.Errorf("Status() = %v, want off", got)
	}
}

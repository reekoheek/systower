package caffeine

import (
	"os"
	"path/filepath"
	"testing"
)

type mockAdapter struct{}

func (m *mockAdapter) On()  {}
func (m *mockAdapter) Off() {}

func newTestCaffeine(t *testing.T) *Caffeine {
	t.Helper()
	tmpDir := t.TempDir()
	statusFile := filepath.Join(tmpDir, "caffeine.status")
	return NewWithStatusFile(&mockAdapter{}, statusFile)
}

func TestNew(t *testing.T) {
	adapter := &mockAdapter{}
	tmpDir := t.TempDir()
	statusFile := filepath.Join(tmpDir, "caffeine.status")
	c := NewWithStatusFile(adapter, statusFile)

	if c == nil {
		t.Error("New() should not return nil")
	}
	if c.adapter != adapter {
		t.Error("adapter should be set")
	}
}

func TestCaffeine_Read(t *testing.T) {
	c := newTestCaffeine(t)

	// Initially should be off (false) - file doesn't exist
	if got := c.Read(); got.Active {
		t.Errorf("Read().Active = %v, want false", got.Active)
	}

	// After On(), should be on (true)
	c.On()
	if got := c.Read(); !got.Active {
		t.Errorf("Read().Active = %v, want true", got.Active)
	}

	// After Off(), should be off (false)
	c.Off()
	if got := c.Read(); got.Active {
		t.Errorf("Read().Active = %v, want false", got.Active)
	}
}

func TestCaffeine_Toggle(t *testing.T) {
	c := newTestCaffeine(t)

	// Initially off, toggle should turn on
	c.Toggle()
	if got := c.Read(); !got.Active {
		t.Errorf("after toggle from off, Read().Active = %v, want true", got.Active)
	}

	// Now on, toggle should turn off
	c.Toggle()
	if got := c.Read(); got.Active {
		t.Errorf("after toggle from on, Read().Active = %v, want false", got.Active)
	}
}

func TestCaffeine_InitFdAndDrain(t *testing.T) {
	c := newTestCaffeine(t)

	fd, err := c.InitFd()
	if err != nil {
		t.Fatalf("InitFd() error = %v", err)
	}
	if fd < 0 {
		t.Error("InitFd() should return valid fd")
	}

	// Drain should not panic
	c.Drain()

	// Close should not panic
	c.Close()
}

func TestCaffeine_ReadExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	statusFile := filepath.Join(tmpDir, "caffeine.status")

	// Pre-create status file with "1"
	if err := os.WriteFile(statusFile, []byte("1"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	c := NewWithStatusFile(&mockAdapter{}, statusFile)
	if got := c.Read(); !got.Active {
		t.Errorf("Read().Active = %v, want true", got.Active)
	}
}

func TestStats_String(t *testing.T) {
	tests := []struct {
		active bool
		want   string
	}{
		{true, "on"},
		{false, "off"},
	}

	for _, tt := range tests {
		s := Stats{Active: tt.active}
		if got := s.String(); got != tt.want {
			t.Errorf("Stats{Active: %v}.String() = %q, want %q", tt.active, got, tt.want)
		}
	}
}

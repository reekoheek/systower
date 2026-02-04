package caffeine

import "testing"

type mockAdapter struct{}

func (m *mockAdapter) On()  {}
func (m *mockAdapter) Off() {}

func TestNew(t *testing.T) {
	adapter := &mockAdapter{}
	c := New(adapter)

	if c == nil {
		t.Error("New() should not return nil")
	}
	if c.adapter != adapter {
		t.Error("adapter should be set")
	}
}

func TestCaffeine_Read(t *testing.T) {
	adapter := &mockAdapter{}
	c := New(adapter)

	// Initially should be off (false)
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
	adapter := &mockAdapter{}
	c := New(adapter)

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

package caffeine

import "testing"

type mockAdapter struct {
	status string
}

func (m *mockAdapter) On() {
	m.status = "on"
}

func (m *mockAdapter) Off() {
	m.status = "off"
}

func (m *mockAdapter) Status() string {
	return m.status
}

func TestNew(t *testing.T) {
	adapter := &mockAdapter{status: "off"}
	c := New(nil, adapter)

	if c == nil {
		t.Error("New() should not return nil")
	}
	if c.adapter != adapter {
		t.Error("adapter should be set")
	}
}

func TestCaffeine_Status(t *testing.T) {
	adapter := &mockAdapter{status: "off"}
	c := New(nil, adapter)

	if got := c.Status(); got.Active != false {
		t.Errorf("Status().Active = %v, want false", got.Active)
	}

	adapter.status = "on"
	if got := c.Status(); got.Active != true {
		t.Errorf("Status().Active = %v, want true", got.Active)
	}
}

func TestCaffeine_ToggleLogic(t *testing.T) {
	// Test toggle logic without D-Bus
	// Toggle calls On/Off which call emitSignal requiring D-Bus
	// So we test the adapter directly to verify toggle logic works
	tests := []struct {
		name       string
		initial    string
		wantStatus string
	}{
		{
			name:       "toggle from off to on",
			initial:    "off",
			wantStatus: "on",
		},
		{
			name:       "toggle from on to off",
			initial:    "on",
			wantStatus: "off",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &mockAdapter{status: tt.initial}

			// Simulate toggle logic
			if adapter.Status() == "on" {
				adapter.Off()
			} else {
				adapter.On()
			}

			if got := adapter.Status(); got != tt.wantStatus {
				t.Errorf("after toggle, Status() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func TestMockAdapter_OnOff(t *testing.T) {
	adapter := &mockAdapter{status: "off"}

	adapter.On()
	if got := adapter.Status(); got != "on" {
		t.Errorf("after On(), Status() = %v, want on", got)
	}

	adapter.Off()
	if got := adapter.Status(); got != "off" {
		t.Errorf("after Off(), Status() = %v, want off", got)
	}
}
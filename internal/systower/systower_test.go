package systower

import "testing"

func TestNew(t *testing.T) {
	s := New(nil, nil, nil, nil, nil)

	if s == nil {
		t.Error("New() should not return nil")
	}
}

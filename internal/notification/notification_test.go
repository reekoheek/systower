package notification

import "testing"

func TestNew(t *testing.T) {
	title := "Test Title"
	n := New(nil, title)

	if n == nil {
		t.Error("New() should not return nil")
	}
	if n.title != title {
		t.Errorf("title = %v, want %v", n.title, title)
	}
	if n.conn != nil {
		t.Error("conn should be nil when passed nil")
	}
}

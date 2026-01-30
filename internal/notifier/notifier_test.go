package notifier

import (
	"testing"
	"time"
)

func TestNotifier_Debounce(t *testing.T) {
	n := New(50 * time.Millisecond)

	// Rapid notifications should be debounced
	for i := 0; i < 5; i++ {
		n.Notify()
		time.Sleep(10 * time.Millisecond)
	}

	// Should receive exactly one notification after debounce
	select {
	case <-n.Notified():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("expected notification after debounce")
	}

	// Should not receive another
	select {
	case <-n.Notified():
		t.Error("unexpected second notification")
	case <-time.After(100 * time.Millisecond):
		// OK
	}
}

func TestNotifier_MultipleRounds(t *testing.T) {
	n := New(20 * time.Millisecond)

	// First round
	n.Notify()
	select {
	case <-n.Notified():
	case <-time.After(50 * time.Millisecond):
		t.Error("expected first notification")
	}

	// Second round
	n.Notify()
	select {
	case <-n.Notified():
	case <-time.After(50 * time.Millisecond):
		t.Error("expected second notification")
	}
}

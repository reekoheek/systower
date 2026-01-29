package cpu

import (
	"testing"
	"time"
)

func TestReader_Read(t *testing.T) {
	r := New()

	// first read initializes, returns 0
	first := r.Read()
	if first != 0 {
		t.Errorf("first read should be 0, got %f", first)
	}

	// wait a bit for CPU activity
	time.Sleep(100 * time.Millisecond)

	// second read should return valid percentage
	second := r.Read()
	t.Logf("cpu: %.2f%%", second)
	if second < 0 || second > 100 {
		t.Errorf("cpu percentage should be 0-100, got %f", second)
	}
}

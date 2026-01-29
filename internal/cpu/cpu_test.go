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

func TestReader_ReadProcStat(t *testing.T) {
	r := New()

	idle, total := r.readProcStat()

	if total == 0 {
		t.Error("total should not be 0")
	}
	if idle == 0 {
		t.Error("idle should not be 0")
	}
	if idle > total {
		t.Errorf("idle (%d) should not exceed total (%d)", idle, total)
	}
}

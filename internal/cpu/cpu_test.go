package cpu

import (
	"testing"
	"time"
)

func TestReader_Read(t *testing.T) {
	r := New(time.Second)

	// first read (cumulative since boot)
	stats := r.Read()
	t.Logf("read 1 - idle: %d, total: %d, percent: %.1f%%", stats.Idle, stats.Total, stats.Percent())

	if stats.Total == 0 {
		t.Error("total should be > 0")
	}

	time.Sleep(100 * time.Millisecond)

	// second read (delta-based)
	stats = r.Read()
	t.Logf("read 2 - idle: %d, total: %d, percent: %.1f%%", stats.Idle, stats.Total, stats.Percent())

	if stats.Total == 0 {
		t.Error("delta total should be > 0")
	}
	if stats.Idle > stats.Total {
		t.Errorf("idle (%d) should not exceed total (%d)", stats.Idle, stats.Total)
	}
	if stats.Percent() < 0 || stats.Percent() > 100 {
		t.Errorf("percent should be 0-100, got %.1f", stats.Percent())
	}
}

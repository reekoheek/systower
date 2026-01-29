package cpu

import "testing"

func TestReader_Read(t *testing.T) {
	r := New()
	stats := r.Read()

	t.Logf("idle:  %d", stats.Idle)
	t.Logf("total: %d", stats.Total)

	if stats.Total == 0 {
		t.Error("total should be > 0")
	}
	if stats.Idle > stats.Total {
		t.Errorf("idle (%d) should not exceed total (%d)", stats.Idle, stats.Total)
	}
}

//go:build linux

package storage

import "testing"

func TestReader_Read(t *testing.T) {
	r := New("/")
	stats := r.Read()

	toGB := func(b uint64) float64 { return float64(b) / 1024 / 1024 / 1024 }
	pct := func(used, total uint64) float64 {
		if total == 0 {
			return 0
		}
		return float64(used) / float64(total) * 100
	}

	t.Logf("total: %12d B (%6.2f GB)", stats.Total, toGB(stats.Total))
	t.Logf("used:  %12d B (%6.2f GB) %5.1f%%", stats.Used, toGB(stats.Used), pct(stats.Used, stats.Total))
	t.Logf("free:  %12d B (%6.2f GB) %5.1f%%", stats.Free, toGB(stats.Free), pct(stats.Free, stats.Total))

	if stats.Total == 0 {
		t.Error("total should be > 0")
	}
	if stats.Used > stats.Total {
		t.Errorf("used (%d) should not exceed total (%d)", stats.Used, stats.Total)
	}
	if stats.Total != stats.Used+stats.Free {
		t.Errorf("total (%d) should equal used + free (%d)", stats.Total, stats.Used+stats.Free)
	}
}

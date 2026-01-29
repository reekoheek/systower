//go:build linux

package poller

import (
	"testing"
	"time"
)

func TestPoller_Start(t *testing.T) {
	p := New(100*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond)
	ch := p.Listen()

	// first poll (cumulative since boot, skip)
	stats := <-ch
	t.Logf("poll 1 - cpu: %5.1f%%, mem: %5.1f%%, storage: %5.1f%%",
		stats.CPU.Percent(),
		pct(stats.Mem.MemUsed, stats.Mem.Mem),
		pct(stats.Storage.Used, stats.Storage.Total))

	// second poll (delta-based, accurate)
	stats = <-ch
	cpuPct := stats.CPU.Percent()
	t.Logf("poll 2 - cpu: %5.1f%%, mem: %5.1f%%, storage: %5.1f%%",
		cpuPct,
		pct(stats.Mem.MemUsed, stats.Mem.Mem),
		pct(stats.Storage.Used, stats.Storage.Total))

	if cpuPct < 0 || cpuPct > 100 {
		t.Errorf("cpu should be 0-100, got %f", cpuPct)
	}
	if stats.Mem.Mem == 0 {
		t.Error("mem should be > 0")
	}
	if stats.Storage.Total == 0 {
		t.Error("storage total should be > 0")
	}
}

func pct(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

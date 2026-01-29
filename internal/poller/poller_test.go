//go:build linux

package poller

import (
	"testing"
	"time"

	"github.com/reekoheek/systower/internal/cpu"
)

func TestPoller_Start(t *testing.T) {
	p := New(100*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond)
	ch := p.Listen()

	var prevCPU cpu.Stats

	// first poll
	stats := <-ch
	t.Logf("poll 1 - cpu: idle=%d total=%d, mem: %5.1f%%, storage: %5.1f%%",
		stats.CPU.Idle, stats.CPU.Total,
		pct(stats.Mem.MemUsed, stats.Mem.Mem),
		pct(stats.Storage.Used, stats.Storage.Total))

	if stats.CPU.Total == 0 {
		t.Error("cpu total should be > 0")
	}
	prevCPU = stats.CPU

	// second poll
	stats = <-ch
	cpuPct := calcCPUPct(prevCPU, stats.CPU)
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

func calcCPUPct(prev, curr cpu.Stats) float64 {
	if prev.Total == 0 {
		return 0
	}
	deltaIdle := curr.Idle - prev.Idle
	deltaTotal := curr.Total - prev.Total
	if deltaTotal == 0 {
		return 0
	}
	return 100 * (1 - float64(deltaIdle)/float64(deltaTotal))
}

package mem

import "testing"

func TestReader_Read(t *testing.T) {
	r := New()
	stats := r.Read()

	toGB := func(kb uint64) float64 { return float64(kb) / 1024 / 1024 }
	pct := func(used, total uint64) float64 {
		if total == 0 {
			return 0
		}
		return float64(used) / float64(total) * 100
	}

	t.Logf("mem:  %12d kB (%6.2f GB)", stats.Mem, toGB(stats.Mem))
	t.Logf("used: %12d kB (%6.2f GB) %5.1f%%", stats.MemUsed, toGB(stats.MemUsed), pct(stats.MemUsed, stats.Mem))
	t.Logf("free: %12d kB (%6.2f GB) %5.1f%%", stats.MemFree, toGB(stats.MemFree), pct(stats.MemFree, stats.Mem))
	t.Logf("swap: %12d kB (%6.2f GB)", stats.Swap, toGB(stats.Swap))
	t.Logf("used: %12d kB (%6.2f GB) %5.1f%%", stats.SwapUsed, toGB(stats.SwapUsed), pct(stats.SwapUsed, stats.Swap))
	t.Logf("free: %12d kB (%6.2f GB) %5.1f%%", stats.SwapFree, toGB(stats.SwapFree), pct(stats.SwapFree, stats.Swap))

	if stats.Mem == 0 {
		t.Error("mem should be > 0")
	}
	if stats.MemUsed > stats.Mem {
		t.Errorf("memUsed (%d) should not exceed mem (%d)", stats.MemUsed, stats.Mem)
	}
	if stats.Mem != stats.MemUsed+stats.MemFree {
		t.Errorf("mem (%d) should equal memUsed + memFree (%d)", stats.Mem, stats.MemUsed+stats.MemFree)
	}
}

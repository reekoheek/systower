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

	t.Logf("all:  %12d kB (%6.2f GB)", stats.All, toGB(stats.All))
	t.Logf("used: %12d kB (%6.2f GB) %5.1f%%", stats.AllUsed, toGB(stats.AllUsed), pct(stats.AllUsed, stats.All))
	t.Logf("free: %12d kB (%6.2f GB) %5.1f%%", stats.AllFree, toGB(stats.AllFree), pct(stats.AllFree, stats.All))
	t.Logf("mem:  %12d kB (%6.2f GB)", stats.Mem, toGB(stats.Mem))
	t.Logf("used: %12d kB (%6.2f GB) %5.1f%%", stats.MemUsed, toGB(stats.MemUsed), pct(stats.MemUsed, stats.Mem))
	t.Logf("free: %12d kB (%6.2f GB) %5.1f%%", stats.MemFree, toGB(stats.MemFree), pct(stats.MemFree, stats.Mem))
	t.Logf("swap: %12d kB (%6.2f GB)", stats.Swap, toGB(stats.Swap))
	t.Logf("used: %12d kB (%6.2f GB) %5.1f%%", stats.SwapUsed, toGB(stats.SwapUsed), pct(stats.SwapUsed, stats.Swap))
	t.Logf("free: %12d kB (%6.2f GB) %5.1f%%", stats.SwapFree, toGB(stats.SwapFree), pct(stats.SwapFree, stats.Swap))

	if stats.All == 0 {
		t.Error("all should be > 0")
	}
	if stats.AllUsed > stats.All {
		t.Errorf("allUsed (%d) should not exceed all (%d)", stats.AllUsed, stats.All)
	}
	if stats.All != stats.AllUsed+stats.AllFree {
		t.Errorf("all (%d) should equal allUsed + allFree (%d)", stats.All, stats.AllUsed+stats.AllFree)
	}
}

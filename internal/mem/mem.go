package mem

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Stats struct {
	Mem      uint64 // MemTotal (kB)
	MemUsed  uint64 // MemTotal - MemAvailable (kB)
	MemFree  uint64 // MemAvailable (kB)
	Swap     uint64 // swapfile total (kB)
	SwapUsed uint64 // swapfile used (kB)
	SwapFree uint64 // swapfile free (kB)
}

func (s Stats) Total() uint64 {
	return s.Mem + s.Swap
}

func (s Stats) TotalUsed() uint64 {
	return s.MemUsed + s.SwapUsed
}

func (s Stats) Percent() float64 {
	total := s.Total()
	if total == 0 {
		return 0
	}
	return float64(s.TotalUsed()) * 100 / float64(total)
}

func (s Stats) TotalUsedInGB() float64 {
	return float64(s.TotalUsed()) / 1024 / 1024
}

type Monitor struct {
	interval time.Duration
}

func New(interval time.Duration) *Monitor {
	return &Monitor{interval: interval}
}

func (r *Monitor) Read() Stats {
	memTotal, memAvailable := r.readMemInfo()
	swapTotal, swapUsed := r.readSwapFile()

	return Stats{
		Mem:      memTotal,
		MemUsed:  memTotal - memAvailable,
		MemFree:  memAvailable,
		Swap:     swapTotal,
		SwapUsed: swapUsed,
		SwapFree: swapTotal - swapUsed,
	}
}

func (r *Monitor) Listen() <-chan Stats {
	ch := make(chan Stats)
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		defer close(ch)

		ch <- r.Read()

		for range ticker.C {
			ch <- r.Read()
		}
	}()
	return ch
}

func (r *Monitor) readMemInfo() (total, available uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemAvailable:":
			available = val
		}

		if total > 0 && available > 0 {
			break
		}
	}

	return total, available
}

func (r *Monitor) readSwapFile() (total, used uint64) {
	f, err := os.Open("/proc/swaps")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		// only count swapfile, not zram
		if fields[1] != "file" {
			continue
		}
		size, _ := strconv.ParseUint(fields[2], 10, 64)
		usedVal, _ := strconv.ParseUint(fields[3], 10, 64)
		total += size
		used += usedVal
	}

	return total, used
}

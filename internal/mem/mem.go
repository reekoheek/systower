package mem

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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

type Reader struct {
	memTotal uint64
}

func New() *Reader {
	r := &Reader{}
	r.memTotal = r.readMemTotal()
	return r
}

func (r *Reader) Read() Stats {
	memAvailable := r.readMemAvailable()
	swapTotal, swapUsed := r.readSwapFile()

	return Stats{
		Mem:      r.memTotal,
		MemUsed:  r.memTotal - memAvailable,
		MemFree:  memAvailable,
		Swap:     swapTotal,
		SwapUsed: swapUsed,
		SwapFree: swapTotal - swapUsed,
	}
}

func (r *Reader) readMemTotal() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			return val
		}
	}
	return 0
}

func (r *Reader) readMemAvailable() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemAvailable:" {
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			return val
		}
	}
	return 0
}

func (r *Reader) readSwapFile() (total, used uint64) {
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

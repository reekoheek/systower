package mem

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Stats struct {
	All      uint64 // Mem + Swap (kB)
	AllUsed  uint64 // MemUsed + SwapUsed (kB)
	AllFree  uint64 // MemFree + SwapFree (kB)
	Mem      uint64 // MemTotal (kB)
	MemUsed  uint64 // MemTotal - MemAvailable (kB)
	MemFree  uint64 // MemAvailable (kB)
	Swap     uint64 // swapfile total (kB)
	SwapUsed uint64 // swapfile used (kB)
	SwapFree uint64 // swapfile free (kB)
}

type Reader struct{}

func New() *Reader {
	return &Reader{}
}

func (r *Reader) Read() Stats {
	memTotal, memAvailable := r.readMemInfo()
	swapTotal, swapUsed := r.readSwapFile()

	memUsed := memTotal - memAvailable
	swapFree := swapTotal - swapUsed

	return Stats{
		All:      memTotal + swapTotal,
		AllUsed:  memUsed + swapUsed,
		AllFree:  memAvailable + swapFree,
		Mem:      memTotal,
		MemUsed:  memUsed,
		MemFree:  memAvailable,
		Swap:     swapTotal,
		SwapUsed: swapUsed,
		SwapFree: swapFree,
	}
}

func (r *Reader) readMemInfo() (total, available uint64) {
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

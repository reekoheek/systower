package cpu

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Stats struct {
	Idle  uint64
	Total uint64
}

func (s Stats) Percent() float64 {
	if s.Total == 0 {
		return 0
	}
	return 100 - float64(s.Idle)*100/float64(s.Total)
}

type Reader struct{}

func New() *Reader {
	return &Reader{}
}

func (r *Reader) Read() Stats {
	return r.readProcStat()
}

func (r *Reader) readProcStat() Stats {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return Stats{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return Stats{}
	}

	// cpu  user nice system idle iowait irq softirq steal guest guest_nice
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return Stats{}
	}

	var stats Stats
	// user nice system idle iowait irq softirq steal | guest guest_nice
	// guest/guest_nice already included in user/nice, skip them
	for i, field := range fields[1:] {
		if i >= 8 {
			break
		}
		val, _ := strconv.ParseUint(field, 10, 64)
		stats.Total += val
		if i == 3 || i == 4 { // idle + iowait
			stats.Idle += val
		}
	}

	return stats
}

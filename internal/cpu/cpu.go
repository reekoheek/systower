package cpu

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
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

type Monitor struct {
	prevIdle  uint64
	prevTotal uint64
}

func New() *Monitor {
	return &Monitor{}
}

func (r *Monitor) Read() Stats {
	idle, total := r.readProcStat()

	deltaIdle := idle - r.prevIdle
	deltaTotal := total - r.prevTotal

	r.prevIdle = idle
	r.prevTotal = total

	return Stats{
		Idle:  deltaIdle,
		Total: deltaTotal,
	}
}

func (r *Monitor) Listen(interval time.Duration) <-chan Stats {
	ch := make(chan Stats)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(ch)

		ch <- r.Read()

		for range ticker.C {
			ch <- r.Read()
		}
	}()
	return ch
}

func (r *Monitor) readProcStat() (idle, total uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, 0
	}

	// cpu  user nice system idle iowait irq softirq steal guest guest_nice
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0
	}

	// user nice system idle iowait irq softirq steal | guest guest_nice
	// guest/guest_nice already included in user/nice, skip them
	for i, field := range fields[1:] {
		if i >= 8 {
			break
		}
		val, _ := strconv.ParseUint(field, 10, 64)
		total += val
		if i == 3 || i == 4 { // idle + iowait
			idle += val
		}
	}

	return idle, total
}

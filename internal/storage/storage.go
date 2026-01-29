//go:build linux

package storage

import (
	"syscall"
	"time"
)

type Stats struct {
	Total uint64 // bytes
	Used  uint64 // bytes
	Free  uint64 // bytes
}

func (s Stats) Percent() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Used) * 100 / float64(s.Total)
}

type Monitor struct {
	path string
}

func New(path string) *Monitor {
	return &Monitor{path: path}
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

func (r *Monitor) Read() Stats {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(r.path, &stat); err != nil {
		return Stats{}
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	return Stats{
		Total: total,
		Used:  used,
		Free:  free,
	}
}

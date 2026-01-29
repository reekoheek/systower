//go:build linux

package storage

import "syscall"

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

type Reader struct {
	path string
}

func New(path string) *Reader {
	return &Reader{path: path}
}

func (r *Reader) Read() Stats {
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

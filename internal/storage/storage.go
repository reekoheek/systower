//go:build linux

package storage

import "syscall"

type Stats struct {
	Total uint64 // bytes
	Used  uint64 // bytes
	Free  uint64 // bytes
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

//go:build linux

package poller

import (
	"time"

	"github.com/reekoheek/systower/internal/cpu"
	"github.com/reekoheek/systower/internal/mem"
	"github.com/reekoheek/systower/internal/storage"
)

type Stats struct {
	CPU     cpu.Stats
	Mem     mem.Stats
	Storage storage.Stats
}

type Poller struct {
	cpuInterval     time.Duration
	memInterval     time.Duration
	storageInterval time.Duration
}

func New(cpuInterval, memInterval, storageInterval time.Duration) *Poller {
	return &Poller{
		cpuInterval:     cpuInterval,
		memInterval:     memInterval,
		storageInterval: storageInterval,
	}
}

func (p *Poller) Listen() <-chan Stats {
	ch := make(chan Stats)

	go func() {
		cpuReader := cpu.New()
		memReader := mem.New()
		storageReader := storage.New("/")

		baseInterval := time.Second

		cpuMult := max(1, int(p.cpuInterval/baseInterval))
		memMult := max(1, int(p.memInterval/baseInterval))
		storageMult := max(1, int(p.storageInterval/baseInterval))

		ticker := time.NewTicker(baseInterval)
		defer ticker.Stop()
		defer close(ch)

		var stats, prevStats Stats
		tick := 0

		// Initial read
		stats.CPU = cpuReader.Read()
		stats.Mem = memReader.Read()
		stats.Storage = storageReader.Read()
		ch <- stats
		prevStats = stats

		for range ticker.C {
			if tick%cpuMult == 0 {
				stats.CPU = cpuReader.Read()
			}

			if tick%memMult == 0 {
				stats.Mem = memReader.Read()
			}

			if tick%storageMult == 0 {
				stats.Storage = storageReader.Read()
			}

			if stats != prevStats {
				ch <- stats
				prevStats = stats
			}

			tick++
			if tick >= max(cpuMult, max(memMult, storageMult)) {
				tick = 0
			}
		}
	}()

	return ch
}

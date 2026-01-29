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
	interval    time.Duration
	storagePath string
}

func New(interval time.Duration, storagePath string) *Poller {
	return &Poller{
		interval:    interval,
		storagePath: storagePath,
	}
}

func (p *Poller) Start() <-chan Stats {
	ch := make(chan Stats)

	go func() {
		cpuReader := cpu.New()
		memReader := mem.New()
		storageReader := storage.New(p.storagePath)

		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		defer close(ch)

		for range ticker.C {
			ch <- Stats{
				CPU:     cpuReader.Read(),
				Mem:     memReader.Read(),
				Storage: storageReader.Read(),
			}
		}
	}()

	return ch
}

package systower

import (
	"fmt"
	"os"
	"time"

	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/cpu"
	"github.com/reekoheek/systower/internal/mem"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/poller"
	"github.com/reekoheek/systower/internal/storage"
	"github.com/reekoheek/systower/internal/sys"
)

type Stats struct {
	Caffeine caffeine.Stats
	Battery  battery.Stats
	CPU      cpu.Stats
	Mem      mem.Stats
	Storage  storage.Stats
}

type Systower struct {
	caff      *caffeine.Caffeine
	notif     *notification.Notification
	batMon    *battery.Monitor
	sysMgr    *sys.Sys
	poller    *poller.Poller
	stats     Stats
	prevStats Stats
	dirty     bool
}

func New(caff *caffeine.Caffeine, notif *notification.Notification, batMon *battery.Monitor, sysMgr *sys.Sys, poller *poller.Poller) *Systower {
	return &Systower{caff: caff, notif: notif, batMon: batMon, sysMgr: sysMgr, poller: poller}
}

func (s *Systower) Watch() {
	batCh, err := s.batMon.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	caffCh, err := s.caff.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pollerCh := s.poller.Listen()

	for {
		select {
		case info, ok := <-batCh:
			if !ok {
				return
			}
			s.stats.Battery = info
			s.dirty = true

			if info.Status != "charging" {
				if s.caff.Status().Active && info.Percent < 15 {
					s.caff.Off()
					s.notif.Send("Low battery, disable caffeine")
				}
				if info.Percent < 5 {
					s.notif.Send("Battery almost drained, have a nice sleep")
					time.Sleep(5 * time.Second)
					s.sysMgr.Poweroff()
				}
			}
		case status, ok := <-caffCh:
			if !ok {
				return
			}
			s.stats.Caffeine = status
			s.dirty = true
		case stats, ok := <-pollerCh:
			if !ok {
				return
			}
			s.stats.CPU = stats.CPU
			s.stats.Mem = stats.Mem
			s.stats.Storage = stats.Storage
			s.dirty = true
		}
		s.printIfChanged()
	}
}

func (s *Systower) Stats() Stats {
	return s.stats
}

func (s *Systower) printIfChanged() {
	if !s.dirty || s.stats == s.prevStats {
		return
	}
	s.prevStats = s.stats
	s.dirty = false
	s.print()
}

func (s *Systower) print() {
	fmt.Printf("caffeine|bool|%t\n", s.stats.Caffeine.Active)
	fmt.Printf("bat_status|string|%s\n", s.stats.Battery.Status)
	fmt.Printf("bat_percent|int|%d\n", s.stats.Battery.Percent)
	fmt.Printf("bat_estimate|string|%s\n", s.stats.Battery.Estimate)
	fmt.Printf("cpu_percent|float|%.5f\n", s.stats.CPU.Percent())
	fmt.Printf("mem_used|float|%.5f\n", s.stats.Mem.TotalUsedInGB())
	fmt.Printf("mem_percent|float|%.5f\n", s.stats.Mem.Percent())
	fmt.Printf("storage_percent|float|%.5f\n", s.stats.Storage.Percent())
	fmt.Println()
}

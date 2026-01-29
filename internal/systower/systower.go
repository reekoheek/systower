package systower

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/clock"
	"github.com/reekoheek/systower/internal/cpu"
	"github.com/reekoheek/systower/internal/mem"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/storage"
	"github.com/reekoheek/systower/internal/sys"
)

type Stats struct {
	Clock    clock.Stats
	Caffeine string
	Battery  battery.Stats
	CPU      cpu.Stats
	Mem      mem.Stats
	Storage  storage.Stats
}

type Systower struct {
	conn       *dbus.Conn
	sysConn    *dbus.Conn
	caff       *caffeine.Caffeine
	notif      *notification.Notification
	batMon     *battery.Monitor
	sysMgr     *sys.Sys
	clockMon   *clock.Monitor
	cpuMon     *cpu.Monitor
	memMon     *mem.Monitor
	storageMon *storage.Monitor
	stats      Stats
}

func New(clockInterval, cpuInterval, memInterval, storageInterval time.Duration) (*Systower, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("session bus: %w", err)
	}

	sysConn, err := dbus.SystemBus()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("system bus: %w", err)
	}

	return &Systower{
		conn:       conn,
		sysConn:    sysConn,
		caff:       caffeine.New(conn, caffeine.DetectAdapter()),
		notif:      notification.New(conn, "Systower"),
		batMon:     battery.New(sysConn),
		sysMgr:     sys.New(sysConn),
		clockMon:   clock.New(clockInterval),
		cpuMon:     cpu.New(cpuInterval),
		memMon:     mem.New(memInterval),
		storageMon: storage.New("/", storageInterval),
	}, nil
}

func (s *Systower) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
	if s.sysConn != nil {
		s.sysConn.Close()
	}
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

	clockCh := s.clockMon.Listen()
	cpuCh := s.cpuMon.Listen()
	memCh := s.memMon.Listen()
	storageCh := s.storageMon.Listen()

	debounce := time.NewTimer(0)
	<-debounce.C // drain initial tick

	var lastOutput string

	for {
		select {
		case info, ok := <-batCh:
			if !ok {
				return
			}
			s.stats.Battery = info

			if info.Status != "charging" {
				if s.caff.Read() == "on" && info.Percent < 15 {
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
		case stats, ok := <-clockCh:
			if !ok {
				return
			}
			s.stats.Clock = stats
		case stats, ok := <-cpuCh:
			if !ok {
				return
			}
			s.stats.CPU = stats
		case stats, ok := <-memCh:
			if !ok {
				return
			}
			s.stats.Mem = stats
		case stats, ok := <-storageCh:
			if !ok {
				return
			}
			s.stats.Storage = stats
		case <-debounce.C:
			if output := s.output(); output != lastOutput {
				lastOutput = output
				os.Stdout.WriteString(output)
			}
			continue
		}
		debounce.Reset(100 * time.Millisecond)
	}
}

func (s *Systower) Stats() Stats {
	return s.stats
}

func (s *Systower) output() string {
	var b strings.Builder
	fmt.Fprintf(&b, "clock_date|string|%s\n", s.stats.Clock.Date())
	fmt.Fprintf(&b, "clock_time|string|%s\n", s.stats.Clock.TimeStr())
	fmt.Fprintf(&b, "caffeine|string|%s\n", s.stats.Caffeine)
	fmt.Fprintf(&b, "bat_status|string|%s\n", s.stats.Battery.Status)
	fmt.Fprintf(&b, "bat_percent|int|%d\n", s.stats.Battery.Percent)
	fmt.Fprintf(&b, "bat_estimate|string|%s\n", s.stats.Battery.Estimate)
	fmt.Fprintf(&b, "cpu_percent|float|%.5f\n", s.stats.CPU.Percent())
	fmt.Fprintf(&b, "mem_used|float|%.5f\n", s.stats.Mem.TotalUsedInGB())
	fmt.Fprintf(&b, "mem_percent|float|%.5f\n", s.stats.Mem.Percent())
	fmt.Fprintf(&b, "storage_percent|float|%.5f\n", s.stats.Storage.Percent())
	b.WriteByte('\n')
	return b.String()
}

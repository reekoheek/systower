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
	"github.com/reekoheek/systower/internal/poller"
	"github.com/reekoheek/systower/internal/storage"
	"github.com/reekoheek/systower/internal/sys"
)

type updateKind int

const (
	updateBattery updateKind = iota
	updateCaffeine
	updateClock
	updateCPU
	updateMem
	updateStorage
)

type update struct {
	kind     updateKind
	battery  battery.Stats
	caffeine string
	clock    clock.Stats
	cpu      cpu.Stats
	mem      mem.Stats
	storage  storage.Stats
}

type Stats struct {
	Clock    clock.Stats
	Caffeine string
	Battery  battery.Stats
	CPU      cpu.Stats
	Mem      mem.Stats
	Storage  storage.Stats
}

type Intervals struct {
	Clock    time.Duration
	CPU      time.Duration
	Mem      time.Duration
	Storage  time.Duration
	Debounce time.Duration
}

type Systower struct {
	conn    *dbus.Conn
	sysConn *dbus.Conn
	caff    *caffeine.Caffeine
	notif   *notification.Notification
	batMon  *battery.Monitor
	sysMgr  *sys.Sys
	stats   Stats

	clockReader   *clock.Reader
	cpuReader     *cpu.Reader
	memReader     *mem.Reader
	storageReader *storage.Reader

	intervals Intervals
}

func New(intervals Intervals) (*Systower, error) {
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
		conn:          conn,
		sysConn:       sysConn,
		caff:          caffeine.New(conn, caffeine.DetectAdapter()),
		notif:         notification.New(conn, "Systower"),
		batMon:        battery.New(sysConn),
		sysMgr:        sys.New(sysConn),
		clockReader:   clock.New(),
		cpuReader:     cpu.New(),
		memReader:     mem.New(),
		storageReader: storage.New("/"),
		intervals:     intervals,
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
	updates := make(chan update, 10)

	// Battery monitor
	if err := s.batMon.Listen(func(info battery.Stats) {
		updates <- update{kind: updateBattery, battery: info}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Caffeine monitor
	if err := s.caff.Listen(func(status string) {
		updates <- update{kind: updateCaffeine, caffeine: status}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Polling readers - send updates via channel
	p := poller.New()
	p.Register(s.intervals.Clock, func() {
		updates <- update{kind: updateClock, clock: s.clockReader.Read()}
	})
	p.Register(s.intervals.CPU, func() {
		updates <- update{kind: updateCPU, cpu: s.cpuReader.Read()}
	})
	p.Register(s.intervals.Mem, func() {
		updates <- update{kind: updateMem, mem: s.memReader.Read()}
	})
	p.Register(s.intervals.Storage, func() {
		updates <- update{kind: updateStorage, storage: s.storageReader.Read()}
	})
	p.Poll()

	// Main loop - single goroutine owns stats
	var (
		lastOutput string
		timer      *time.Timer
		timerC     <-chan time.Time
		pending    bool
	)

	for {
		select {
		case u := <-updates:
			s.applyUpdate(u)
			// Reset debounce timer
			if timer == nil {
				timer = time.NewTimer(s.intervals.Debounce)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timerC:
					default:
					}
				}
				timer.Reset(s.intervals.Debounce)
			}
			pending = true

		case <-timerC:
			if pending {
				if output := s.output(); output != lastOutput {
					lastOutput = output
					os.Stdout.WriteString(output)
				}
				pending = false
			}
			timer = nil
			timerC = nil
		}
	}
}

func (s *Systower) applyUpdate(u update) {
	switch u.kind {
	case updateBattery:
		s.stats.Battery = u.battery
		s.handleBatteryLogic(u.battery)
	case updateCaffeine:
		s.stats.Caffeine = u.caffeine
	case updateClock:
		s.stats.Clock = u.clock
	case updateCPU:
		s.stats.CPU = u.cpu
	case updateMem:
		s.stats.Mem = u.mem
	case updateStorage:
		s.stats.Storage = u.storage
	}
}

func (s *Systower) handleBatteryLogic(info battery.Stats) {
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
}

func (s *Systower) Stats() Stats {
	return s.stats
}

func (s *Systower) output() string {
	var b strings.Builder
	fmt.Fprintf(&b, "clock_day|string|%s\n", s.stats.Clock.Day())
	fmt.Fprintf(&b, "clock_date|string|%s\n", s.stats.Clock.Date())
	fmt.Fprintf(&b, "clock_time|string|%s\n", s.stats.Clock.Time())
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

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
	events  *eventBus

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

	s := &Systower{
		conn:          conn,
		sysConn:       sysConn,
		caff:          caffeine.New(conn, caffeine.DetectAdapter()),
		notif:         notification.New(conn, "Systower"),
		batMon:        battery.New(sysConn),
		sysMgr:        sys.New(sysConn),
		events:        newEventBus(),
		clockReader:   clock.New(),
		cpuReader:     cpu.New(),
		memReader:     mem.New(),
		storageReader: storage.New("/"),
		intervals:     intervals,
	}

	s.On(BatteryUpdated, s.onBatteryUpdated)

	return s, nil
}

func (s *Systower) On(kind EventKind, h EventHandler) {
	s.events.on(kind, h)
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
	events := make(chan Event, 10)

	// Battery monitor
	if err := s.batMon.Listen(func(info battery.Stats) {
		events <- Event{Kind: BatteryUpdated, Payload: info}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Caffeine monitor
	if err := s.caff.Listen(func(status string) {
		events <- Event{Kind: CaffeineUpdated, Payload: status}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Polling readers - send events via channel
	p := poller.New()
	p.Register(s.intervals.Clock, func() {
		events <- Event{Kind: ClockUpdated, Payload: s.clockReader.Read()}
	})
	p.Register(s.intervals.CPU, func() {
		events <- Event{Kind: CPUUpdated, Payload: s.cpuReader.Read()}
	})
	p.Register(s.intervals.Mem, func() {
		events <- Event{Kind: MemUpdated, Payload: s.memReader.Read()}
	})
	p.Register(s.intervals.Storage, func() {
		events <- Event{Kind: StorageUpdated, Payload: s.storageReader.Read()}
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
		case e := <-events:
			s.dispatch(e)
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

func (s *Systower) dispatch(e Event) {
	switch e.Kind {
	case BatteryUpdated:
		s.stats.Battery = e.Payload.(battery.Stats)
	case CaffeineUpdated:
		s.stats.Caffeine = e.Payload.(string)
	case ClockUpdated:
		s.stats.Clock = e.Payload.(clock.Stats)
	case CPUUpdated:
		s.stats.CPU = e.Payload.(cpu.Stats)
	case MemUpdated:
		s.stats.Mem = e.Payload.(mem.Stats)
	case StorageUpdated:
		s.stats.Storage = e.Payload.(storage.Stats)
	}

	s.events.publish(e)
}

func (s *Systower) onBatteryUpdated(e Event) {
	info := e.Payload.(battery.Stats)
	if info.Status != "charging" {
		if s.caff.Read() == "on" && info.Percent < 15 {
			s.caff.Off()
			s.notif.Send("Low battery, disable caffeine")
		}
		if info.Percent < 5 {
			s.notif.Send("Battery almost drained, have a nice sleep")
			go func() {
				time.Sleep(5 * time.Second)
				s.sysMgr.Poweroff()
			}()
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

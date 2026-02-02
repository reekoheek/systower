package systower

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/backlight"
	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/clock"
	"github.com/reekoheek/systower/internal/cpu"
	"github.com/reekoheek/systower/internal/mem"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/poller"
	"github.com/reekoheek/systower/internal/storage"
	"github.com/reekoheek/systower/internal/sys"
	"github.com/reekoheek/systower/internal/volume"
)

type Stats struct {
	Backlight backlight.Stats
	Clock     clock.Stats
	Caffeine  string
	Battery   battery.Stats
	CPU       cpu.Stats
	Mem       mem.Stats
	Storage   storage.Stats
	Volume    volume.Stats
}

type Intervals struct {
	Clock    time.Duration
	CPU      time.Duration
	Mem      time.Duration
	Storage  time.Duration
	Throttle time.Duration
}

type Systower struct {
	conn    *dbus.Conn
	sysConn *dbus.Conn
	caff    *caffeine.Caffeine
	notif   *notification.Notification
	blMon   *backlight.Monitor
	batMon  *battery.Monitor
	volMon  *volume.Monitor
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

	volMon, err := volume.New()
	if err != nil {
		conn.Close()
		sysConn.Close()
		return nil, fmt.Errorf("volume monitor: %w", err)
	}

	// Backlight monitor is optional (may not exist on desktops)
	blMon, _ := backlight.New("")

	s := &Systower{
		conn:          conn,
		sysConn:       sysConn,
		caff:          caffeine.New(conn, caffeine.DetectAdapter()),
		notif:         notification.New(conn, "Systower"),
		blMon:         blMon,
		batMon:        battery.New(sysConn),
		volMon:        volMon,
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
	if s.blMon != nil {
		s.blMon.Close()
	}
	if s.volMon != nil {
		s.volMon.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
	if s.sysConn != nil {
		s.sysConn.Close()
	}
}

func (s *Systower) Watch(ctx context.Context) {
	events := make(chan Event, 16)

	// Backlight monitor (optional)
	if s.blMon != nil {
		s.blMon.Listen(ctx, func(stats backlight.Stats) {
			events <- Event{Kind: BacklightUpdated, Payload: stats}
		})
	}

	// Battery monitor
	if err := s.batMon.Listen(ctx, func(info battery.Stats) {
		events <- Event{Kind: BatteryUpdated, Payload: info}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Caffeine monitor
	if err := s.caff.Listen(ctx, func(status string) {
		events <- Event{Kind: CaffeineUpdated, Payload: status}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Volume monitor
	if err := s.volMon.Listen(ctx, func(stats volume.Stats) {
		events <- Event{Kind: VolumeUpdated, Payload: stats}
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
	p.Poll(ctx)

	// Main loop - single goroutine owns stats
	var (
		lastStats Stats
		timer     *time.Timer
		timerC    <-chan time.Time
	)

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case e := <-events:
			s.dispatch(e)
			// Throttle: start timer only if not already pending
			if timer == nil {
				timer = time.NewTimer(s.intervals.Throttle)
				timerC = timer.C
			}
		case <-timerC:
			if s.stats != lastStats {
				lastStats = s.stats
				os.Stdout.WriteString(s.output())
			}
			timer = nil
			timerC = nil
		}
	}
}

func (s *Systower) dispatch(e Event) {
	switch e.Kind {
	case BacklightUpdated:
		s.stats.Backlight = e.Payload.(backlight.Stats)
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
	case VolumeUpdated:
		s.stats.Volume = e.Payload.(volume.Stats)
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
	fmt.Fprintf(&b, "backlight_percent|int|%d\n", s.stats.Backlight.Percent())
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
	fmt.Fprintf(&b, "vol_percent|int|%d\n", s.stats.Volume.Percent)
	fmt.Fprintf(&b, "vol_muted|bool|%t\n", s.stats.Volume.Muted)
	b.WriteByte('\n')
	return b.String()
}

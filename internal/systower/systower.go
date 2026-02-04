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
	"github.com/reekoheek/systower/internal/storage"
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
	Clock   uint64
	CPU     uint64
	Mem     uint64
	Storage uint64
}

// gcd calculates greatest common divisor of two numbers
func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// baseInterval calculates the optimal ticker interval using GCD
func (i Intervals) baseInterval() time.Duration {
	result := i.Clock
	result = gcd(result, i.CPU)
	result = gcd(result, i.Mem)
	result = gcd(result, i.Storage)
	if result == 0 {
		result = 1
	}
	return time.Duration(result) * time.Second
}

type Systower struct {
	conn      *dbus.Conn
	sysConn   *dbus.Conn
	caff      *caffeine.Caffeine
	blMon     *backlight.Monitor
	batMon    *battery.Monitor
	volMon    *volume.Monitor
	stats     Stats
	intervals Intervals

	clockReader   *clock.Reader
	cpuReader     *cpu.Reader
	memReader     *mem.Reader
	storageReader *storage.Reader
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

	blMon, err := backlight.New("")
	if err != nil {
		conn.Close()
		sysConn.Close()
		return nil, fmt.Errorf("backlight monitor: %w", err)
	}

	s := &Systower{
		conn:          conn,
		sysConn:       sysConn,
		caff:          caffeine.New(conn, caffeine.DetectAdapter()),
		blMon:         blMon,
		batMon:        battery.New(sysConn),
		volMon:        volume.New(),
		intervals:     intervals,
		clockReader:   clock.New(),
		cpuReader:     cpu.New(),
		memReader:     mem.New(),
		storageReader: storage.New("/"),
	}

	return s, nil
}

func (s *Systower) Watch(ctx context.Context) error {
	// Cleanup when context is cancelled
	context.AfterFunc(ctx, func() {
		s.conn.Close()
		s.sysConn.Close()
	})

	// Start all monitors - they return channels
	blCh := s.blMon.Listen(ctx)

	batCh, err := s.batMon.Listen(ctx)
	if err != nil {
		return fmt.Errorf("battery monitor: %w", err)
	}

	caffCh, err := s.caff.Listen(ctx)
	if err != nil {
		return fmt.Errorf("caffeine monitor: %w", err)
	}

	volCh, err := s.volMon.Listen(ctx)
	if err != nil {
		return fmt.Errorf("volume monitor: %w", err)
	}

	// Calculate optimal ticker interval using GCD of all intervals
	baseInterval := s.intervals.baseInterval()
	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	// Convert intervals to tick counts based on base interval
	baseSeconds := uint64(baseInterval / time.Second)
	clockTicks := s.intervals.Clock / baseSeconds
	cpuTicks := s.intervals.CPU / baseSeconds
	memTicks := s.intervals.Mem / baseSeconds
	storageTicks := s.intervals.Storage / baseSeconds

	var tickCount uint64

	// Read initial values
	s.stats.Clock = s.clockReader.Read()
	s.stats.CPU = s.cpuReader.Read()
	s.stats.Mem = s.memReader.Read()
	s.stats.Storage = s.storageReader.Read()
	s.stats.Caffeine = s.caff.Read()
	s.stats.Battery = s.batMon.Read(nil)

	var lastStats Stats

	for {
		select {
		case <-ctx.Done():
			return nil

		case stats := <-blCh:
			s.stats.Backlight = stats

		case sig := <-batCh:
			s.stats.Battery = s.batMon.Read(sig)
			s.onBatteryUpdated(s.stats.Battery)

		case <-caffCh:
			s.stats.Caffeine = s.caff.Read()

		case stats := <-volCh:
			s.stats.Volume = stats

		case <-ticker.C:
			tickCount++
			if tickCount%clockTicks == 0 {
				s.stats.Clock = s.clockReader.Read()
			}
			if tickCount%cpuTicks == 0 {
				s.stats.CPU = s.cpuReader.Read()
			}
			if tickCount%memTicks == 0 {
				s.stats.Mem = s.memReader.Read()
			}
			if tickCount%storageTicks == 0 {
				s.stats.Storage = s.storageReader.Read()
			}
		}

		if s.stats != lastStats {
			lastStats = s.stats
			os.Stdout.WriteString(s.output())
		}
	}
}

func (s *Systower) onBatteryUpdated(info battery.Stats) {
	if info.Status != "charging" {
		if s.caff.Read() == "on" && info.Percent < 15 {
			s.caff.Off()
			s.notify("Low battery, disable caffeine")
		}
		if info.Percent <= 5 {
			s.notify("Battery almost drained, have a nice sleep")
			time.Sleep(5 * time.Second)
			s.poweroff()
		}
	}
}

func (s *Systower) notify(body string) {
	s.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").
		Call("org.freedesktop.Notifications.Notify", 0,
			"systower", uint32(0), "", "Systower", body,
			[]string{}, map[string]dbus.Variant{}, int32(-1))
}

func (s *Systower) poweroff() {
	s.sysConn.Object("org.freedesktop.login1", "/org/freedesktop/login1").
		Call("org.freedesktop.login1.Manager.PowerOff", 0, false)
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
	fmt.Fprintf(&b, "cpu_percent|int|%d\n", s.stats.CPU.Percent())
	fmt.Fprintf(&b, "mem_used|int|%d\n", s.stats.Mem.TotalUsedInGB())
	fmt.Fprintf(&b, "mem_percent|int|%d\n", s.stats.Mem.Percent())
	fmt.Fprintf(&b, "storage_percent|int|%d\n", s.stats.Storage.Percent())
	fmt.Fprintf(&b, "vol_percent|int|%d\n", s.stats.Volume.Percent)
	fmt.Fprintf(&b, "vol_muted|bool|%t\n", s.stats.Volume.Muted)
	b.WriteByte('\n')
	return b.String()
}

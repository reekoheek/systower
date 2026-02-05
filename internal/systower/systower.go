package systower

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/backlight"
	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/clock"
	"github.com/reekoheek/systower/internal/cpu"
	"github.com/reekoheek/systower/internal/mem"
	"github.com/reekoheek/systower/internal/network"
	"github.com/reekoheek/systower/internal/poller"
	"github.com/reekoheek/systower/internal/storage"
	"github.com/reekoheek/systower/internal/volume"
)

type Stats struct {
	Backlight backlight.Stats
	Clock     clock.Stats
	Caffeine  caffeine.Stats
	Battery   battery.Stats
	CPU       cpu.Stats
	Mem       mem.Stats
	Storage   storage.Stats
	Volume    volume.Stats
	Network   network.Stats
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
	sysConn   *dbus.Conn
	caff      *caffeine.Caffeine
	blMon     *backlight.Monitor
	batMon    *battery.Monitor
	volMon    *volume.Monitor
	netMon    *network.Monitor
	stats     Stats
	intervals Intervals

	clockReader   *clock.Reader
	cpuReader     *cpu.Reader
	memReader     *mem.Reader
	storageReader *storage.Reader
}

func New(intervals Intervals) (*Systower, error) {
	sysConn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("system bus: %w", err)
	}

	blMon, err := backlight.New("")
	if err != nil {
		sysConn.Close()
		return nil, fmt.Errorf("backlight monitor: %w", err)
	}

	s := &Systower{
		sysConn:       sysConn,
		caff:          caffeine.New(caffeine.DetectAdapter()),
		blMon:         blMon,
		batMon:        battery.New(sysConn),
		volMon:        volume.New(),
		netMon:        network.New(sysConn),
		intervals:     intervals,
		clockReader:   clock.New(),
		cpuReader:     cpu.New(),
		memReader:     mem.New(),
		storageReader: storage.New("/"),
	}

	return s, nil
}

func (s *Systower) Watch(ctx context.Context) error {
	// Initialize unified poller with base interval
	baseInterval := s.intervals.baseInterval()
	poll, err := poller.New(baseInterval)
	if err != nil {
		return fmt.Errorf("poller: %w", err)
	}

	// Cleanup when context is cancelled
	context.AfterFunc(ctx, func() {
		poll.Cancel() // Wake up poll to exit
	})

	defer func() {
		poll.Close()
		s.sysConn.Close()
		s.blMon.Close()
		s.caff.Close()
		s.volMon.Close()
	}()

	// Initialize backlight fd and add to poller
	blFd, err := s.blMon.InitFd()
	if err != nil {
		return fmt.Errorf("backlight fd: %w", err)
	}
	poll.AddSource(blFd, poller.EventBacklight)

	// Initialize caffeine fd and add to poller
	caffFd, err := s.caff.InitFd()
	if err != nil {
		return fmt.Errorf("caffeine fd: %w", err)
	}
	poll.AddSource(caffFd, poller.EventCaffeine)

	// Initialize volume fd and add to poller
	volFd, err := s.volMon.InitFd()
	if err != nil {
		return fmt.Errorf("volume fd: %w", err)
	}
	poll.AddSource(volFd, poller.EventVolume)

	// Build poll fds array
	poll.Init()

	// Setup single D-Bus signal channel for battery and network
	dbusCh := make(chan *dbus.Signal, 10)
	s.sysConn.Signal(dbusCh)

	// Initialize battery and add match rules
	batRule, err := s.batMon.Init()
	if err != nil {
		return fmt.Errorf("battery init: %w", err)
	}
	if err := s.sysConn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, batRule).Err; err != nil {
		return fmt.Errorf("battery match rule: %w", err)
	}

	// Initialize network and add match rules
	netRules, err := s.netMon.Init()
	if err != nil {
		return fmt.Errorf("network init: %w", err)
	}
	for _, rule := range netRules {
		if err := s.sysConn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Err; err != nil {
			return fmt.Errorf("network match rule: %w", err)
		}
	}

	// Convert intervals to tick counts based on base interval
	baseSeconds := uint64(baseInterval / time.Second)
	clockTicks := s.intervals.Clock / baseSeconds
	cpuTicks := s.intervals.CPU / baseSeconds
	memTicks := s.intervals.Mem / baseSeconds
	storageTicks := s.intervals.Storage / baseSeconds

	var tickCount uint64

	// Read initial values
	s.stats.Backlight = s.blMon.Read()
	s.stats.Clock = s.clockReader.Read()
	s.stats.CPU = s.cpuReader.Read()
	s.stats.Mem = s.memReader.Read()
	s.stats.Storage = s.storageReader.Read()
	s.stats.Caffeine = s.caff.Read()
	s.stats.Battery = s.batMon.Read(nil)
	s.stats.Network = s.netMon.Read(nil)
	s.stats.Volume = s.volMon.Stats()

	var lastStats Stats

	for {
		// Block on poll - wakes up on timer or any event
		events, cancelled := poll.Wait()
		if cancelled {
			return nil
		}

		// Process poll events
		for _, ev := range events {
			switch ev.Type {
			case poller.EventTicker:
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

			case poller.EventBacklight:
				if s.blMon.Drain() {
					s.stats.Backlight = s.blMon.Read()
				}

			case poller.EventCaffeine:
				s.caff.Drain()
				s.stats.Caffeine = s.caff.Read()

			case poller.EventVolume:
				if s.volMon.Drain() {
					s.stats.Volume = s.volMon.Stats()
				}
			}
		}

		// Drain D-Bus signals (non-blocking)
		// D-Bus signals may be delayed up to 1 tick interval, which is acceptable
		for {
			select {
			case sig := <-dbusCh:
				if sig.Path == s.batMon.Path() {
					s.stats.Battery = s.batMon.Read(sig)
					s.onBatteryUpdated(s.stats.Battery)
				} else if sig.Path == s.netMon.WifiPath() || sig.Path == s.netMon.EthPath() {
					s.stats.Network = s.netMon.Read(sig)
				}
			default:
				goto dbusDone
			}
		}
	dbusDone:

		if s.stats != lastStats {
			lastStats = s.stats
			os.Stdout.WriteString(s.output())
		}
	}
}

func (s *Systower) onBatteryUpdated(info battery.Stats) {
	if info.Status != "charging" {
		if s.caff.Read().Active && info.Percent < 15 {
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
	exec.Command("notify-send", "Systower", body).Run()
}

func (s *Systower) poweroff() {
	exec.Command("systemctl", "poweroff").Run()
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
	fmt.Fprintf(&b, "wifi_state|bool|%t\n", s.stats.Network.WifiState)
	fmt.Fprintf(&b, "wifi_ipv4|string|%s\n", s.stats.Network.WifiIPv4)
	fmt.Fprintf(&b, "wifi_ssid|string|%s\n", s.stats.Network.WifiSSID)
	fmt.Fprintf(&b, "eth_state|bool|%t\n", s.stats.Network.EthState)
	fmt.Fprintf(&b, "eth_ipv4|string|%s\n", s.stats.Network.EthIPv4)
	b.WriteByte('\n')
	return b.String()
}

package battery

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	propsIface         = "org.freedesktop.DBus.Properties"
	propsChangedSignal = "PropertiesChanged"
)

type Stats struct {
	Status   string
	Percent  int
	Estimate string // HH:MM format, time to empty or full
}

type Monitor struct {
	conn *dbus.Conn
	path dbus.ObjectPath
	last Stats
}

func New(conn *dbus.Conn) *Monitor {
	return &Monitor{conn: conn}
}

func (m *Monitor) detectBattery() error {
	obj := m.conn.Object("org.freedesktop.UPower", "/org/freedesktop/UPower")
	var devices []dbus.ObjectPath
	if err := obj.Call("org.freedesktop.UPower.EnumerateDevices", 0).Store(&devices); err != nil {
		return fmt.Errorf("enumerate devices: %w", err)
	}

	for _, dev := range devices {
		path := string(dev)
		if strings.Contains(path, "/battery_") {
			m.path = dev
			return nil
		}
	}

	return fmt.Errorf("no battery found")
}

func (m *Monitor) Read(sig *dbus.Signal) Stats {
	if sig == nil {
		if stats, ok := m.fetchStats(); ok {
			m.last = stats
		}
	} else if stats, ok := m.parseSignal(sig); ok {
		m.last = stats
	}

	return m.last
}

var stateNames = map[uint32]string{
	1: "charging",
	2: "discharging",
	3: "empty",
	4: "full",
	5: "charging",    // pending charge
	6: "discharging", // pending discharge
}

func (m *Monitor) parseSignal(sig *dbus.Signal) (Stats, bool) {
	if sig.Path != m.path || sig.Name != propsIface+"."+propsChangedSignal {
		return Stats{}, false
	}

	if len(sig.Body) < 2 {
		return Stats{}, false
	}

	props, ok := sig.Body[1].(map[string]dbus.Variant)
	if !ok {
		return Stats{}, false
	}

	stats := m.last
	changed := false

	if v, ok := props["State"]; ok {
		if state, ok := v.Value().(uint32); ok {
			if name, ok := stateNames[state]; ok {
				stats.Status = name
				changed = true
			}
		}
	}

	if v, ok := props["Percentage"]; ok {
		if pct, ok := v.Value().(float64); ok {
			stats.Percent = int(pct)
			changed = true
		}
	}

	var seconds int64
	var estimateFound bool

	if v, ok := props["TimeToEmpty"]; ok {
		seconds, _ = v.Value().(int64)
		estimateFound = true
	}
	if v, ok := props["TimeToFull"]; ok {
		if s, _ := v.Value().(int64); s > 0 {
			seconds = s
			estimateFound = true
		}
	}

	if estimateFound {
		if seconds > 0 {
			stats.Estimate = fmt.Sprintf("%02d:%02d", seconds/3600, (seconds%3600)/60)
		} else {
			stats.Estimate = ""
		}
		changed = true
	}

	return stats, changed
}

func (m *Monitor) fetchStats() (Stats, bool) {
	obj := m.conn.Object("org.freedesktop.UPower", m.path)
	var props map[string]dbus.Variant
	if err := obj.Call(propsIface+".GetAll", 0, "org.freedesktop.UPower.Device").Store(&props); err != nil {
		return Stats{}, false
	}

	var stats Stats

	if v, ok := props["State"]; ok {
		if state, ok := v.Value().(uint32); ok {
			if name, ok := stateNames[state]; ok {
				stats.Status = name
			}
		}
	}

	if v, ok := props["Percentage"]; ok {
		if pct, ok := v.Value().(float64); ok {
			stats.Percent = int(pct)
		}
	}

	var seconds int64
	if v, ok := props["TimeToEmpty"]; ok {
		seconds, _ = v.Value().(int64)
	}
	if v, ok := props["TimeToFull"]; ok {
		if s, _ := v.Value().(int64); s > 0 {
			seconds = s
		}
	}
	if seconds > 0 {
		stats.Estimate = fmt.Sprintf("%02d:%02d", seconds/3600, (seconds%3600)/60)
	}

	return stats, true
}

func (m *Monitor) Listen(ctx context.Context) (<-chan *dbus.Signal, error) {
	if err := m.detectBattery(); err != nil {
		return nil, err
	}

	matchRule := fmt.Sprintf(
		"type='signal',interface='%s',member='%s',path='%s'",
		propsIface, propsChangedSignal, m.path,
	)

	if err := m.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return nil, fmt.Errorf("failed to add match rule: %w", err)
	}

	ch := make(chan *dbus.Signal, 1)
	m.conn.Signal(ch)

	context.AfterFunc(ctx, func() {
		m.conn.RemoveSignal(ch)
	})

	return ch, nil
}

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
	info Stats
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

func (m *Monitor) Read() Stats {
	return m.info
}

var stateNames = map[uint32]string{
	1: "charging",
	2: "discharging",
	3: "empty",
	4: "full",
	5: "charging",    // pending charge
	6: "discharging", // pending discharge
}

func (m *Monitor) parseSignal(body []interface{}) bool {
	if len(body) < 2 {
		return false
	}

	props, ok := body[1].(map[string]dbus.Variant)
	if !ok {
		return false
	}

	changed := false

	if v, ok := props["State"]; ok {
		if state, ok := v.Value().(uint32); ok {
			if name, ok := stateNames[state]; ok {
				m.info.Status = name
				changed = true
			}
		}
	}

	if v, ok := props["Percentage"]; ok {
		if pct, ok := v.Value().(float64); ok {
			m.info.Percent = int(pct)
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
			m.info.Estimate = fmt.Sprintf("%02d:%02d", seconds/3600, (seconds%3600)/60)
		} else {
			m.info.Estimate = ""
		}
		changed = true
	}

	return changed
}

func (m *Monitor) fetchInitial() error {
	obj := m.conn.Object("org.freedesktop.UPower", m.path)
	var props map[string]dbus.Variant
	if err := obj.Call(propsIface+".GetAll", 0, "org.freedesktop.UPower.Device").Store(&props); err != nil {
		return err
	}

	if v, ok := props["State"]; ok {
		if state, ok := v.Value().(uint32); ok {
			if name, ok := stateNames[state]; ok {
				m.info.Status = name
			}
		}
	}

	if v, ok := props["Percentage"]; ok {
		if pct, ok := v.Value().(float64); ok {
			m.info.Percent = int(pct)
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
		m.info.Estimate = fmt.Sprintf("%02d:%02d", seconds/3600, (seconds%3600)/60)
	}

	return nil
}

func (m *Monitor) Listen(ctx context.Context) (<-chan Stats, error) {
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

	ch := make(chan Stats, 1)

	go func() {
		defer close(ch)

		// Fetch initial state before listening for changes
		if err := m.fetchInitial(); err == nil {
			ch <- m.info
		}

		signals := make(chan *dbus.Signal, 10)
		m.conn.Signal(signals)

		for {
			select {
			case <-ctx.Done():
				m.conn.RemoveSignal(signals)
				return
			case sig := <-signals:
				if sig.Path == m.path && sig.Name == propsIface+"."+propsChangedSignal {
					if m.parseSignal(sig.Body) {
						ch <- m.info
					}
				}
			}
		}
	}()

	return ch, nil
}

package battery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	sysfsPath          = "/sys/class/power_supply"
	upowerPath         = "/org/freedesktop/UPower/devices/battery_BAT0"
	upowerDeviceIface  = "org.freedesktop.UPower.Device"
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
	info Stats
}

func New(conn *dbus.Conn) (*Monitor, error) {
	m := &Monitor{conn: conn}

	info, err := m.readFromSysfs()
	if err != nil {
		return nil, err
	}
	m.info = info

	return m, nil
}

func (m *Monitor) readFromSysfs() (Stats, error) {
	batPath := filepath.Join(sysfsPath, "BAT0")

	status, err := m.readSysfsFile(filepath.Join(batPath, "status"))
	if err != nil {
		return Stats{}, fmt.Errorf("failed to read status: %w", err)
	}

	capacityStr, err := m.readSysfsFile(filepath.Join(batPath, "capacity"))
	if err != nil {
		return Stats{}, fmt.Errorf("failed to read capacity: %w", err)
	}

	var capacity int
	fmt.Sscanf(capacityStr, "%d", &capacity)

	return Stats{
		Status:   strings.ToLower(status),
		Percent:  capacity,
		Estimate: m.info.Estimate,
	}, nil
}

func (m *Monitor) readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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

func (m *Monitor) Listen() (<-chan Stats, error) {
	matchRule := fmt.Sprintf(
		"type='signal',interface='%s',member='%s',path='%s'",
		propsIface, propsChangedSignal, upowerPath,
	)

	if err := m.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return nil, fmt.Errorf("failed to add match rule: %w", err)
	}

	ch := make(chan Stats, 10)

	go func() {
		signals := make(chan *dbus.Signal, 10)
		m.conn.Signal(signals)

		ch <- m.info

		for sig := range signals {
			if sig.Path == dbus.ObjectPath(upowerPath) && sig.Name == propsIface+"."+propsChangedSignal {
				if m.parseSignal(sig.Body) {
					ch <- m.info
				}
			}
		}
	}()

	return ch, nil
}

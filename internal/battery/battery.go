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
	propsIface         = "org.freedesktop.DBus.Properties"
	propsChangedSignal = "PropertiesChanged"
)

type Stats struct {
	Status   string
	Capacity int
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
		Capacity: capacity,
	}, nil
}

func (m *Monitor) readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m *Monitor) Info() Stats {
	return m.info
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
				info, err := m.readFromSysfs()
				if err != nil {
					continue
				}

				if info != m.info {
					m.info = info
					ch <- info
				}
			}
		}
	}()

	return ch, nil
}

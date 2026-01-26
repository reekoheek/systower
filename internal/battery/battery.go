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

type BatteryInfo struct {
	Status   string
	Capacity int
}

type BatteryMonitor struct {
	conn *dbus.Conn
	info BatteryInfo
}

func NewBatteryMonitor(conn *dbus.Conn) (*BatteryMonitor, error) {
	m := &BatteryMonitor{conn: conn}

	info, err := m.readFromSysfs()
	if err != nil {
		return nil, err
	}
	m.info = info

	return m, nil
}

func (m *BatteryMonitor) readFromSysfs() (BatteryInfo, error) {
	batPath := filepath.Join(sysfsPath, "BAT0")

	status, err := m.readSysfsFile(filepath.Join(batPath, "status"))
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("failed to read status: %w", err)
	}

	capacityStr, err := m.readSysfsFile(filepath.Join(batPath, "capacity"))
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("failed to read capacity: %w", err)
	}

	var capacity int
	fmt.Sscanf(capacityStr, "%d", &capacity)

	return BatteryInfo{
		Status:   strings.ToLower(status),
		Capacity: capacity,
	}, nil
}

func (m *BatteryMonitor) readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m *BatteryMonitor) Info() BatteryInfo {
	return m.info
}

func (m *BatteryMonitor) Listen() (<-chan BatteryInfo, error) {
	matchRule := fmt.Sprintf(
		"type='signal',interface='%s',member='%s',path='%s'",
		propsIface, propsChangedSignal, upowerPath,
	)

	if err := m.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return nil, fmt.Errorf("failed to add match rule: %w", err)
	}

	ch := make(chan BatteryInfo, 10)

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

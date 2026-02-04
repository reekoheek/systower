package network

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	nmDest             = "org.freedesktop.NetworkManager"
	nmPath             = "/org/freedesktop/NetworkManager"
	nmIface            = "org.freedesktop.NetworkManager"
	nmDeviceIface      = "org.freedesktop.NetworkManager.Device"
	nmWirelessIface    = "org.freedesktop.NetworkManager.Device.Wireless"
	nmAccessPointIface = "org.freedesktop.NetworkManager.AccessPoint"
	nmIP4ConfigIface   = "org.freedesktop.NetworkManager.IP4Config"
	propsIface         = "org.freedesktop.DBus.Properties"
	propsChanged       = "PropertiesChanged"
	stateChangedSig    = "StateChanged"
)

// Device types
const (
	deviceTypeEthernet uint32 = 1
	deviceTypeWifi     uint32 = 2
)

// Device states
const (
	deviceStateActivated uint32 = 100
)

type Stats struct {
	WifiState bool
	WifiIPv4  string
	WifiSSID  string
	EthState  bool
	EthIPv4   string
}

type Monitor struct {
	conn    *dbus.Conn
	wifiDev dbus.ObjectPath
	ethDev  dbus.ObjectPath
	last    Stats
}

func New(conn *dbus.Conn) *Monitor {
	return &Monitor{conn: conn}
}

func (m *Monitor) detectDevices() error {
	obj := m.conn.Object(nmDest, nmPath)
	var devices []dbus.ObjectPath
	if err := obj.Call(nmIface+".GetDevices", 0).Store(&devices); err != nil {
		return fmt.Errorf("get devices: %w", err)
	}

	for _, devPath := range devices {
		devObj := m.conn.Object(nmDest, devPath)
		var devType dbus.Variant
		if err := devObj.Call(propsIface+".Get", 0, nmDeviceIface, "DeviceType").Store(&devType); err != nil {
			continue
		}

		dtype, ok := devType.Value().(uint32)
		if !ok {
			continue
		}

		switch dtype {
		case deviceTypeWifi:
			if m.wifiDev == "" {
				m.wifiDev = devPath
			}
		case deviceTypeEthernet:
			if m.ethDev == "" {
				m.ethDev = devPath
			}
		}
	}

	return nil
}

func (m *Monitor) Read(sig *dbus.Signal) Stats {
	if sig == nil {
		m.last = m.fetchStats()
	} else if stats, ok := m.parseSignal(sig); ok {
		m.last = stats
	}

	return m.last
}

func (m *Monitor) parseSignal(sig *dbus.Signal) (Stats, bool) {
	// Check if signal is from our devices
	if sig.Path != m.wifiDev && sig.Path != m.ethDev {
		return Stats{}, false
	}

	// Handle StateChanged signal
	if sig.Name == nmDeviceIface+"."+stateChangedSig {
		return m.fetchStats(), true
	}

	// Handle PropertiesChanged signal for IP4Config changes
	if sig.Name == propsIface+"."+propsChanged {
		if len(sig.Body) >= 2 {
			if props, ok := sig.Body[1].(map[string]dbus.Variant); ok {
				if _, hasIP4 := props["Ip4Config"]; hasIP4 {
					return m.fetchStats(), true
				}
			}
		}
	}

	return Stats{}, false
}

func (m *Monitor) fetchStats() Stats {
	var stats Stats

	if m.wifiDev != "" {
		stats.WifiState, stats.WifiIPv4, stats.WifiSSID = m.getWifiStatus()
	}
	if m.ethDev != "" {
		stats.EthState, stats.EthIPv4 = m.getEthStatus()
	}

	return stats
}

func (m *Monitor) getWifiStatus() (bool, string, string) {
	devObj := m.conn.Object(nmDest, m.wifiDev)

	// Get State
	var stateVar dbus.Variant
	if err := devObj.Call(propsIface+".Get", 0, nmDeviceIface, "State").Store(&stateVar); err != nil {
		return false, "", ""
	}

	state, ok := stateVar.Value().(uint32)
	if !ok {
		return false, "", ""
	}

	connected := state == deviceStateActivated
	if !connected {
		return false, "", ""
	}

	// Get SSID from ActiveAccessPoint
	var apVar dbus.Variant
	if err := devObj.Call(propsIface+".Get", 0, nmWirelessIface, "ActiveAccessPoint").Store(&apVar); err != nil {
		return true, m.getDeviceIP(m.wifiDev), ""
	}

	apPath, ok := apVar.Value().(dbus.ObjectPath)
	if !ok || apPath == "/" || apPath == "" {
		return true, m.getDeviceIP(m.wifiDev), ""
	}

	apObj := m.conn.Object(nmDest, apPath)
	var ssidVar dbus.Variant
	if err := apObj.Call(propsIface+".Get", 0, nmAccessPointIface, "Ssid").Store(&ssidVar); err != nil {
		return true, m.getDeviceIP(m.wifiDev), ""
	}

	ssidBytes, ok := ssidVar.Value().([]byte)
	if !ok {
		return true, m.getDeviceIP(m.wifiDev), ""
	}

	return true, m.getDeviceIP(m.wifiDev), string(ssidBytes)
}

func (m *Monitor) getEthStatus() (bool, string) {
	devObj := m.conn.Object(nmDest, m.ethDev)

	// Get State
	var stateVar dbus.Variant
	if err := devObj.Call(propsIface+".Get", 0, nmDeviceIface, "State").Store(&stateVar); err != nil {
		return false, ""
	}

	state, ok := stateVar.Value().(uint32)
	if !ok {
		return false, ""
	}

	connected := state == deviceStateActivated
	if !connected {
		return false, ""
	}

	return true, m.getDeviceIP(m.ethDev)
}

func (m *Monitor) getDeviceIP(devPath dbus.ObjectPath) string {
	devObj := m.conn.Object(nmDest, devPath)

	// Get IP4Config path
	var ip4ConfigVar dbus.Variant
	if err := devObj.Call(propsIface+".Get", 0, nmDeviceIface, "Ip4Config").Store(&ip4ConfigVar); err != nil {
		return ""
	}

	ip4ConfigPath, ok := ip4ConfigVar.Value().(dbus.ObjectPath)
	if !ok || ip4ConfigPath == "/" || ip4ConfigPath == "" {
		return ""
	}

	// Get AddressData from IP4Config
	ip4Obj := m.conn.Object(nmDest, ip4ConfigPath)
	var addrDataVar dbus.Variant
	if err := ip4Obj.Call(propsIface+".Get", 0, nmIP4ConfigIface, "AddressData").Store(&addrDataVar); err != nil {
		return ""
	}

	addrData, ok := addrDataVar.Value().([]map[string]dbus.Variant)
	if !ok || len(addrData) == 0 {
		return ""
	}

	if addrVar, ok := addrData[0]["address"]; ok {
		if addr, ok := addrVar.Value().(string); ok {
			return addr
		}
	}

	return ""
}

func (m *Monitor) Listen(ctx context.Context) (<-chan *dbus.Signal, error) {
	if err := m.detectDevices(); err != nil {
		return nil, err
	}

	// Add match rules for device state changes
	var matchRules []string

	if m.wifiDev != "" {
		matchRules = append(matchRules,
			fmt.Sprintf("type='signal',interface='%s',member='%s',path='%s'",
				nmDeviceIface, stateChangedSig, m.wifiDev),
			fmt.Sprintf("type='signal',interface='%s',member='%s',path='%s'",
				propsIface, propsChanged, m.wifiDev),
		)
	}

	if m.ethDev != "" {
		matchRules = append(matchRules,
			fmt.Sprintf("type='signal',interface='%s',member='%s',path='%s'",
				nmDeviceIface, stateChangedSig, m.ethDev),
			fmt.Sprintf("type='signal',interface='%s',member='%s',path='%s'",
				propsIface, propsChanged, m.ethDev),
		)
	}

	for _, rule := range matchRules {
		if err := m.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Err; err != nil {
			return nil, fmt.Errorf("add match rule: %w", err)
		}
	}

	ch := make(chan *dbus.Signal, 1)
	m.conn.Signal(ch)

	context.AfterFunc(ctx, func() {
		m.conn.RemoveSignal(ch)
	})

	return ch, nil
}

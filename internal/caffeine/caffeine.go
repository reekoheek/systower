package caffeine

import (
	"fmt"
	"path/filepath"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/sys"
)

const (
	dbusPath      = "/reekoheek/Caffeine"
	dbusInterface = "reekoheek.Caffeine"
	dbusSignal    = "StatusChanged"
)

type Caffeine struct {
	conn    *dbus.Conn
	adapter Adapter
}

func New(conn *dbus.Conn, adapter Adapter) *Caffeine {
	return &Caffeine{
		conn:    conn,
		adapter: adapter,
	}
}

func (c *Caffeine) Read() string {
	return c.adapter.Status()
}

func (c *Caffeine) emitSignal() error {
	return c.conn.Emit(dbus.ObjectPath(dbusPath), dbusInterface+"."+dbusSignal)
}

func (c *Caffeine) On() {
	c.adapter.On()
	c.emitSignal()
}

func (c *Caffeine) Off() {
	c.adapter.Off()
	c.emitSignal()
}

func (c *Caffeine) Toggle() {
	if c.Read() == "on" {
		c.Off()
	} else {
		c.On()
	}
}

func (c *Caffeine) Listen() (<-chan string, error) {
	matchRule := fmt.Sprintf("type='signal',interface='%s',member='%s'", dbusInterface, dbusSignal)
	if err := c.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return nil, fmt.Errorf("failed to add match rule: %w", err)
	}

	signals := make(chan *dbus.Signal, 10)
	c.conn.Signal(signals)

	statusCh := make(chan string, 10)

	go func() {
		statusCh <- c.Read()
		for range signals {
			statusCh <- c.Read()
		}
		close(statusCh)
	}()

	return statusCh, nil
}

func DetectAdapter() Adapter {
	if !sys.IsWayland() {
		return NewX11Adapter()
	}
	lockfile := filepath.Join(sys.GetRuntimeDir(), "swayidle.lock")
	return NewWaylandAdapter(lockfile)
}

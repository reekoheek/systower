package caffeine

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	dbusPath      = "/reekoheek/Caffeine"
	dbusInterface = "reekoheek.Caffeine"
	dbusSignal    = "StatusChanged"
)

type Stats struct {
	Active bool
}

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

func (c *Caffeine) Status() Stats {
	return Stats{Active: c.adapter.Status() == "on"}
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
	if c.Status().Active {
		c.Off()
	} else {
		c.On()
	}
}

func (c *Caffeine) Listen() (<-chan Stats, error) {
	matchRule := fmt.Sprintf("type='signal',interface='%s',member='%s'", dbusInterface, dbusSignal)
	if err := c.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return nil, fmt.Errorf("failed to add match rule: %w", err)
	}

	signals := make(chan *dbus.Signal, 10)
	c.conn.Signal(signals)

	statusCh := make(chan Stats, 10)

	go func() {
		statusCh <- c.Status()
		for range signals {
			statusCh <- c.Status()
		}
		close(statusCh)
	}()

	return statusCh, nil
}

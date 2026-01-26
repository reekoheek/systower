package caffeine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/godbus/dbus/v5"
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

func New() (*Caffeine, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	var adapter Adapter
	isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"

	if isWayland {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = "/tmp"
		}
		pidfile := filepath.Join(runtimeDir, "swayidle.pid")
		adapter = NewWaylandAdapter(pidfile)
	} else {
		adapter = NewX11Adapter()
	}

	return &Caffeine{
		conn:    conn,
		adapter: adapter,
	}, nil
}

func (c *Caffeine) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Caffeine) Status() string {
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
	if c.Status() == "on" {
		c.Off()
	} else {
		c.On()
	}
}

func (c *Caffeine) Listen() error {
	matchRule := fmt.Sprintf("type='signal',interface='%s',member='%s'", dbusInterface, dbusSignal)
	if err := c.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return fmt.Errorf("failed to add match rule: %w", err)
	}

	signals := make(chan *dbus.Signal, 10)
	c.conn.Signal(signals)

	fmt.Printf("status|string|%s\n\n", c.Status())

	for range signals {
		fmt.Printf("status|string|%s\n\n", c.Status())
	}

	return nil
}

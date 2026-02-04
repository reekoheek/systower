package caffeine

import (
	"context"
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

func (c *Caffeine) Listen(ctx context.Context) (<-chan string, error) {
	matchRule := fmt.Sprintf("type='signal',interface='%s',member='%s'", dbusInterface, dbusSignal)
	if err := c.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return nil, fmt.Errorf("failed to add match rule: %w", err)
	}

	ch := make(chan string, 1)
	signals := make(chan *dbus.Signal, 10)
	c.conn.Signal(signals)

	go func() {
		defer close(ch)

		ch <- c.Read()
		for {
			select {
			case <-ctx.Done():
				c.conn.RemoveSignal(signals)
				return
			case <-signals:
				ch <- c.Read()
			}
		}
	}()

	return ch, nil
}

func DetectAdapter() Adapter {
	if !isWayland() {
		return NewX11Adapter()
	}
	lockfile := filepath.Join(getRuntimeDir(), "swayidle.lock")
	return NewWaylandAdapter(lockfile)
}

func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func getRuntimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	return "/tmp"
}

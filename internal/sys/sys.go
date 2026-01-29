package sys

import (
	"os"

	"github.com/godbus/dbus/v5"
)

func IsWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func GetRuntimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	return "/tmp"
}

type Sys struct {
	conn *dbus.Conn
}

func New(conn *dbus.Conn) *Sys {
	return &Sys{conn: conn}
}

func (s *Sys) Poweroff() {
	s.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1").
		Call("org.freedesktop.login1.Manager.PowerOff", 0, false)
}

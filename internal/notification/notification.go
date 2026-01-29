package notification

import "github.com/godbus/dbus/v5"

type Notification struct {
	conn *dbus.Conn
}

func New(conn *dbus.Conn) *Notification {
	return &Notification{conn: conn}
}

func (n *Notification) Send(summary, body string) {
	n.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").
		Call("org.freedesktop.Notifications.Notify", 0,
			"systower",
			uint32(0),
			"",
			summary,
			body,
			[]string{},
			map[string]dbus.Variant{},
			int32(-1),
		)
}

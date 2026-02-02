package notification

import "github.com/godbus/dbus/v5"

type Notification struct {
	conn  *dbus.Conn
	title string
}

func New(conn *dbus.Conn, title string) *Notification {
	return &Notification{conn: conn, title: title}
}

func (n *Notification) Send(body string) error {
	return n.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").
		Call("org.freedesktop.Notifications.Notify", 0,
			"systower",
			uint32(0),
			"",
			n.title,
			body,
			[]string{},
			map[string]dbus.Variant{},
			int32(-1),
		).Err
}

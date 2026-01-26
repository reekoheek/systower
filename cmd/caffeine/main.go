package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/caffeine/internal/battery"
	"github.com/reekoheek/caffeine/internal/caffeine"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	c := caffeine.New(conn, createAdapter())

	switch os.Args[1] {
	case "listen":
		sysConn, err := dbus.SystemBus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer sysConn.Close()

		batMon, err := battery.NewBatteryMonitor(sysConn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		batCh, err := batMon.Listen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		go func() {
			for info := range batCh {
				if info.Status != "charging" {
					continue
				}

				if c.Status() == "on" && info.Capacity < 15 {
					c.Off()
					sendNotification(conn, "Caffeine", "Low battery, disable caffeine")
				}

				if info.Capacity < 5 {
					sendNotification(conn, "Caffeine", "Battery almost drained, have a nice sleep")
					time.Sleep(5 * time.Second)
					poweroff(sysConn)
				}
			}
		}()

		if err := c.Listen(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "on":
		c.On()
	case "off":
		c.Off()
	case "toggle":
		c.Toggle()
	case "status":
		fmt.Println(c.Status())
	default:
		os.Exit(1)
	}
}

func createAdapter() caffeine.Adapter {
	isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
	if !isWayland {
		return caffeine.NewX11Adapter()
	}

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	pidfile := filepath.Join(runtimeDir, "swayidle.pid")
	return caffeine.NewWaylandAdapter(pidfile)
}

func sendNotification(conn *dbus.Conn, summary, body string) {
	conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").
		Call("org.freedesktop.Notifications.Notify", 0,
			"caffeine",
			uint32(0),
			"",
			summary,
			body,
			[]string{},
			map[string]dbus.Variant{},
			int32(-1),
		)
}

func poweroff(conn *dbus.Conn) {
	conn.Object("org.freedesktop.login1", "/org/freedesktop/login1").
		Call("org.freedesktop.login1.Manager.PowerOff", 0, false)
}

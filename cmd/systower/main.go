package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/sys"
	"github.com/reekoheek/systower/internal/systower"
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

	caff := caffeine.New(conn, createAdapter())

	switch os.Args[1] {
	case "listen":
		sysConn, err := dbus.SystemBus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer sysConn.Close()

		notif := notification.New(conn, "Systower")
		bat, err := battery.New(sysConn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		sysMgr := sys.New(sysConn)

		systower.New(caff, notif, bat, sysMgr).Daemon()
	case "on":
		caff.On()
	case "off":
		caff.Off()
	case "toggle":
		caff.Toggle()
	case "status":
		fmt.Println(caff.Status())
	default:
		os.Exit(1)
	}
}

func createAdapter() caffeine.Adapter {
	if !sys.IsWayland() {
		return caffeine.NewX11Adapter()
	}

	lockfile := filepath.Join(sys.GetRuntimeDir(), "swayidle.lock")
	return caffeine.NewWaylandAdapter(lockfile)
}

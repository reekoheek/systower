package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/poller"
	"github.com/reekoheek/systower/internal/sys"
	"github.com/reekoheek/systower/internal/systower"
)

const usage = "usage: systower <watch|caffeine <on|off|toggle|status>>"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "watch":
		runWatch()
	case "caffeine":
		runCaffeine()
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}

func runWatch() {
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	sysConn, err := dbus.SystemBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer sysConn.Close()

	caff := caffeine.New(conn, createAdapter())
	notif := notification.New(conn, "Systower")
	bat, err := battery.New(sysConn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	sysMgr := sys.New(sysConn)
	poll := poller.New(1*time.Second, 1*time.Second, 60*time.Second)

	systower.New(caff, notif, bat, sysMgr, poll).Watch()
}

func runCaffeine() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	caff := caffeine.New(conn, createAdapter())

	switch os.Args[2] {
	case "on":
		caff.On()
	case "off":
		caff.Off()
	case "toggle":
		caff.Toggle()
	case "status":
		fmt.Println(caff.Read())
	default:
		fmt.Fprintln(os.Stderr, usage)
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

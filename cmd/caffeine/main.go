package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/godbus/dbus/v5"
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

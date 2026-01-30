package main

import (
	"fmt"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/caffeine"
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
	s, err := systower.New(systower.Intervals{
		Clock:    5 * time.Second,
		CPU:      5 * time.Second,
		Mem:      5 * time.Second,
		Storage:  60 * time.Second,
		Debounce: 100 * time.Millisecond,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	s.Watch()
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

	caff := caffeine.New(conn, caffeine.DetectAdapter())

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

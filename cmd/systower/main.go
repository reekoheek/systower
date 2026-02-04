package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/systower"
)

const usage = "usage: systower <watch|caffeine <on|off|toggle|status>>"

func main() {
	// runtime.GOMAXPROCS(1)

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
	s, err := systower.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := s.Watch(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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

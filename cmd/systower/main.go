package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/systower"
)

const usage = "usage: systower <watch|caffeine <on|off|toggle|status>>"

func main() {
	runtime.GOMAXPROCS(1)

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
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	clockInterval := fs.Duration("clock", 3*time.Second, "clock polling interval")
	cpuInterval := fs.Duration("cpu", 3*time.Second, "cpu polling interval")
	memInterval := fs.Duration("mem", 3*time.Second, "memory polling interval")
	storageInterval := fs.Duration("storage", 300*time.Second, "storage polling interval")
	fs.Parse(os.Args[2:])

	s, err := systower.New(systower.Intervals{
		Clock:   *clockInterval,
		CPU:     *cpuInterval,
		Mem:     *memInterval,
		Storage: *storageInterval,
	})
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
